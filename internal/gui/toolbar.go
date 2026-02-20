// Package gui provides the graphical user interface for LiaCheckScanner.
// This file contains toolbar buttons, tab creation, and event handlers for
// the search, configuration, and logs tabs.
package gui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/lia/liacheckscanner_go/internal/config"
)

// createSearchTab creates the advanced search tab with professional features
// Returns a CanvasObject containing the search interface
func (a *App) createSearchTab() fyne.CanvasObject {
	// Professional title
	title := widget.NewLabel("🔍 Advanced Search & IP Enrichment")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	// Advanced search section
	searchLabel := widget.NewLabel("🔎 Search Criteria")
	searchLabel.TextStyle = fyne.TextStyle{Bold: true}

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Enter IP, CIDR, scanner name, or country code...")

	// Professional filters
	filterLabel := widget.NewLabel("🎯 Advanced Filters")
	filterLabel.TextStyle = fyne.TextStyle{Bold: true}

	countryFilter := widget.NewSelect([]string{"All Countries", "US", "FR", "DE", "GB", "CA", "AU", "JP", "BR", "IN", "RU", "CN"}, nil)
	countryFilter.SetSelected("All Countries")

	scannerFilter := widget.NewSelect([]string{"All Scanners", "Shodan", "Censys", "BinaryEdge", "Fofa", "Quake", "Hunter", "LeakIX", "ShadowServer"}, nil)
	scannerFilter.SetSelected("All Scanners")

	riskFilter := widget.NewSelect([]string{"All Risk Levels", "High", "Medium", "Low", "Unknown"}, nil)
	riskFilter.SetSelected("All Risk Levels")

	// Professional action buttons
	searchBtn := widget.NewButton("🔍 Perform Search", func() {
		a.performAdvancedSearch(searchEntry.Text, countryFilter.Selected, scannerFilter.Selected, riskFilter.Selected)
	})

	enrichBtn := widget.NewButton("🌍 Enrich IP Data", func() {
		a.enrichIPData(searchEntry.Text)
	})

	exportBtn := widget.NewButton("📤 Export Results", func() {
		a.exportSearchResults()
	})

	clearBtn := widget.NewButton("🗑️ Clear Results", func() {
		searchEntry.SetText("")
		countryFilter.SetSelected("All Countries")
		scannerFilter.SetSelected("All Scanners")
		riskFilter.SetSelected("All Risk Levels")
		a.clearSearchResults()
	})

	// Professional filter layout
	filtersContainer := container.NewGridWithColumns(3,
		container.NewVBox(widget.NewLabel("Country:"), countryFilter),
		container.NewVBox(widget.NewLabel("Scanner:"), scannerFilter),
		container.NewVBox(widget.NewLabel("Risk Level:"), riskFilter),
	)

	// Professional button layout
	buttonsContainer := container.NewHBox(
		searchBtn,
		enrichBtn,
		exportBtn,
		clearBtn,
	)

	// Professional results section
	resultsLabel := widget.NewLabel("📊 Search Results")
	resultsLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Professional results table
	a.searchResultsTable = widget.NewTable(
		func() (int, int) {
			return len(a.searchResults), 8
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			if i.Row < len(a.searchResults) {
				item := a.searchResults[i.Row]
				switch i.Col {
				case 0:
					label.SetText(item.IPOrCIDR)
				case 1:
					label.SetText(item.ScannerName)
				case 2:
					label.SetText(string(item.ScannerType))
				case 3:
					label.SetText(item.CountryCode)
				case 4:
					label.SetText(item.ISP)
				case 5:
					label.SetText(item.RiskLevel)
				case 6:
					label.SetText(fmt.Sprintf("%d", item.AbuseConfidenceScore))
				case 7:
					label.SetText(item.LastSeen.Format("2006-01-02"))
				}
			}
		},
	)

	// Professional results headers
	resultsHeaders := []string{"IP/CIDR", "Scanner", "Type", "Country", "ISP", "Risk", "Score", "Last Seen"}
	resultsHeaderContainer := container.NewHBox()
	for _, header := range resultsHeaders {
		headerLabel := widget.NewLabel(header)
		headerLabel.TextStyle = fyne.TextStyle{Bold: true}
		headerLabel.Alignment = fyne.TextAlignCenter
		resultsHeaderContainer.Add(headerLabel)
	}

	// Professional enrichment section
	enrichmentLabel := widget.NewLabel("🌍 Real-time IP Enrichment")
	enrichmentLabel.TextStyle = fyne.TextStyle{Bold: true}

	a.enrichmentText = widget.NewEntry()
	a.enrichmentText.SetPlaceHolder("IP enrichment information will appear here...")
	a.enrichmentText.MultiLine = true
	a.enrichmentText.Disable()

	// Professional statistics
	a.searchStatsLabel = widget.NewLabel("📈 Statistics: 0 results found")
	a.searchStatsLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Professional main layout
	searchContainer := container.NewVBox(
		title,
		searchLabel,
		searchEntry,
		filterLabel,
		filtersContainer,
		buttonsContainer,
		resultsLabel,
		resultsHeaderContainer,
		container.NewScroll(a.searchResultsTable),
		enrichmentLabel,
		a.enrichmentText,
		a.searchStatsLabel,
	)

	return container.NewScroll(searchContainer)
}

// createConfigTab creates the configuration tab with professional settings
// Returns a CanvasObject containing the configuration interface
func (a *App) createConfigTab() fyne.CanvasObject {
	// Professional title
	title := widget.NewLabel("⚙️ System Configuration")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	// Database configuration
	dbTitle := widget.NewLabel("🗄️ Database Settings")
	dbTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Results and logs directories
	resultsEntry := widget.NewEntry()
	resultsEntry.SetText(a.config.Database.ResultsDir)
	resultsEntry.SetPlaceHolder("Results directory...")

	logsEntry := widget.NewEntry()
	logsEntry.SetText(a.config.Database.LogsDir)
	logsEntry.SetPlaceHolder("Logs directory...")

	// Repository configuration
	repoTitle := widget.NewLabel("📥 Repository Settings")
	repoTitle.TextStyle = fyne.TextStyle{Bold: true}

	repoURLEntry := widget.NewEntry()
	repoURLEntry.SetText(a.config.Database.RepoURL)
	repoURLEntry.SetPlaceHolder("Repository URL...")

	localPathEntry := widget.NewEntry()
	localPathEntry.SetText(a.config.Database.LocalPath)
	localPathEntry.SetPlaceHolder("Local repository path...")

	// Throttling configuration
	throttleTitle := widget.NewLabel("⏱️ RDAP/Geo Throttle (ms)")
	throttleTitle.TextStyle = fyne.TextStyle{Bold: true}
	throttleEntry := widget.NewEntry()
	throttleEntry.SetPlaceHolder("e.g. 500")
	throttleEntry.SetText(fmt.Sprintf("%d", int(a.config.Database.APIThrottle*1000)))

	// Parallelism configuration
	parTitle := widget.NewLabel("🧵 Parallelism (workers)")
	parTitle.TextStyle = fyne.TextStyle{Bold: true}
	parEntry := widget.NewEntry()
	parEntry.SetPlaceHolder("e.g. 4")
	if a.config.Database.Parallelism <= 0 {
		a.config.Database.Parallelism = 1
	}
	parEntry.SetText(fmt.Sprintf("%d", a.config.Database.Parallelism))

	// RDAP Registries selection
	rTitle := widget.NewLabel("🌐 RDAP Registries")
	rTitle.TextStyle = fyne.TextStyle{Bold: true}
	allRegs := []string{"arin", "ripe", "apnic", "lacnic", "afrinic"}
	regChecks := []*widget.Check{}
	if len(a.config.Database.Registries) == 0 {
		a.config.Database.Registries = allRegs
	}
	for _, r := range allRegs {
		val := false
		for _, sr := range a.config.Database.Registries {
			if sr == r {
				val = true
				break
			}
		}
		chk := widget.NewCheck(strings.ToUpper(r), func(bool) {})
		chk.SetChecked(val)
		regChecks = append(regChecks, chk)
	}

	// Save button update for registries
	saveBtn := widget.NewButton("💾 Save Configuration", func() {
		// Update configuration
		a.config.Database.RepoURL = repoURLEntry.Text
		a.config.Database.LocalPath = localPathEntry.Text
		if ms, err := strconv.Atoi(strings.TrimSpace(throttleEntry.Text)); err == nil && ms >= 0 {
			a.config.Database.APIThrottle = float64(ms) / 1000.0
		}
		if p, err := strconv.Atoi(strings.TrimSpace(parEntry.Text)); err == nil && p > 0 {
			a.config.Database.Parallelism = p
		}
		// registries
		var regs []string
		for i, r := range allRegs {
			if regChecks[i].Checked {
				regs = append(regs, r)
			}
		}
		if len(regs) == 0 {
			regs = allRegs
		}
		a.config.Database.Registries = regs
		// Save
		cm := config.NewConfigManager()
		_, _ = cm.Load()
		if err := cm.Save(a.config); err != nil {
			dialog.ShowError(err, a.mainWindow)
		} else {
			dialog.ShowInformation("Success", "Configuration saved successfully", a.mainWindow)
		}
	})

	resetBtn := widget.NewButton("🔄 Reset to Defaults", func() {
		dialog.ShowConfirm("Reset Configuration", "Are you sure you want to reset to defaults?", func(confirm bool) {
			if confirm {
				// Reset to defaults
				repoURLEntry.SetText("https://github.com/MDMCK10/internet-scanners")
				localPathEntry.SetText("./internet-scanners")
			}
		}, a.mainWindow)
	})

	// Professional layout
	configContainer := container.NewVBox(
		title,
		dbTitle,
		container.NewVBox(
			widget.NewLabel("Results Directory:"),
			resultsEntry,
		),
		container.NewVBox(
			widget.NewLabel("Logs Directory:"),
			logsEntry,
		),
		repoTitle,
		container.NewVBox(
			widget.NewLabel("Repository URL:"),
			repoURLEntry,
		),
		container.NewVBox(
			widget.NewLabel("Local Path:"),
			localPathEntry,
		),
		container.NewVBox(
			throttleTitle,
			throttleEntry,
		),
		container.NewVBox(
			parTitle,
			parEntry,
		),
		rTitle,
		container.NewGridWithColumns(3, func() []fyne.CanvasObject {
			items := []fyne.CanvasObject{}
			for _, c := range regChecks {
				items = append(items, c)
			}
			return items
		}()...),
		container.NewHBox(
			saveBtn,
			resetBtn,
		),
	)

	return container.NewScroll(configContainer)
}

// createLogsTab creates the logs tab with professional log viewing
// Returns a CanvasObject containing the logs interface
func (a *App) createLogsTab() fyne.CanvasObject {
	// Professional title
	title := widget.NewLabel("📋 System Logs")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	// Log level filter
	levelLabel := widget.NewLabel("🔍 Log Level Filter")
	levelLabel.TextStyle = fyne.TextStyle{Bold: true}

	levelFilter := widget.NewSelect([]string{"All", "INFO", "WARNING", "ERROR"}, func(level string) {
		// Filter logs by level
		a.filterLogs(level)
	})
	levelFilter.SetSelected("All")

	// Professional log display
	logDisplay := widget.NewMultiLineEntry()
	logDisplay.SetPlaceHolder("System logs will appear here...")
	logDisplay.Disable()

	// Professional action buttons
	refreshBtn := widget.NewButton("🔄 Refresh Logs", func() {
		a.refreshLogs(logDisplay)
	})

	exportBtn := widget.NewButton("📤 Export Logs", func() {
		a.exportLogs()
	})

	exportZipBtn := widget.NewButton("📦 Export Logs (ZIP)", func() {
		ts := time.Now().Format("20060102_150405")
		zipPath := filepath.Join("build", fmt.Sprintf("logs_%s.zip", ts))
		if err := a.zipDirectory("logs", zipPath); err != nil {
			dialog.ShowError(err, a.mainWindow)
			return
		}
		dialog.ShowInformation("Logs", "Exported to "+zipPath, a.mainWindow)
	})

	clearBtn := widget.NewButton("🗑️ Clear Display", func() {
		logDisplay.SetText("")
	})

	// Professional layout
	logsContainer := container.NewVBox(
		title,
		levelLabel,
		levelFilter,
		container.NewHBox(
			refreshBtn,
			exportBtn,
			exportZipBtn,
			clearBtn,
		),
		container.NewScroll(logDisplay),
	)

	return container.NewScroll(logsContainer)
}

// performAdvancedSearch performs advanced search with multiple criteria
func (a *App) performAdvancedSearch(query, country, scanner, risk string) {
	results := FilterAdvancedSearch(a.data, query, country, scanner, risk)
	a.searchResults = results
	if a.searchResultsTable != nil {
		a.searchResultsTable.Refresh()
	}

	// Update search statistics
	if a.searchStatsLabel != nil {
		a.searchStatsLabel.SetText(fmt.Sprintf("📈 Search Results: %d records found", len(results)))
	}

	a.displaySearchStatistics(results)
}

// enrichIPData performs IP enrichment with real APIs
func (a *App) enrichIPData(query string) {
	if query == "" {
		dialog.ShowInformation("Enrichment", "Please enter an IP address to enrich", a.mainWindow)
		return
	}

	// Show loading message
	if a.enrichmentText != nil {
		a.enrichmentText.SetText("🔄 Enriching IP data... Please wait...")
	}

	// Run enrichment in background
	go func() {
		result := a.performRealIPEnrichment(query)
		if a.enrichmentText != nil {
			a.enrichmentText.SetText(result)
		}
	}()
}

// filterLogs filters logs by level (placeholder implementation)
func (a *App) filterLogs(level string) {
	// Implementation would filter logs by level
	a.logger.Info("GUI", "Filtering logs by level: "+level)
}

// refreshLogs refreshes the log display (placeholder implementation)
func (a *App) refreshLogs(logDisplay *widget.Entry) {
	// Implementation would load and display actual logs
	logDisplay.SetText("📋 System Logs\n• Application started\n• Data loaded successfully\n• Interface ready")
}
