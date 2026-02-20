package extractor

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lia/liacheckscanner_go/internal/models"
)

// rdapCache manages simple on-disk cache for RDAP query results.
type rdapCache struct {
	Entries map[string]models.RDAPCacheEntry `json:"entries"`
	Path    string                           `json:"-"`
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

func (e *Extractor) applyCache(ip string, data *models.ScannerData, c *rdapCache) bool {
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

func (e *Extractor) updateCache(ip string, data *models.ScannerData, c *rdapCache) {
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

// enrichData enriches IP data with scanner metadata and API lookups.
func (e *Extractor) enrichData(ips []string) ([]models.ScannerData, error) {
	e.logger.Info("Extractor", "Enrichissement des donnees...")

	ipToScanner := e.mapIPsToScanners(ips)

	var scannerData []models.ScannerData
	now := time.Now()

	for i, ip := range ips {
		scannerInfo := ipToScanner[ip]

		data := models.ScannerData{
			ID:          fmt.Sprintf("scanner_%d", i+1),
			IPOrCIDR:    ip,
			ScannerName: scannerInfo.Name,
			ScannerType: scannerInfo.Type,
			SourceFile:  scannerInfo.SourceFile,
			LastSeen:    now,
			FirstSeen:   now,
			ExportDate:  now,
			CreatedAt:   now,
			UpdatedAt:   now,
			Tags:        []string{"extracted", scannerInfo.Name},
			RiskLevel:   "unknown",
		}

		if err := e.enrichWithAPI(&data); err != nil {
			e.logger.Warning("Extractor", fmt.Sprintf("Erreur lors de l'enrichissement de %s: %v", ip, err))
		}

		scannerData = append(scannerData, data)
	}

	e.logger.Info("Extractor", fmt.Sprintf("%d enregistrements enrichis", len(scannerData)))
	return scannerData, nil
}

// enrichWithAPI enriches data with RDAP and public geolocation APIs.
func (e *Extractor) enrichWithAPI(data *models.ScannerData) error {
	// Use the rate limiter for throttling instead of a raw sleep.
	if e.rateLimiter != nil {
		e.rateLimiter.Wait()
	}

	cache := e.loadRDAPCache()
	if e.applyCache(data.IPOrCIDR, data, cache) {
		return nil
	}

	_ = e.performRDAPFull(data.IPOrCIDR, data)

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

	e.updateCache(data.IPOrCIDR, data, cache)
	cache.save()
	return nil
}

// performRDAPFull populates RDAP and contact fields on data from RDAP registries.
func (e *Extractor) performRDAPFull(ip string, data *models.ScannerData) error {
	all := map[string]string{
		"arin":    "https://rdap.arin.net/registry/ip/",
		"ripe":    "https://rdap.ripe.net/ip/",
		"apnic":   "https://rdap.apnic.net/ip/",
		"lacnic":  "https://rdap.lacnic.net/rdap/ip/",
		"afrinic": "https://rdap.afrinic.net/rdap/ip/",
	}
	var endpoints []string
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
	client := &http.Client{Timeout: 12 * time.Second}
	for _, base := range endpoints {
		url := base + ip
		resp, err := client.Get(url)
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
	return nil
}

// performRDAPDetail returns a few common RDAP fields (name/handle/cidr/registry).
func (e *Extractor) performRDAPDetail(ip string) (string, string, string, string) {
	endpoints := []string{
		"https://rdap.arin.net/registry/ip/",
		"https://rdap.ripe.net/ip/",
		"https://rdap.apnic.net/ip/",
		"https://rdap.lacnic.net/rdap/ip/",
		"https://rdap.afrinic.net/rdap/ip/",
	}
	client := &http.Client{Timeout: 12 * time.Second}
	for _, base := range endpoints {
		url := base + ip
		resp, err := client.Get(url)
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
		name := ""
		handle := ""
		cidr := ""
		registry := ""
		if v, ok := m["name"].(string); ok {
			name = v
		}
		if v, ok := m["handle"].(string); ok {
			handle = v
		}
		if v, ok := m["port43"].(string); ok && registry == "" {
			registry = v
		}
		if v, ok := m["objectClassName"].(string); ok && registry == "" {
			registry = v
		}
		if network, ok := m["network"].(map[string]interface{}); ok {
			if v, ok := network["cidr0_cidrs"].([]interface{}); ok && len(v) > 0 {
				if first, ok := v[0].(map[string]interface{}); ok {
					start, _ := first["v4prefix"].(string)
					length := fmt.Sprintf("%v", first["length"])
					if start != "" && length != "<nil>" {
						cidr = fmt.Sprintf("%s/%s", start, length)
					}
				}
			}
		}
		if name != "" || handle != "" || cidr != "" || registry != "" {
			return name, handle, cidr, registry
		}
	}
	return "", "", "", ""
}

// performGeoLookupExtended queries ip-api.com for country/ISP/AS/reverse info.
func (e *Extractor) performGeoLookupExtended(ip string) (string, string, string, string, string) {
	client := &http.Client{Timeout: 10 * time.Second}
	url := "http://ip-api.com/json/" + ip + "?fields=status,country,countryCode,isp,as,reverse"
	resp, err := client.Get(url)
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
	client := &http.Client{Timeout: 10 * time.Second}
	url := "http://ip-api.com/json/" + ip + "?fields=status,continent,continentCode,country,countryCode"
	resp, err := client.Get(url)
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
		return tracker
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(tracker); err != nil {
		return tracker
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
