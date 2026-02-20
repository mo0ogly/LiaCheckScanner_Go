package extractor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// parseFilesForIPs parses all .nft files in the given directory for IPs.
func (e *Extractor) parseFilesForIPs(localPath string) ([]string, error) {
	e.logger.Info("Extractor", "Parsing des fichiers pour extraire les IPs...")

	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("le repertoire %s n'existe pas", localPath)
	}

	e.logger.Info("Extractor", fmt.Sprintf("Parsing du repertoire: %s", localPath))

	var ips []string

	ipv4Regex := regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}(?:/\d{1,2})?\b`)
	ipv6Regex := regexp.MustCompile(`(?:[a-fA-F0-9]{0,4}:){2,7}[a-fA-F0-9]{0,4}(?:/\d{1,3})?`)

	err := filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && (info.Name() == ".git" || strings.HasPrefix(info.Name(), ".")) {
			return filepath.SkipDir
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".nft") {
			e.logger.Info("Extractor", fmt.Sprintf("Traitement du fichier: %s", filepath.Base(path)))
			fileIPs, err := e.extractIPsFromNFTFile(path, ipv4Regex, ipv6Regex)
			if err != nil {
				e.logger.Warning("Extractor", fmt.Sprintf("Erreur lors du parsing de %s: %v", path, err))
				return nil
			}
			e.logger.Info("Extractor", fmt.Sprintf("%s: %d IPs extraites", filepath.Base(path), len(fileIPs)))
			ips = append(ips, fileIPs...)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking directory %s: %w", localPath, err)
	}

	uniqueIPs := make(map[string]bool)
	var uniqueIPList []string
	for _, ip := range ips {
		if !uniqueIPs[ip] {
			uniqueIPs[ip] = true
			uniqueIPList = append(uniqueIPList, ip)
		}
	}

	e.logger.Info("Extractor", fmt.Sprintf("%d IPs uniques extraites au total", len(uniqueIPList)))
	return uniqueIPList, nil
}

// extractIPsFromNFTFile extracts IPs from a single .nft file.
func (e *Extractor) extractIPsFromNFTFile(filePath string, ipv4Regex, ipv6Regex *regexp.Regexp) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening nft file %s: %w", filePath, err)
	}
	defer file.Close()

	var ips []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		ipv4Matches := ipv4Regex.FindAllString(line, -1)
		ips = append(ips, ipv4Matches...)

		ipv6Matches := ipv6Regex.FindAllString(line, -1)
		ips = append(ips, ipv6Matches...)
	}

	if err := scanner.Err(); err != nil {
		return ips, fmt.Errorf("scanning nft file %s: %w", filePath, err)
	}
	return ips, nil
}
