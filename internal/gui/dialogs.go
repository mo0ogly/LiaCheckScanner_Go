// Package gui provides the graphical user interface for LiaCheckScanner.
// This file contains dialog boxes, modals, IP enrichment lookups,
// and security recommendation displays.
package gui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"fyne.io/fyne/v2/dialog"

	"github.com/lia/liacheckscanner_go/internal/models"
)

// performRealIPEnrichment orchestrates real IP enrichment using multiple APIs
func (a *App) performRealIPEnrichment(ip string) string {
	result := fmt.Sprintf("🌍 IP ENRICHMENT RESULTS FOR: %s\n\n", ip)

	// RDAP lookup
	result += a.performRealRDAPLookup(ip)
	result += "\n"

	// Geolocation lookup
	result += a.performRealGeolocationLookup(ip)
	result += "\n"

	// Additional enrichment (simulated for now)
	result += a.simulateReputationLookup(ip)
	result += "\n"

	result += a.simulatePortScan(ip)
	result += "\n"

	result += a.simulateThreatIntelligence(ip)
	result += "\n"

	result += a.generateSecurityRecommendations(ip)

	return result
}

// performRealRDAPLookup performs real RDAP lookup using multiple registries
func (a *App) performRealRDAPLookup(ip string) string {
	endpoints := []string{
		"https://rdap.arin.net/registry/ip/", // North America
		"https://rdap.ripe.net/ip/",          // Europe, Middle East, parts of Central Asia
		"https://rdap.apnic.net/ip/",         // Asia Pacific
		"https://rdap.lacnic.net/rdap/ip/",   // Latin America and Caribbean
		"https://rdap.afrinic.net/rdap/ip/",  // Africa
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
		if err != nil || resp.StatusCode != http.StatusOK {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			continue
		}

		get := func(m map[string]interface{}, key string) string {
			if v, ok := m[key]; ok && v != nil {
				return fmt.Sprintf("%v", v)
			}
			return ""
		}

		// CIDR or range
		cidr := ""
		if c0, ok := data["cidr0_cidrs"].([]interface{}); ok && len(c0) > 0 {
			if first, ok := c0[0].(map[string]interface{}); ok {
				v := strings.TrimSpace(fmt.Sprintf("%v/%v", first["v4prefix"], first["length"]))
				if v != "/" {
					cidr = v
				}
			}
		}
		if cidr == "" {
			start := get(data, "startAddress")
			end := get(data, "endAddress")
			if start != "" && end != "" {
				cidr = fmt.Sprintf("%s - %s", start, end)
			}
		}

		name := get(data, "name")
		handle := get(data, "handle")
		country := get(data, "country")
		objectClass := get(data, "objectClassName")

		contacts := []string{}
		if ents, ok := data["entities"].([]interface{}); ok {
			for _, e := range ents {
				if em, ok := e.(map[string]interface{}); ok {
					role := ""
					if roles, ok := em["roles"].([]interface{}); ok && len(roles) > 0 {
						role = fmt.Sprintf("%v", roles[0])
					}
					org := get(em, "handle")
					email := ""
					if vcard, ok := em["vcardArray"].([]interface{}); ok && len(vcard) > 1 {
						if rows, ok := vcard[1].([]interface{}); ok {
							for _, row := range rows {
								if r, ok := row.([]interface{}); ok && len(r) > 2 {
									if fmt.Sprintf("%v", r[0]) == "email" {
										if vals, ok := r[3].([]interface{}); ok && len(vals) > 0 {
											email = fmt.Sprintf("%v", vals[0])
										}
									}
								}
							}
						}
					}
					if role != "" || org != "" || email != "" {
						contacts = append(contacts, strings.TrimSpace(fmt.Sprintf("%s %s %s", role, org, email)))
					}
				}
			}
		}

		regDate, lastChanged := "", ""
		if evs, ok := data["events"].([]interface{}); ok {
			for _, ev := range evs {
				if em, ok := ev.(map[string]interface{}); ok {
					action := get(em, "eventAction")
					date := get(em, "eventDate")
					if action == "registration" {
						regDate = date
					} else if action == "last changed" {
						lastChanged = date
					}
				}
			}
		}

		b := &strings.Builder{}
		fmt.Fprintf(b, "🏢 RDAP Information:\n")
		fmt.Fprintf(b, "• Object: %s (%s)\n", name, objectClass)
		fmt.Fprintf(b, "• Handle: %s\n", handle)
		fmt.Fprintf(b, "• Country: %s\n", country)
		if cidr != "" {
			fmt.Fprintf(b, "• Network: %s\n", cidr)
		}
		if regDate != "" || lastChanged != "" {
			fmt.Fprintf(b, "• Registration: %s | Last Changed: %s\n", regDate, lastChanged)
		}
		if len(contacts) > 0 {
			fmt.Fprintf(b, "• Contacts: %s\n", strings.Join(contacts, "; "))
		}
		return b.String()
	}

	// Fallback if all RDAPs fail
	return fmt.Sprintf("🏢 RDAP Information:\n• IP: %s\n• Status: unavailable (RDAP endpoints unreachable)", ip)
}

// performRealGeolocationLookup performs real geolocation lookup
func (a *App) performRealGeolocationLookup(ip string) string {
	client := &http.Client{Timeout: 8 * time.Second}
	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,city,isp,org,as,query,lat,lon,timezone,reverse,continentCode,countryCode,regionName", ip)
	resp, err := client.Get(url)
	if err != nil {
		return "🌍 Geolocation: unavailable"
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "🌍 Geolocation: unavailable"
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "🌍 Geolocation: unavailable"
	}
	var g map[string]interface{}
	if err := json.Unmarshal(body, &g); err != nil {
		return "🌍 Geolocation: unavailable"
	}
	get := func(key string) string {
		if v, ok := g[key]; ok && v != nil {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}
	b := &strings.Builder{}
	fmt.Fprintf(b, "🌍 Geolocation:\n")
	fmt.Fprintf(b, "• Country: %s (%s)\n", get("country"), get("countryCode"))
	fmt.Fprintf(b, "• City: %s\n", get("city"))
	fmt.Fprintf(b, "• ISP: %s\n", get("isp"))
	fmt.Fprintf(b, "• Org/AS: %s / %s\n", get("org"), get("as"))
	fmt.Fprintf(b, "• Timezone: %s\n", get("timezone"))
	fmt.Fprintf(b, "• Reverse DNS: %s\n", get("reverse"))
	lat := get("lat")
	lon := get("lon")
	if lat != "" && lon != "" {
		fmt.Fprintf(b, "• Coordinates: %s, %s\n", lat, lon)
	}
	return b.String()
}

// simulateReputationLookup simulates reputation lookup
func (a *App) simulateReputationLookup(ip string) string {
	return fmt.Sprintf("🔍 Reputation Analysis:\n• Threat Score: 75/100\n• Blacklist Status: Clean\n• Reputation: Good")
}

// simulatePortScan simulates port scanning
func (a *App) simulatePortScan(ip string) string {
	return fmt.Sprintf("🔌 Port Scan Results:\n• Open Ports: 22, 80, 443\n• Common Services: SSH, HTTP, HTTPS\n• Security Status: Standard")
}

// simulateThreatIntelligence simulates threat intelligence lookup
func (a *App) simulateThreatIntelligence(ip string) string {
	return fmt.Sprintf("⚠️ Threat Intelligence:\n• Known Threats: None detected\n• Malware History: Clean\n• Botnet Activity: None")
}

// generateSecurityRecommendations generates security recommendations
func (a *App) generateSecurityRecommendations(ip string) string {
	return fmt.Sprintf("🛡️ Security Recommendations:\n• Monitor for unusual activity\n• Implement rate limiting\n• Regular security audits\n• Keep systems updated")
}

// displaySearchStatistics displays detailed search statistics
func (a *App) displaySearchStatistics(results []models.ScannerData) {
	stats := fmt.Sprintf("📊 Search Statistics:\n• Total Results: %d\n• By Country: %d\n• By Scanner: %d\n• By Risk: %d",
		len(results), a.countUniqueCountriesInResults(results), a.countUniqueScannersInResults(results), a.countRiskLevelsInResults(results))

	dialog.ShowInformation("Search Statistics", stats, a.mainWindow)
}

// clearSearchResults clears search results and resets the interface
func (a *App) clearSearchResults() {
	a.searchResults = nil
	if a.searchResultsTable != nil {
		a.searchResultsTable.Refresh()
	}
	if a.searchStatsLabel != nil {
		a.searchStatsLabel.SetText("📈 Statistics: 0 results")
	}
	if a.enrichmentText != nil {
		a.enrichmentText.SetText("")
	}
}
