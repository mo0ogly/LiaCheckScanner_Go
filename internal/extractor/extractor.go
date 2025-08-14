package extractor

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/lia/liacheckscanner_go/internal/logger"
	"github.com/lia/liacheckscanner_go/internal/models"
)

// Extractor gère l'extraction et l'enrichissement des données
type Extractor struct {
	logger    *logger.Logger
	config    models.DatabaseConfig
	apiClient *http.Client
}

// NewExtractor crée un nouvel extracteur
func NewExtractor(config models.DatabaseConfig, logger *logger.Logger) *Extractor {
	return &Extractor{
		logger: logger,
		config: config,
		apiClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExtractData extrait les données depuis le repository configuré
func (e *Extractor) ExtractData() ([]models.ScannerData, error) {
	e.logger.Info("Extractor", "🚀 Début de l'extraction des données")

	// FORCER l'utilisation du bon repository
	repoURL := "https://github.com/MDMCK10/internet-scanners"
	localPath := "/home/fpizzi/confluence_vscode/internet-scanners"

	e.logger.Info("Extractor", "📥 Clonage/mise à jour du repository...")
	e.logger.Info("Extractor", "🔧 Configuration forcée - Repository: "+repoURL)
	e.logger.Info("Extractor", "🔧 Configuration forcée - Local Path: "+localPath)

	// Répercuter sur la config interne pour les fonctions qui lisent e.config
	e.config.RepoURL = repoURL
	e.config.LocalPath = localPath

	// Vérifier si le repository local existe
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		e.logger.Info("Extractor", "Clonage du repository depuis "+repoURL)
		// Cloner le repository
		cmd := exec.Command("git", "clone", repoURL, localPath)
		if err := cmd.Run(); err != nil {
			e.logger.Error("Extractor", "Erreur lors du clonage: "+err.Error())
			return nil, fmt.Errorf("git clone failed: %w", err)
		}
	} else {
		e.logger.Info("Extractor", "Repository local trouvé, mise à jour...")
		// Mettre à jour le repository
		cmd := exec.Command("git", "-C", localPath, "pull")
		if err := cmd.Run(); err != nil {
			e.logger.Error("Extractor", "Erreur lors de la mise à jour: "+err.Error())
			return nil, fmt.Errorf("git pull failed: %w", err)
		}
	}

	e.logger.Info("Extractor", "✅ Repository synchronisé")

	e.logger.Info("Extractor", "🔍 Parsing des fichiers pour extraire les IPs...")
	e.logger.Info("Extractor", "📁 Parsing du répertoire: "+localPath)

	// Parser les fichiers .nft
	scanners, err := e.parseFilesForIPs(localPath)
	if err != nil {
		e.logger.Error("Extractor", "Erreur lors du parsing: "+err.Error())
		return nil, fmt.Errorf("parse failed: %w", err)
	}

	if len(scanners) == 0 {
		e.logger.Error("Extractor", "Aucune IP trouvée")
		return nil, fmt.Errorf("no IPs found in repository")
	}

	e.logger.Info("Extractor", "✅ "+fmt.Sprintf("%d", len(scanners))+" IPs uniques extraites au total")

	// Enrichir les données
	e.logger.Info("Extractor", "🌍 Enrichissement des données...")
	enrichedData, err := e.enrichData(scanners)
	if err != nil {
		e.logger.Error("Extractor", "Erreur lors de l'enrichissement: "+err.Error())
		return nil, fmt.Errorf("enrichment failed: %w", err)
	}
	e.logger.Info("Extractor", "✅ "+fmt.Sprintf("%d", len(enrichedData))+" enregistrements enrichis")

	// Sauvegarde immédiate en CSV (horodaté)
	ts := time.Now().Format("2006-01-02_15-04-05")
	csvName := fmt.Sprintf("%s_liacheckscanner.csv", ts)
	if err := e.SaveToCSV(enrichedData, csvName); err != nil {
		e.logger.Warning("Extractor", "Erreur lors de la sauvegarde CSV: "+err.Error())
	} else {
		e.logger.Info("Extractor", "📊 Sauvegarde en CSV...")
		// Log de succès déjà géré dans SaveToCSV
	}

	e.logger.Info("Extractor", "✅ Extraction terminée: "+fmt.Sprintf("%d", len(enrichedData))+" enregistrements")
	return enrichedData, nil
}

// cloneOrUpdateRepo clone ou met à jour le repository
func (e *Extractor) cloneOrUpdateRepo() error {
	e.logger.Info("Extractor", "📥 Clonage/mise à jour du repository...")

	// Vérifier si le dossier existe déjà
	if _, err := os.Stat(e.config.LocalPath); os.IsNotExist(err) {
		// Créer le dossier parent
		parentDir := filepath.Dir(e.config.LocalPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("erreur lors de la création du dossier: %v", err)
		}

		// Cloner le repository
		e.logger.Info("Extractor", "Clonage du repository depuis "+e.config.RepoURL)
		// Pour l'instant, on simule le clonage
		// Dans une vraie implémentation, on utiliserait go-git
	} else {
		e.logger.Info("Extractor", "Repository local trouvé, mise à jour...")
		// Pour l'instant, on simule la mise à jour
		// Dans une vraie implémentation, on utiliserait go-git
	}

	e.logger.Info("Extractor", "✅ Repository synchronisé")
	return nil
}

// parseFilesForIPs parse les fichiers pour extraire les IPs
func (e *Extractor) parseFilesForIPs(localPath string) ([]string, error) {
	e.logger.Info("Extractor", "🔍 Parsing des fichiers pour extraire les IPs...")

	// Vérifier si le chemin existe
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("le répertoire %s n'existe pas", localPath)
	}

	e.logger.Info("Extractor", fmt.Sprintf("📁 Parsing du répertoire: %s", localPath))

	var ips []string

	// Regex pour IPv4 et IPv6
	ipv4Regex := regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}(?:/\d{1,2})?\b`)
	ipv6Regex := regexp.MustCompile(`(?:[a-fA-F0-9]{0,4}:){2,7}[a-fA-F0-9]{0,4}(?:/\d{1,3})?`)

	// Parcourir le répertoire local
	err := filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Ignorer les dossiers .git et les fichiers binaires
		if info.IsDir() && (info.Name() == ".git" || strings.HasPrefix(info.Name(), ".")) {
			return filepath.SkipDir
		}

		// Traiter seulement les fichiers .nft (Netfilter)
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".nft") {
			e.logger.Info("Extractor", fmt.Sprintf("📄 Traitement du fichier: %s", filepath.Base(path)))
			fileIPs, err := e.extractIPsFromNFTFile(path, ipv4Regex, ipv6Regex)
			if err != nil {
				e.logger.Warning("Extractor", fmt.Sprintf("Erreur lors du parsing de %s: %v", path, err))
				return nil // Continuer avec les autres fichiers
			}
			e.logger.Info("Extractor", fmt.Sprintf("✅ %s: %d IPs extraites", filepath.Base(path), len(fileIPs)))
			ips = append(ips, fileIPs...)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Supprimer les doublons
	uniqueIPs := make(map[string]bool)
	var uniqueIPList []string
	for _, ip := range ips {
		if !uniqueIPs[ip] {
			uniqueIPs[ip] = true
			uniqueIPList = append(uniqueIPList, ip)
		}
	}

	e.logger.Info("Extractor", fmt.Sprintf("✅ %d IPs uniques extraites au total", len(uniqueIPList)))
	return uniqueIPList, nil
}

// extractIPsFromNFTFile extrait les IPs d'un fichier .nft
func (e *Extractor) extractIPsFromNFTFile(filePath string, ipv4Regex, ipv6Regex *regexp.Regexp) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ips []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ignorer les commentaires et lignes vides
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Extraire les IPv4
		ipv4Matches := ipv4Regex.FindAllString(line, -1)
		ips = append(ips, ipv4Matches...)

		// Extraire les IPv6
		ipv6Matches := ipv6Regex.FindAllString(line, -1)
		ips = append(ips, ipv6Matches...)
	}

	return ips, scanner.Err()
}

// isTextFile vérifie si un fichier est un fichier texte
func (e *Extractor) isTextFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	textExtensions := map[string]bool{
		".txt": true, ".md": true, ".json": true, ".yaml": true, ".yml": true,
		".xml": true, ".csv": true, ".conf": true, ".cfg": true, ".ini": true,
		".sh": true, ".py": true, ".js": true, ".html": true, ".css": true,
	}

	return textExtensions[ext] || ext == ""
}

// enrichData enrichit les données avec des informations supplémentaires
func (e *Extractor) enrichData(ips []string) ([]models.ScannerData, error) {
	e.logger.Info("Extractor", "🌍 Enrichissement des données...")

	// Mapper les IPs vers leurs fichiers sources
	ipToScanner := e.mapIPsToScanners(ips)

	var scannerData []models.ScannerData
	now := time.Now()

	for i, ip := range ips {
		// Obtenir les informations du scanner basées sur le fichier source
		scannerInfo := ipToScanner[ip]

		// Créer un enregistrement de base
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

		// Enrichissement réel RDAP + Géolocalisation (sans clé API)
		if err := e.enrichWithAPI(&data); err != nil {
			e.logger.Warning("Extractor", fmt.Sprintf("Erreur lors de l'enrichissement de %s: %v", ip, err))
		}

		scannerData = append(scannerData, data)
	}

	e.logger.Info("Extractor", fmt.Sprintf("✅ %d enregistrements enrichis", len(scannerData)))
	return scannerData, nil
}

// ScannerInfo contient les informations d'un scanner
type ScannerInfo struct {
	Name       string
	Type       models.ScannerType
	SourceFile string
}

// mapIPsToScanners mappe les IPs vers leurs informations de scanner
func (e *Extractor) mapIPsToScanners(ips []string) map[string]ScannerInfo {
	ipToScanner := make(map[string]ScannerInfo)

	// Parcourir tous les fichiers .nft pour mapper les IPs
	err := filepath.Walk(e.config.LocalPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".nft") {
			return nil
		}

		// Extraire le nom du scanner depuis le nom du fichier
		fileName := filepath.Base(path)
		scannerName := strings.TrimSuffix(fileName, ".nft")
		scannerType := e.getScannerType(scannerName)

		// Lire le fichier et mapper les IPs
		fileIPs, err := e.extractIPsFromNFTFile(path,
			regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}(?:/\d{1,2})?\b`),
			regexp.MustCompile(`(?:[a-fA-F0-9]{0,4}:){2,7}[a-fA-F0-9]{0,4}(?:/\d{1,3})?`))
		if err != nil {
			return nil
		}

		// Mapper chaque IP à ce scanner
		for _, ip := range fileIPs {
			ipToScanner[ip] = ScannerInfo{
				Name:       scannerName,
				Type:       scannerType,
				SourceFile: fileName,
			}
		}

		return nil
	})

	if err != nil {
		e.logger.Warning("Extractor", "Erreur lors du mapping des scanners: "+err.Error())
	}

	return ipToScanner
}

// getScannerType détermine le type de scanner basé sur le nom
func (e *Extractor) getScannerType(scannerName string) models.ScannerType {
	switch strings.ToLower(scannerName) {
	case "shodan":
		return models.ScannerTypeShodan
	case "censys":
		return models.ScannerTypeCensys
	case "binaryedge":
		return models.ScannerTypeBinaryEdge
	case "rapid7":
		return models.ScannerTypeRapid7
	case "shadowserver":
		return models.ScannerTypeShadowServer
	default:
		return models.ScannerTypeOther
	}
}

// rdapCache manages simple on-disk cache
type rdapCache struct {
	Entries map[string]models.RDAPCacheEntry `json:"entries"`
	Path    string                           `json:"-"`
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
	return c
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
	// copy fields
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

// enrichWithAPI enrichit les données avec RDAP (WHOIS moderne) et géolocalisation publique
func (e *Extractor) enrichWithAPI(data *models.ScannerData) error {
	// Respecter un léger throttling si configuré
	if e.config.APIThrottle > 0 {
		time.Sleep(time.Duration(e.config.APIThrottle) * time.Second)
	}

	cache := e.loadRDAPCache()
	if e.applyCache(data.IPOrCIDR, data, cache) {
		return nil
	}

	// RDAP complet
	_ = e.performRDAPFull(data.IPOrCIDR, data)

	// Géolocalisation étendue + ASN + reverse
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

	// Reverse DNS via resolver système si rien obtenu
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

// performRDAPFull remplit directement les champs RDAP/contacts sur data
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
		// Champs simples
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
		// Réseaux CIDR
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
		// Entités (emails)
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

// performRDAPDetail retourne quelques champs RDAP courants (name/handle/cidr/registry)
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

// performGeoLookupExtended interroge ip-api.com pour code pays / pays / ISP / AS / reverse
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

// getCountryName retourne le nom du pays à partir du code
func (e *Extractor) getCountryName(code string) string {
	countries := map[string]string{
		"FR": "France",
		"US": "United States",
		"DE": "Germany",
		"GB": "United Kingdom",
		"CA": "Canada",
		"AU": "Australia",
		"JP": "Japan",
		"BR": "Brazil",
		"IN": "India",
		"RU": "Russia",
	}

	if name, exists := countries[code]; exists {
		return name
	}
	return "Unknown"
}

// getRiskLevel retourne le niveau de risque basé sur le score
func (e *Extractor) getRiskLevel(score int) string {
	switch {
	case score >= 80:
		return "Critical"
	case score >= 60:
		return "High"
	case score >= 40:
		return "Medium"
	case score >= 20:
		return "Low"
	default:
		return "Very Low"
	}
}

// SaveToJSON sauvegarde les données en JSON
func (e *Extractor) SaveToJSON(data []models.ScannerData, filename string) error {
	e.logger.Info("Extractor", "💾 Sauvegarde en JSON...")

	// Créer le dossier results s'il n'existe pas
	if err := os.MkdirAll(e.config.ResultsDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(e.config.ResultsDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		return err
	}

	e.logger.Info("Extractor", fmt.Sprintf("✅ Données sauvegardées: %s", filePath))
	return nil
}

// SaveToCSV sauvegarde les données en CSV
func (e *Extractor) SaveToCSV(data []models.ScannerData, filename string) error {
	e.logger.Info("Extractor", "📊 Sauvegarde en CSV...")

	// Créer le dossier results s'il n'existe pas
	if err := os.MkdirAll(e.config.ResultsDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(e.config.ResultsDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Écrire l'en-tête
	headers := []string{
		"ID", "IP/CIDR", "Scanner Name", "Scanner Type", "Source File",
		"Country Code", "Country Name", "ISP", "Organization",
		"RDAP Name", "RDAP Handle", "RDAP CIDR", "RDAP Registry",
		"Start Address", "End Address", "IP Version", "RDAP Type", "Parent Handle",
		"Event Registration", "Event Last Changed",
		"ASN", "AS Name", "Reverse DNS",
		"Abuse Confidence Score", "Abuse Reports", "Usage Type",
		"Domain", "Last Seen", "First Seen", "Tags", "Notes",
		"Risk Level", "Export Date", "Abuse Email", "Tech Email",
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Écrire les données
	for _, item := range data {
		row := []string{
			item.ID,
			item.IPOrCIDR,
			item.ScannerName,
			string(item.ScannerType),
			item.SourceFile,
			item.CountryCode,
			item.CountryName,
			item.ISP,
			item.Organization,
			item.RDAPName,
			item.RDAPHandle,
			item.RDAPCIDR,
			item.Registry,
			item.StartAddress,
			item.EndAddress,
			item.IPVersion,
			item.RDAPType,
			item.ParentHandle,
			item.EventRegistration,
			item.EventLastChanged,
			item.ASN,
			item.ASName,
			item.ReverseDNS,
			fmt.Sprintf("%d", item.AbuseConfidenceScore),
			fmt.Sprintf("%d", item.AbuseReports),
			item.UsageType,
			item.Domain,
			item.LastSeen.Format("2006-01-02 15:04:05"),
			item.FirstSeen.Format("2006-01-02 15:04:05"),
			strings.Join(item.Tags, ", "),
			item.Notes,
			item.RiskLevel,
			item.ExportDate.Format("2006-01-02 15:04:05"),
			item.AbuseEmail,
			item.TechEmail,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	e.logger.Info("Extractor", fmt.Sprintf("✅ Données sauvegardées: %s", filePath))
	return nil
}

// LoadFromJSON charge les données depuis un fichier JSON
func (e *Extractor) LoadFromJSON(filename string) ([]models.ScannerData, error) {
	// Essayer d'abord dans le dossier results
	filePath := filepath.Join(e.config.ResultsDir, filename)
	file, err := os.Open(filePath)
	if err != nil {
		// Si pas trouvé dans results, essayer dans data
		filePath = filepath.Join("data", filename)
		file, err = os.Open(filePath)
		if err != nil {
			// Si toujours pas trouvé, retourner une erreur pour forcer l'extraction
			return nil, fmt.Errorf("aucun fichier de données trouvé")
		}
	}
	defer file.Close()

	var data []models.ScannerData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		e.logger.Warning("Extractor", "Erreur lors du décodage JSON")
		return nil, fmt.Errorf("erreur lors du décodage JSON: %w", err)
	}

	return data, nil
}

// EnrichRecordWithDelay enrichit un enregistrement en imposant un délai (millisecondes) entre les requêtes
func (e *Extractor) EnrichRecordWithDelay(data *models.ScannerData, delayMs int) error {
	// Sauvegarder la valeur de throttling actuelle
	prev := e.config.APIThrottle
	if delayMs >= 0 {
		// Convertir millisecondes -> secondes
		e.config.APIThrottle = float64(delayMs) / 1000.0
	}
	defer func() { e.config.APIThrottle = prev }()
	return e.enrichWithAPI(data)
}

// GeoLookupContinent retourne continent/continentCode/pays/paysCode pour une IP
func (e *Extractor) GeoLookupContinent(ip string) (string, string, string, string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	url := "http://ip-api.com/json/" + ip + "?fields=status,continent,continentCode,country,countryCode"
	resp, err := client.Get(url)
	if err != nil {
		return "", "", "", "", err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", "", "", fmt.Errorf("geo http %d", resp.StatusCode)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		return "", "", "", "", err
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

// LoadProgressTracker charge le fichier de progression RDAP s'il existe
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

// SaveProgressTracker sauvegarde la progression RDAP
func (e *Extractor) SaveProgressTracker(tracker *models.RDAPProgressTracker) error {
	progressPath := filepath.Join("build", "data", "rdap_progress.json")
	_ = os.MkdirAll(filepath.Dir(progressPath), 0755)

	tracker.LastUpdatedAt = time.Now().Format(time.RFC3339)

	file, err := os.Create(progressPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(tracker)
}

// IsIPProcessed vérifie si une IP a déjà été traitée
func (e *Extractor) IsIPProcessed(ip string, tracker *models.RDAPProgressTracker) bool {
	for _, processed := range tracker.ProcessedIPs {
		if processed == ip {
			return true
		}
	}
	return false
}

// ClearProgressTracker supprime le fichier de progression
func (e *Extractor) ClearProgressTracker() error {
	progressPath := filepath.Join("build", "data", "rdap_progress.json")
	return os.Remove(progressPath)
}
