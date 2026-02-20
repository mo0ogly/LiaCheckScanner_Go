package extractor

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lia/liacheckscanner_go/internal/models"
)

// cacheAccessor abstracts cache read/write so the same enrichment logic
// works for both single-threaded (rdapCache) and concurrent (safeRDAPCache) use.
type cacheAccessor interface {
	applyCache(ip string, data *models.ScannerData) bool
	updateCache(ip string, data *models.ScannerData)
}

// rdapCache manages simple on-disk cache for RDAP query results.
type rdapCache struct {
	Entries map[string]models.RDAPCacheEntry `json:"entries"`
	Path    string                           `json:"-"`
}

func (c *rdapCache) applyCache(ip string, data *models.ScannerData) bool {
	entry, ok := c.Entries[ip]
	if !ok {
		return false
	}
	data.RDAPName = entry.RDAPName
	data.RDAPHandle = entry.RDAPHandle
	data.RDAPCIDR = entry.RDAPCIDR
	data.Registry = entry.Registry
	data.StartAddress = entry.StartAddress
	data.EndAddress = entry.EndAddress
	data.IPVersion = entry.IPVersion
	data.RDAPType = entry.RDAPType
	data.ParentHandle = entry.ParentHandle
	data.EventRegistration = entry.EventRegistration
	data.EventLastChanged = entry.EventLastChanged
	data.ASN = entry.ASN
	data.ASName = entry.ASName
	data.ReverseDNS = entry.ReverseDNS
	data.CountryCode = entry.CountryCode
	data.CountryName = entry.CountryName
	data.ISP = entry.ISP
	data.Organization = entry.Organization
	data.AbuseEmail = entry.AbuseEmail
	data.TechEmail = entry.TechEmail
	return true
}

func (c *rdapCache) updateCache(ip string, data *models.ScannerData) {
	c.Entries[ip] = models.RDAPCacheEntry{
		RDAPName:          data.RDAPName,
		RDAPHandle:        data.RDAPHandle,
		RDAPCIDR:          data.RDAPCIDR,
		Registry:          data.Registry,
		StartAddress:      data.StartAddress,
		EndAddress:        data.EndAddress,
		IPVersion:         data.IPVersion,
		RDAPType:          data.RDAPType,
		ParentHandle:      data.ParentHandle,
		EventRegistration: data.EventRegistration,
		EventLastChanged:  data.EventLastChanged,
		ASN:               data.ASN,
		ASName:            data.ASName,
		ReverseDNS:        data.ReverseDNS,
		CountryCode:       data.CountryCode,
		CountryName:       data.CountryName,
		ISP:               data.ISP,
		Organization:      data.Organization,
		AbuseEmail:        data.AbuseEmail,
		TechEmail:         data.TechEmail,
		CachedAt:          time.Now().Format(time.RFC3339),
	}
}

// safeRDAPCache wraps rdapCache with a mutex for concurrent access.
type safeRDAPCache struct {
	mu    sync.Mutex
	cache *rdapCache
}

func newSafeRDAPCache(c *rdapCache) *safeRDAPCache {
	return &safeRDAPCache{cache: c}
}

func (sc *safeRDAPCache) applyCache(ip string, data *models.ScannerData) bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.cache.applyCache(ip, data)
}

func (sc *safeRDAPCache) updateCache(ip string, data *models.ScannerData) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.cache.updateCache(ip, data)
}

func (sc *safeRDAPCache) save() {
	sc.cache.save()
}

// cacheTTL returns the configured cache TTL as a time.Duration.
// If CacheTTLHours is 0 or negative, it defaults to 168 hours (7 days).
func (e *Extractor) cacheTTL() time.Duration {
	ttl := e.config.CacheTTLHours
	if ttl <= 0 {
		ttl = 168 // default: 7 days
	}
	return time.Duration(ttl) * time.Hour
}

func (e *Extractor) loadRDAPCache() *rdapCache {
	cachePath := filepath.Join("build", "data", "rdap_cache.json")
	_ = os.MkdirAll(filepath.Dir(cachePath), 0755)
	c := &rdapCache{Entries: map[string]models.RDAPCacheEntry{}, Path: cachePath}
	f, err := os.Open(cachePath)
	if err != nil {
		return c
	}
	defer f.Close()
	_ = json.NewDecoder(f).Decode(&c)

	// Evict entries older than the configured TTL.
	ttl := e.cacheTTL()
	now := time.Now()
	for ip, entry := range c.Entries {
		if cachedAt, err := time.Parse(time.RFC3339, entry.CachedAt); err == nil {
			if now.Sub(cachedAt) > ttl {
				delete(c.Entries, ip)
			}
		}
	}

	return c
}

// CleanExpiredCache removes all cache entries older than the configured TTL
// and persists the cleaned cache to disk.
func (e *Extractor) CleanExpiredCache() {
	cache := e.loadRDAPCache() // loadRDAPCache already evicts expired entries
	cache.save()
	e.logger.Info("Extractor", fmt.Sprintf("Cache cleaned: %d entries remaining", len(cache.Entries)))
}

func (c *rdapCache) save() {
	f, err := os.Create(c.Path)
	if err != nil {
		return
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	_ = enc.Encode(c)
}


// enrichJob represents a single IP enrichment task for the worker pool.
type enrichJob struct {
	index       int
	ip          string
	scannerInfo ScannerInfo
}

// enrichData enriches IP data with scanner metadata and API lookups.
// When config.Parallelism > 1, it uses a worker pool for concurrent enrichment.
func (e *Extractor) enrichData(ips []string) ([]models.ScannerData, error) {
	e.logger.Info("Extractor", "Enrichissement des donnees...")

	ipToScanner := e.mapIPsToScanners(ips)

	// Load cache once for the entire enrichment batch.
	cache := e.loadRDAPCache()
	safeCache := newSafeRDAPCache(cache)

	workers := e.config.Parallelism
	if workers <= 0 {
		workers = 1
	}

	now := time.Now()
	scannerData := make([]models.ScannerData, len(ips))

	e.logger.Info("Extractor", fmt.Sprintf("Enrichissement avec %d worker(s) pour %d IPs", workers, len(ips)))

	if workers == 1 {
		// Sequential path (backward compatible).
		for i, ip := range ips {
			scannerInfo := ipToScanner[ip]
			scannerData[i] = e.buildRecord(i, ip, scannerInfo, now)
			if err := e.enrichUsingCache(&scannerData[i], safeCache); err != nil {
				e.logger.Warning("Extractor", fmt.Sprintf("Erreur lors de l'enrichissement de %s: %v", ip, err))
			}
		}
	} else {
		// Parallel path with worker pool.
		jobs := make(chan enrichJob, len(ips))
		var wg sync.WaitGroup

		// Pre-populate the result slice with base records.
		for i, ip := range ips {
			scannerInfo := ipToScanner[ip]
			scannerData[i] = e.buildRecord(i, ip, scannerInfo, now)
		}

		// Start workers.
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for job := range jobs {
					if err := e.enrichUsingCache(&scannerData[job.index], safeCache); err != nil {
						e.logger.Warning("Extractor", fmt.Sprintf("Erreur lors de l'enrichissement de %s: %v", job.ip, err))
					}
				}
			}()
		}

		// Send jobs.
		for i, ip := range ips {
			jobs <- enrichJob{index: i, ip: ip, scannerInfo: ipToScanner[ip]}
		}
		close(jobs)

		wg.Wait()
	}

	// Persist cache once after processing all IPs.
	safeCache.save()

	e.logger.Info("Extractor", fmt.Sprintf("%d enregistrements enrichis", len(scannerData)))
	return scannerData, nil
}

// buildRecord creates a base ScannerData record for the given IP.
func (e *Extractor) buildRecord(i int, ip string, info ScannerInfo, now time.Time) models.ScannerData {
	return models.ScannerData{
		ID:          fmt.Sprintf("scanner_%d", i+1),
		IPOrCIDR:    ip,
		ScannerName: info.Name,
		ScannerType: info.Type,
		SourceFile:  info.SourceFile,
		LastSeen:    now,
		FirstSeen:   now,
		ExportDate:  now,
		CreatedAt:   now,
		UpdatedAt:   now,
		Tags:        []string{"extracted", info.Name},
		RiskLevel:   "unknown",
	}
}

// enrichUsingCache enriches a single ScannerData record via RDAP + geo APIs,
// using the provided cacheAccessor (either rdapCache or safeRDAPCache).
func (e *Extractor) enrichUsingCache(data *models.ScannerData, ca cacheAccessor) error {
	if e.rateLimiter != nil {
		e.rateLimiter.Wait()
	}

	if ca.applyCache(data.IPOrCIDR, data) {
		return nil
	}

	if err := e.performRDAPFull(data.IPOrCIDR, data); err != nil {
		e.logger.Warning("Extractor", fmt.Sprintf("RDAP lookup failed for %s: %v", data.IPOrCIDR, err))
	}

	cc, country, isp, asStr, reverse := e.performGeoLookupExtended(data.IPOrCIDR)
	if cc != "" {
		data.CountryCode = cc
		data.CountryName = country
	}
	if isp != "" {
		data.ISP = isp
	}
	if asStr != "" {
		data.ASN = asStr
		if parts := strings.SplitN(asStr, " ", 2); len(parts) == 2 {
			data.ASName = parts[1]
		}
	}
	if reverse != "" {
		data.ReverseDNS = reverse
		if data.Domain == "" {
			data.Domain = reverse
		}
	}

	if data.Domain == "" {
		if hostnames, err := net.LookupAddr(data.IPOrCIDR); err == nil && len(hostnames) > 0 {
			data.Domain = strings.TrimSuffix(hostnames[0], ".")
			if data.ReverseDNS == "" {
				data.ReverseDNS = data.Domain
			}
		}
	}

	ca.updateCache(data.IPOrCIDR, data)
	return nil
}

// enrichWithAPI enriches data with RDAP and public geolocation APIs.
// It loads and persists the cache per call (use enrichUsingCache for batch operations).
func (e *Extractor) enrichWithAPI(data *models.ScannerData) error {
	cache := e.loadRDAPCache()
	err := e.enrichUsingCache(data, cache)
	cache.save()
	return err
}

// performRDAPFull populates RDAP and contact fields on data from RDAP registries.
func (e *Extractor) performRDAPFull(ip string, data *models.ScannerData) error {
	var endpoints []string
	if len(e.rdapEndpoints) > 0 {
		endpoints = e.rdapEndpoints
	} else {
		all := map[string]string{
			"arin":    "https://rdap.arin.net/registry/ip/",
			"ripe":    "https://rdap.ripe.net/ip/",
			"apnic":   "https://rdap.apnic.net/ip/",
			"lacnic":  "https://rdap.lacnic.net/rdap/ip/",
			"afrinic": "https://rdap.afrinic.net/rdap/ip/",
		}
		if len(e.config.Registries) > 0 {
			for _, k := range e.config.Registries {
				if url, ok := all[k]; ok {
					endpoints = append(endpoints, url)
				}
			}
		}
		if len(endpoints) == 0 {
			endpoints = []string{all["arin"], all["ripe"], all["apnic"], all["lacnic"], all["afrinic"]}
		}
	}
	for _, base := range endpoints {
		rdapURL := base + ip
		resp, err := e.httpGetWithRetry(rdapURL)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
			continue
		}
		var m map[string]interface{}
		if err := json.Unmarshal(body, &m); err != nil {
			continue
		}
		if v, ok := m["name"].(string); ok && v != "" {
			data.RDAPName = v
			if data.Organization == "" {
				data.Organization = v
			}
		}
		if v, ok := m["handle"].(string); ok {
			data.RDAPHandle = v
		}
		if v, ok := m["port43"].(string); ok && data.Registry == "" {
			data.Registry = v
		}
		if v, ok := m["objectClassName"].(string); ok && data.Registry == "" {
			data.Registry = v
		}
		if v, ok := m["startAddress"].(string); ok {
			data.StartAddress = v
		}
		if v, ok := m["endAddress"].(string); ok {
			data.EndAddress = v
		}
		if v, ok := m["ipVersion"].(string); ok {
			data.IPVersion = v
		}
		if v, ok := m["type"].(string); ok {
			data.RDAPType = v
		}
		if v, ok := m["parentHandle"].(string); ok {
			data.ParentHandle = v
		}
		if ev, ok := m["events"].([]interface{}); ok {
			for _, eraw := range ev {
				if em, ok := eraw.(map[string]interface{}); ok {
					action, _ := em["eventAction"].(string)
					date, _ := em["eventDate"].(string)
					if action == "registration" && data.EventRegistration == "" {
						data.EventRegistration = date
					}
					if action == "last changed" && data.EventLastChanged == "" {
						data.EventLastChanged = date
					}
				}
			}
		}
		if network, ok := m["network"].(map[string]interface{}); ok {
			if v, ok := network["cidr0_cidrs"].([]interface{}); ok && len(v) > 0 {
				if first, ok := v[0].(map[string]interface{}); ok {
					start, _ := first["v4prefix"].(string)
					length := fmt.Sprintf("%v", first["length"])
					if start != "" && length != "<nil>" {
						data.RDAPCIDR = fmt.Sprintf("%s/%s", start, length)
					}
				}
			}
		}
		if ents, ok := m["entities"].([]interface{}); ok {
			for _, eraw := range ents {
				em, ok := eraw.(map[string]interface{})
				if !ok {
					continue
				}
				roles := map[string]bool{}
				if rs, ok := em["roles"].([]interface{}); ok {
					for _, r := range rs {
						if s, ok := r.(string); ok {
							roles[strings.ToLower(s)] = true
						}
					}
				}
				if vcard, ok := em["vcardArray"].([]interface{}); ok && len(vcard) > 1 {
					if arr, ok := vcard[1].([]interface{}); ok {
						for _, fld := range arr {
							if pair, ok := fld.([]interface{}); ok && len(pair) >= 3 {
								key, _ := pair[0].(string)
								if key == "email" {
									val, _ := pair[3].(string)
									if roles["abuse"] && data.AbuseEmail == "" {
										data.AbuseEmail = val
									}
									if (roles["technical"] || roles["tech"]) && data.TechEmail == "" {
										data.TechEmail = val
									}
								}
								if key == "fn" && data.RDAPName == "" {
									val, _ := pair[3].(string)
									if val != "" {
										data.RDAPName = val
										if data.Organization == "" {
											data.Organization = val
										}
									}
								}
							}
						}
					}
				}
			}
		}
		return nil
	}
	return fmt.Errorf("no RDAP registry responded for %s", ip)
}


// performGeoLookupExtended queries ip-api.com for country/ISP/AS/reverse info.
func (e *Extractor) performGeoLookupExtended(ip string) (string, string, string, string, string) {
	base := e.geoBaseURL
	if base == "" {
		base = "http://ip-api.com/json/"
	}
	geoURL := base + ip + "?fields=status,country,countryCode,isp,as,reverse"
	resp, err := e.httpGetWithRetry(geoURL)
	if err != nil {
		return "", "", "", "", ""
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", "", "", ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		return "", "", "", "", ""
	}
	if st, _ := m["status"].(string); st != "success" {
		return "", "", "", "", ""
	}
	cc, _ := m["countryCode"].(string)
	country, _ := m["country"].(string)
	isp, _ := m["isp"].(string)
	asStr, _ := m["as"].(string)
	rev, _ := m["reverse"].(string)
	return cc, country, isp, asStr, rev
}

// GeoLookupContinent returns the continent, continent code, country, and country code for the given IP.
func (e *Extractor) GeoLookupContinent(ip string) (string, string, string, string, error) {
	base := e.geoBaseURL
	if base == "" {
		base = "http://ip-api.com/json/"
	}
	geoURL := base + ip + "?fields=status,continent,continentCode,country,countryCode"
	resp, err := e.httpGetWithRetry(geoURL)
	if err != nil {
		return "", "", "", "", fmt.Errorf("geo lookup request for %s: %w", ip, err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", "", "", fmt.Errorf("geo http %d", resp.StatusCode)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		return "", "", "", "", fmt.Errorf("unmarshaling geo response for %s: %w", ip, err)
	}
	if st, _ := m["status"].(string); st != "success" {
		return "", "", "", "", fmt.Errorf("geo status %s", st)
	}
	continent, _ := m["continent"].(string)
	continentCode, _ := m["continentCode"].(string)
	country, _ := m["country"].(string)
	countryCode, _ := m["countryCode"].(string)
	return continent, continentCode, country, countryCode, nil
}

// LoadProgressTracker loads the RDAP enrichment progress tracker from disk.
func (e *Extractor) LoadProgressTracker() *models.RDAPProgressTracker {
	progressPath := filepath.Join("build", "data", "rdap_progress.json")
	tracker := &models.RDAPProgressTracker{
		ProcessedIPs: []string{},
	}

	file, err := os.Open(progressPath)
	if err != nil {
		tracker.ProcessedIPSet = map[string]struct{}{}
		return tracker
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(tracker); err != nil {
		tracker.ProcessedIPSet = map[string]struct{}{}
		return tracker
	}

	// Build the set from the slice for O(1) lookups.
	tracker.ProcessedIPSet = make(map[string]struct{}, len(tracker.ProcessedIPs))
	for _, ip := range tracker.ProcessedIPs {
		tracker.ProcessedIPSet[ip] = struct{}{}
	}

	return tracker
}

// SaveProgressTracker persists the RDAP enrichment progress tracker to disk.
func (e *Extractor) SaveProgressTracker(tracker *models.RDAPProgressTracker) error {
	progressPath := filepath.Join("build", "data", "rdap_progress.json")
	_ = os.MkdirAll(filepath.Dir(progressPath), 0755)

	tracker.LastUpdatedAt = time.Now().Format(time.RFC3339)

	file, err := os.Create(progressPath)
	if err != nil {
		return fmt.Errorf("creating progress file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(tracker); err != nil {
		return fmt.Errorf("encoding progress tracker: %w", err)
	}
	return nil
}

// IsIPProcessed reports whether the given IP has already been processed.
func (e *Extractor) IsIPProcessed(ip string, tracker *models.RDAPProgressTracker) bool {
	// Use the set for O(1) lookup if available.
	if tracker.ProcessedIPSet != nil {
		_, ok := tracker.ProcessedIPSet[ip]
		return ok
	}
	// Fallback to linear scan for backward compatibility.
	for _, processed := range tracker.ProcessedIPs {
		if processed == ip {
			return true
		}
	}
	return false
}

// ClearProgressTracker removes the RDAP enrichment progress file from disk.
func (e *Extractor) ClearProgressTracker() error {
	progressPath := filepath.Join("build", "data", "rdap_progress.json")
	return os.Remove(progressPath)
}
