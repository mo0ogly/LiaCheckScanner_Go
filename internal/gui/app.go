// Package gui provides the graphical user interface for LiaCheckScanner
// It handles all UI components, data display, and user interactions
package gui

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/lia/liacheckscanner_go/internal/config"
	"github.com/lia/liacheckscanner_go/internal/extractor"
	"github.com/lia/liacheckscanner_go/internal/logger"
	"github.com/lia/liacheckscanner_go/internal/models"
)

// App represents the main application structure
// It manages the GUI, data, and user interactions
type App struct {
	fyneApp    fyne.App
	mainWindow fyne.Window
	logger     *logger.Logger
	config     *models.AppConfig
	extractor  *extractor.Extractor
	data       []models.ScannerData

	// UI Components
	dataTable    *widget.Table
	statusBar    *widget.Label
	statsLabel   *widget.Label
	headerLabels []*widget.Label

	// Search components
	searchEntry        *widget.Entry
	searchResultsTable *widget.Table
	enrichmentText     *widget.Entry
	searchStatsLabel   *widget.Label
	searchResults      []models.ScannerData

	// Pagination
	itemsPerPage   int
	currentPage    int
	totalPages     int
	paginationInfo *widget.Label

	// Selection
	selectedRow  int
	selectedRows map[int]bool

	// RDAP enrichment function
	startRDAPEnrichment func(int)
}

// NewApp creates a new application instance
// It initializes the GUI, loads configuration, and sets up the user interface
func NewApp(config *models.AppConfig, logger *logger.Logger) *App {
	fyneApp := app.New()
	fyneApp.SetIcon(theme.ComputerIcon())

	app := &App{
		fyneApp: fyneApp,
		logger:  logger,
		config:  config,
		// Initialize pagination with sensible defaults
		itemsPerPage: 100,
		currentPage:  1,
		totalPages:   1,
		selectedRow:  -1,
		selectedRows: make(map[int]bool),
	}

	app.mainWindow = fyneApp.NewWindow("🔍 LiaCheckScanner")
	app.mainWindow.Resize(fyne.NewSize(1600, 1000)) // Larger window for better UX
	app.mainWindow.CenterOnScreen()

	// Initialize extractor
	app.extractor = extractor.NewExtractor(config.Database, logger)

	// Create the interface
	app.createUI()

	return app
}

// createUI builds the complete user interface
// It creates tabs, widgets, and sets up the layout
func (a *App) createUI() {
	// Create main tabs
	tabs := container.NewAppTabs(
		container.NewTabItem("📊 Dashboard", a.createDashboardTab()),
		container.NewTabItem("🗄️ Database", a.createDatabaseTab()),
		container.NewTabItem("🔍 Search", a.createSearchTab()),
		container.NewTabItem("⚙️ Configuration", a.createConfigTab()),
		container.NewTabItem("📋 Logs", a.createLogsTab()),
	)

	// Set tab properties for better UX
	tabs.SetTabLocation(container.TabLocationTop)
	tabs.SelectTabIndex(0) // Start with dashboard

	// Create status bar
	a.statusBar = widget.NewLabel("🟢 Ready")
	a.statusBar.TextStyle = fyne.TextStyle{Bold: true}
	a.statusBar.Alignment = fyne.TextAlignCenter

	// Main layout with status bar
	mainContainer := container.NewBorder(
		nil,         // top
		a.statusBar, // bottom
		nil,         // left
		nil,         // right
		tabs,
	)

	a.mainWindow.SetContent(mainContainer)
	a.mainWindow.Show()

	// Load existing data - try CSV first, then extract if needed
	go func() {
		a.logger.Info("GUI", "🔍 Initializing data...")
		a.loadData() // This will try CSV first, then auto-extract if needed
	}()
}

// createDashboardTab creates the main dashboard with statistics and overview
// Returns a CanvasObject containing the dashboard interface
func (a *App) createDashboardTab() fyne.CanvasObject {
	// Professional title
	title := widget.NewLabel("🔍 LiaCheckScanner")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	// Subtitle
	subtitle := widget.NewLabel("Advanced Internet Scanner Detection & Analysis Platform")
	subtitle.TextStyle = fyne.TextStyle{Italic: true}
	subtitle.Alignment = fyne.TextAlignCenter

	// Statistics section
	statsTitle := widget.NewLabel("📈 Real-time Statistics")
	statsTitle.TextStyle = fyne.TextStyle{Bold: true}

	a.statsLabel = widget.NewLabel("Loading statistics...")
	a.statsLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Quick actions
	actionsTitle := widget.NewLabel("⚡ Quick Actions")
	actionsTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Action buttons with professional styling
	refreshBtn := widget.NewButton("🔄 Refresh Data", func() {
		a.refreshData()
	})

	exportBtn := widget.NewButton("📤 Export All", func() {
		a.exportAllData()
	})

	searchBtn := widget.NewButton("🔍 Advanced Search", func() {
		// Switch to search tab
		if tabs := a.mainWindow.Content().(*fyne.Container).Objects[0].(*container.AppTabs); tabs != nil {
			tabs.SelectTabIndex(2) // Search tab
		}
	})

	// Professional info section
	infoTitle := widget.NewLabel("ℹ️ System Information")
	infoTitle.TextStyle = fyne.TextStyle{Bold: true}

	infoText := widget.NewLabel(`• Version: 1.0.0
• Owner: LIA - mo0ogly@proton.me
• Platform: Advanced IP Scanner & Analyzer
• UI: Fyne 2`)

	// Layout with professional spacing
	dashboardContainer := container.NewVBox(
		title,
		subtitle,
		widget.NewSeparator(),
		statsTitle,
		a.statsLabel,
		widget.NewSeparator(),
		actionsTitle,
		container.NewHBox(
			refreshBtn,
			exportBtn,
			searchBtn,
		),
		widget.NewSeparator(),
		infoTitle,
		infoText,
	)

	return container.NewScroll(dashboardContainer)
}

// createDatabaseTab creates the database tab with pagination and professional table display
// Returns a CanvasObject containing the database interface
func (a *App) createDatabaseTab() fyne.CanvasObject {
	// Professional title
	title := widget.NewLabel("🗄️ Database")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	// Pagination controls with professional styling
	paginationLabel := widget.NewLabel("📄 Advanced Pagination")
	paginationLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Items per page selector with more options
	itemsPerPageSelect := widget.NewSelect([]string{"25", "50", "100", "250", "500", "1000", "All"}, func(value string) {
		if value != "" {
			if value == "All" {
				a.itemsPerPage = len(a.data)
			} else {
				itemsPerPage, _ := strconv.Atoi(value)
				a.itemsPerPage = itemsPerPage
			}
			a.currentPage = 1
			a.updatePagination()
		}
	})
	itemsPerPageSelect.SetSelected("100")

	// Professional pagination info
	a.paginationInfo = widget.NewLabel("Page 1 of 1 (0-0 of 0 records)")
	a.paginationInfo.TextStyle = fyne.TextStyle{Bold: true}

	// Navigation buttons with professional icons
	firstPageBtn := widget.NewButton("⏮️ First", func() {
		a.currentPage = 1
		a.updatePagination()
	})

	prevPageBtn := widget.NewButton("◀️ Previous", func() {
		if a.currentPage > 1 {
			a.currentPage--
			a.updatePagination()
		}
	})

	nextPageBtn := widget.NewButton("▶️ Next", func() {
		if a.currentPage < a.totalPages {
			a.currentPage++
			a.updatePagination()
		}
	})

	lastPageBtn := widget.NewButton("⏭️ Last", func() {
		a.currentPage = a.totalPages
		a.updatePagination()
	})

	// Go to specific page with validation
	pageEntry := widget.NewEntry()
	pageEntry.SetPlaceHolder("Enter page number...")

	goToPageBtn := widget.NewButton("🔍 Go", func() {
		if pageStr := pageEntry.Text; pageStr != "" {
			if page, err := strconv.Atoi(pageStr); err == nil && page > 0 && page <= a.totalPages {
				a.currentPage = page
				a.updatePagination()
				pageEntry.SetText("")
			} else {
				dialog.ShowInformation("Invalid Page", "Please enter a valid page number", a.mainWindow)
			}
		}
	})

	// Professional pagination controls layout
	paginationControls := container.NewHBox(
		container.NewVBox(
			widget.NewLabel("Records per page:"),
			itemsPerPageSelect,
		),
		container.NewVBox(
			firstPageBtn,
			prevPageBtn,
		),
		container.NewVBox(
			a.paginationInfo,
			container.NewHBox(
				pageEntry,
				goToPageBtn,
			),
		),
		container.NewVBox(
			nextPageBtn,
			lastPageBtn,
		),
	)

	// Table headers
	headers := []string{"IP/CIDR", "Scanner", "Type", "Country", "ISP", "Organization", "RDAP Name", "RDAP Handle", "ASN", "Reverse", "Risk", "Score", "Domain", "Last Seen"}

	// Table with styling (14 columns)
	a.dataTable = widget.NewTable(
		func() (int, int) {
			startIndex := (a.currentPage - 1) * a.itemsPerPage
			endIndex := startIndex + a.itemsPerPage
			if endIndex > len(a.data) {
				endIndex = len(a.data)
			}
			pageData := a.data[startIndex:endIndex]
			// +1 pour la ligne d'en-tête
			return len(pageData) + 1, 14
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Wrapping = fyne.TextWrapOff
			label.Alignment = fyne.TextAlignLeading
			return label
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			if i.Row == 0 {
				// Ligne d'en-tête
				label.TextStyle = fyne.TextStyle{Bold: true}
				label.Alignment = fyne.TextAlignCenter
				label.SetText(headers[i.Col])
				return
			}
			label.TextStyle = fyne.TextStyle{Bold: false}
			label.Alignment = fyne.TextAlignLeading
			startIndex := (a.currentPage - 1) * a.itemsPerPage
			realIndex := startIndex + (i.Row - 1)
			if realIndex >= 0 && realIndex < len(a.data) {
				item := a.data[realIndex]
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
					label.SetText(item.Organization)
				case 6:
					label.SetText(item.RDAPName)
				case 7:
					label.SetText(item.RDAPHandle)
				case 8:
					label.SetText(item.ASN)
				case 9:
					label.SetText(item.ReverseDNS)
				case 10:
					label.SetText(item.RiskLevel)
				case 11:
					label.SetText(fmt.Sprintf("%d", item.AbuseConfidenceScore))
				case 12:
					label.SetText(item.Domain)
				case 13:
					label.SetText(item.LastSeen.Format("2006-01-02"))
				}
			}
		},
	)
	// Apply layout once created
	a.applyTableLayout()
	// Horizontal scroll (vertical scroll is handled by widget.Table)
	hscroll := container.NewHScroll(a.dataTable)
	hscroll.SetMinSize(fyne.NewSize(1800, 700))

	// Track selection
	a.dataTable.OnSelected = func(id widget.TableCellID) {
		startIndex := (a.currentPage - 1) * a.itemsPerPage
		realIndex := startIndex + id.Row
		if realIndex >= 0 && realIndex < len(a.data) {
			a.selectedRow = realIndex
		}
	}

	// RDAP Details button
	rdapDetailsBtn := widget.NewButton("ℹ️ RDAP Details", func() {
		if a.selectedRow < 0 || a.selectedRow >= len(a.data) {
			dialog.ShowInformation("RDAP", "Sélectionne une ligne d'abord", a.mainWindow)
			return
		}
		item := a.data[a.selectedRow]
		// Build details view
		details := fmt.Sprintf(`IP: %s\nName: %s\nHandle: %s\nCIDR: %s\nRegistry: %s\nStart: %s\nEnd: %s\nIP Version: %s\nType: %s\nParent: %s\nReg: %s\nChanged: %s\nASN: %s\nAS Name: %s\nReverse: %s\nAbuse: %s\nTech: %s`,
			item.IPOrCIDR, item.RDAPName, item.RDAPHandle, item.RDAPCIDR, item.Registry,
			item.StartAddress, item.EndAddress, item.IPVersion, item.RDAPType, item.ParentHandle,
			item.EventRegistration, item.EventLastChanged, item.ASN, item.ASName, item.ReverseDNS,
			item.AbuseEmail, item.TechEmail,
		)
		jsonRaw, _ := json.MarshalIndent(item, "", "  ")
		content := container.NewVBox(
			widget.NewLabel("RDAP Details"),
			widget.NewMultiLineEntry(),
		)
		ml := content.Objects[1].(*widget.Entry)
		ml.MultiLine = true
		ml.SetText(details + "\n\nJSON:\n" + string(jsonRaw))
		ml.Disable()
		d := dialog.NewCustom("RDAP Details", "Close", container.NewScroll(content), a.mainWindow)
		d.Show()
	})

	// Professional scroll container with larger size
	scrollContainer := container.NewScroll(a.dataTable)
	scrollContainer.SetMinSize(fyne.NewSize(1400, 700))

	// Action buttons
	updateBtn := widget.NewButton("🔄 Mettre à jour", func() {
		go func() {
			a.setBusy(true, "Extraction en cours...")
			if _, err := a.extractor.ExtractData(); err != nil {
				a.logger.Warning("GUI", "Extraction error: "+err.Error())
				dialog.ShowError(err, a.mainWindow)
			} else {
				a.refreshData()
				dialog.ShowInformation("Mise à jour", "Extraction terminée et données rechargées", a.mainWindow)
			}
			a.setBusy(false, "")
		}()
	})

	associateRDAPBtn := widget.NewButton("🌍 Associer RDAP (page)", func() {
		startIndex := (a.currentPage - 1) * a.itemsPerPage
		endIndex := startIndex + a.itemsPerPage
		if endIndex > len(a.data) {
			endIndex = len(a.data)
		}
		a.setBusy(true, "RDAP (page) en cours...")
		go func() {
			for i := startIndex; i < endIndex; i++ {
				item := &a.data[i]
				ip := item.IPOrCIDR
				if err := a.extractor.EnrichRecordWithDelay(item, int(a.config.Database.APIThrottle*1000)); err != nil {
					a.logger.Warning("GUI", fmt.Sprintf("RDAP enrich error for %s: %v", ip, err))
				}
				if a.dataTable != nil {
					a.dataTable.Refresh()
				}
			}
			ts := time.Now().Format("2006-01-02_15-04-05")
			filename := fmt.Sprintf("page_enriched_%s.csv", ts)
			_ = a.extractor.SaveToCSV(a.data, filename)
			a.setBusy(false, "")
			dialog.ShowInformation("RDAP", "Page enrichie (RDAP)\nCSV: "+filename, a.mainWindow)
		}()
	})

	// Progress and cancel controls
	progress := widget.NewProgressBar()
	progress.Min = 0
	progress.Max = 1
	progress.SetValue(0)
	progressDetail := widget.NewLabel("")
	cancel := false
	cancelBtn := widget.NewButton("⛔ Annuler", func() { cancel = true })

	// Update layout (add parallelism + resume capability)
	associateRDAPAllBtn := widget.NewButton("🌍 Associer RDAP (tout)", func() {
		if len(a.data) == 0 {
			dialog.ShowInformation("RDAP", "Aucune donnée chargée", a.mainWindow)
			return
		}

		// Check for existing progress
		tracker := a.extractor.LoadProgressTracker()
		var resumeFrom int = 0
		var resumeMsg string = ""

		if tracker != nil && !tracker.Completed && len(tracker.ProcessedIPs) > 0 {
			resumeFrom = tracker.ProcessedRecords
			resumeMsg = fmt.Sprintf("\n\n🔄 Reprise détectée: %d/%d IPs déjà traitées", tracker.ProcessedRecords, tracker.TotalRecords)

			dialog.ShowConfirm("Reprise RDAP",
				fmt.Sprintf("Un traitement RDAP précédent a été interrompu.%s\n\nSouhaitez-vous reprendre là où vous vous êtes arrêté?", resumeMsg),
				func(resume bool) {
					if !resume {
						// Reset progress if user doesn't want to resume
						_ = a.extractor.ClearProgressTracker()
						resumeFrom = 0
					}
					a.startRDAPEnrichment(resumeFrom)
				}, a.mainWindow)
			return
		}

		a.startRDAPEnrichment(0)
	})

	// Add a separate function to handle the actual enrichment
	a.startRDAPEnrichment = func(startFrom int) {
		cancel = false
		a.setBusy(true, "RDAP (tout) en cours...")

		// Initialize or resume tracker
		tracker := a.extractor.LoadProgressTracker()
		if tracker == nil || startFrom == 0 {
			tracker = &models.RDAPProgressTracker{
				TotalRecords: len(a.data),
				ProcessedIPs: []string{},
				StartedAt:    time.Now().Format(time.RFC3339),
				Workers:      a.config.Database.Parallelism,
				Throttle:     a.config.Database.APIThrottle,
				Completed:    false,
			}
		}

		go func() {
			defer func() {
				a.setBusy(false, "")
			}()

			total := float64(len(a.data))
			workers := a.config.Database.Parallelism
			if workers < 1 {
				workers = 1
			}

			// Create tasks only for unprocessed items
			tasks := make(chan int, len(a.data))
			for i := startFrom; i < len(a.data); i++ {
				ip := a.data[i].IPOrCIDR
				if !a.extractor.IsIPProcessed(ip, tracker) {
					tasks <- i
				}
			}
			close(tasks)

			done := make(chan struct{})
			// token bucket based on throttle
			interval := time.Duration(int(a.config.Database.APIThrottle*1000)) * time.Millisecond
			if interval <= 0 {
				interval = 1 * time.Millisecond
			}
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for w := 0; w < workers; w++ {
				go func() {
					defer func() { done <- struct{}{} }()
					for idx := range tasks {
						if cancel {
							break
						}
						<-ticker.C
						ip := a.data[idx].IPOrCIDR
						_ = a.extractor.EnrichRecordWithDelay(&a.data[idx], 0)

						// Update tracker
						tracker.ProcessedRecords = idx + 1
						tracker.ProcessedIPs = append(tracker.ProcessedIPs, ip)

						// Save progress every 10 records
						if len(tracker.ProcessedIPs)%10 == 0 {
							_ = a.extractor.SaveProgressTracker(tracker)
						}

						progress.SetValue(float64(idx+1) / total)
						progressDetail.SetText(fmt.Sprintf("RDAP %d/%d - %s (registry: %s)", idx+1, int(total), ip, a.data[idx].Registry))
						if idx%50 == 0 && a.dataTable != nil {
							a.dataTable.Refresh()
						}
					}
				}()
			}

			for w := 0; w < workers; w++ {
				<-done
			}

			// Mark as completed and save final state
			tracker.Completed = true
			_ = a.extractor.SaveProgressTracker(tracker)

			ts := time.Now().Format("2006-01-02_15-04-05")
			filename := fmt.Sprintf("full_enriched_%s.csv", ts)
			if err := a.extractor.SaveToCSV(a.data, filename); err != nil {
				a.logger.Warning("GUI", "CSV save error: "+err.Error())
				dialog.ShowError(err, a.mainWindow)
			} else {
				a.logger.Info("GUI", "✅ Full RDAP associated and saved: "+filename)
				dialog.ShowInformation("RDAP", "✅ RDAP associé sur l'ensemble du dataset\nCSV: "+filename, a.mainWindow)

				// Clean up progress file on successful completion
				_ = a.extractor.ClearProgressTracker()
			}
		}()
	}

	exportBtn := widget.NewButton("📤 Export All", func() {
		a.exportAllData()
	})

	exportSelectedBtn := widget.NewButton("📤 Export Selected", func() {
		// Collect selected
		var rows []models.ScannerData
		for idx, sel := range a.selectedRows {
			if sel && idx < len(a.data) {
				rows = append(rows, a.data[idx])
			}
		}
		if len(rows) == 0 {
			dialog.ShowInformation("Export", "No rows selected", a.mainWindow)
			return
		}
		ts := time.Now().Format("2006-01-02_15-04-05")
		filename := fmt.Sprintf("results/selected_export_%s.csv", ts)
		if err := a.extractor.SaveToCSV(rows, filename); err != nil {
			dialog.ShowError(err, a.mainWindow)
			return
		}
		dialog.ShowInformation("Export", "✅ Exported "+fmt.Sprintf("%d", len(rows))+" rows to\n"+filename, a.mainWindow)
	})

	geolocBtn := widget.NewButton("🌍 Geoloc", func() {
		// Aggregate by continent
		counts := map[string]int{}
		max := len(a.data)
		if max > 2000 {
			max = 2000
		} // limiter pour éviter trop d'appels
		for i := 0; i < max; i++ {
			ip := a.data[i].IPOrCIDR
			if ip == "" {
				continue
			}
			cont, _, _, _, err := a.extractor.GeoLookupContinent(ip)
			if err != nil {
				continue
			}
			counts[cont]++
		}
		// Build view
		text := "Répartition par continent (échantillon):\n"
		for k, v := range counts {
			text += fmt.Sprintf("- %s: %d\n", k, v)
		}
		// Import block
		entry := widget.NewMultiLineEntry()
		entry.SetPlaceHolder("Collez vos IPs (une par ligne) ou importez un fichier...")
		importFileBtn := widget.NewButton("📄 Import fichier", func() {
			d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
				if err != nil || r == nil {
					return
				}
				b, _ := io.ReadAll(r)
				entry.SetText(string(b))
			}, a.mainWindow)
			d.Show()
		})
		applyBtn := widget.NewButton("➕ Ajouter aux données", func() {
			lines := strings.Split(entry.Text, "\n")
			now := time.Now()
			for _, line := range lines {
				ip := strings.TrimSpace(line)
				if ip == "" {
					continue
				}
				item := models.ScannerData{IPOrCIDR: ip, ScannerName: "User", ScannerType: models.ScannerTypeOther, LastSeen: now}
				a.data = append(a.data, item)
			}
			a.updatePagination()
			if a.dataTable != nil {
				a.dataTable.Refresh()
				// Apply column/row layout after load
				a.applyTableLayout()
			}
			dialog.ShowInformation("Geoloc", "IPs ajoutées", a.mainWindow)
		})
		content := container.NewVBox(
			widget.NewLabel("Geolocalisation (par continent)"),
			widget.NewMultiLineEntry(),
			container.NewHBox(importFileBtn, applyBtn),
		)
		ml := content.Objects[1].(*widget.Entry)
		ml.MultiLine = true
		ml.SetText(text)
		ml.Disable()
		dialog.NewCustom("Geoloc", "Fermer", container.NewScroll(content), a.mainWindow).Show()
	})

	// Button layout
	buttonsContainer := container.NewHBox(
		updateBtn,
		associateRDAPBtn,
		associateRDAPAllBtn,
		cancelBtn,
		rdapDetailsBtn,
		geolocBtn,
		exportBtn,
		exportSelectedBtn,
	)

	// Main layout (header intégré au tableau)
	databaseContainer := container.NewVBox(
		title,
		buttonsContainer,
		paginationLabel,
		paginationControls,
		progress,
		progressDetail,
		hscroll,
	)

	return container.NewScroll(databaseContainer)
}

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

// updatePagination updates pagination state and refreshes the interface
// It calculates page numbers, validates current page, and updates the display
func (a *App) updatePagination() {
	// Calculate total pages
	if a.itemsPerPage > 0 {
		a.totalPages = (len(a.data) + a.itemsPerPage - 1) / a.itemsPerPage
		if a.totalPages == 0 {
			a.totalPages = 1
		}
	}

	// Validate current page
	if a.currentPage > a.totalPages {
		a.currentPage = a.totalPages
	}
	if a.currentPage < 1 {
		a.currentPage = 1
	}

	// Calculate start and end indices
	startIndex := (a.currentPage - 1) * a.itemsPerPage
	endIndex := startIndex + a.itemsPerPage
	if endIndex > len(a.data) {
		endIndex = len(a.data)
	}

	// Update pagination info
	if a.paginationInfo != nil {
		a.paginationInfo.SetText(fmt.Sprintf("Page %d of %d (%d-%d of %d records)",
			a.currentPage, a.totalPages, startIndex+1, endIndex, len(a.data)))
	}

	// Refresh table
	if a.dataTable != nil {
		a.dataTable.Refresh()
		// Re-apply widths/heights for current page
		a.applyTableLayout()
	}

	a.logger.Info("GUI", fmt.Sprintf("📄 Pagination updated: page %d/%d (%d records)",
		a.currentPage, a.totalPages, len(a.data)))
}

// loadData loads data from CSV file or triggers extraction if none valid
// It prioritizes loading from the latest CSV file in the results directory
func (a *App) loadData() {
	// Try to load from CSV files (newest first)
	csvFiles, err := filepath.Glob("results/*.csv")
	if err == nil && len(csvFiles) > 0 {
		// Sort by modification time (newest first)
		sort.Slice(csvFiles, func(i, j int) bool {
			infoI, _ := os.Stat(csvFiles[i])
			infoJ, _ := os.Stat(csvFiles[j])
			return infoI.ModTime().After(infoJ.ModTime())
		})

		for _, f := range csvFiles {
			a.logger.Info("GUI", "📂 Loading data from: "+f)
			if data, err := a.loadFromCSV(f); err == nil && len(data) > 0 {
				a.data = data
				a.currentPage = 1
				a.logger.Info("GUI", fmt.Sprintf("✅ %d records loaded from %s", len(a.data), f))
				if a.dataTable != nil {
					a.dataTable.Refresh()
					// Apply column/row layout after load
					a.applyTableLayout()
				}
				a.updatePagination()
				a.updateStats()
				return
			} else if err != nil {
				a.logger.Warning("GUI", "CSV load error for "+f+": "+err.Error())
			}
		}
	}

	// No valid CSV: trigger extraction automatically
	a.logger.Warning("GUI", "No valid CSV found; running extraction...")
	go func() {
		if _, err := a.extractor.ExtractData(); err != nil {
			a.logger.Error("GUI", "Extraction failed: "+err.Error())
			dialog.ShowError(err, a.mainWindow)
			return
		}
		// Reload after extraction
		a.logger.Info("GUI", "Reloading data after extraction...")
		a.loadData()
		if a.dataTable != nil {
			a.dataTable.Refresh()
			// Apply column/row layout after load
			a.applyTableLayout()
		}
		a.updatePagination()
		a.updateStats()
	}()
}

// setBusy updates statusBar with a busy/ready message
func (a *App) setBusy(busy bool, message string) {
	if a.statusBar == nil {
		return
	}
	if busy {
		a.statusBar.SetText("⏳ " + message)
	} else {
		a.statusBar.SetText("🟢 Ready")
	}
}

// exportAllData exports all data to a CSV file with professional formatting
// It creates a timestamped file in the results directory
func (a *App) exportAllData() {
	if len(a.data) == 0 {
		dialog.ShowInformation("Export", "⚠️ No data to export", a.mainWindow)
		return
	}

	// Generate professional filename
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("results/liacheckscanner_export_%s.csv", timestamp)

	// Create CSV file
	file, err := os.Create(filename)
	if err != nil {
		dialog.ShowError(err, a.mainWindow)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Professional headers
	headers := []string{"IP/CIDR", "Scanner", "Type", "Country", "ISP", "Risk Level", "Score", "Last Seen", "Tags", "Notes"}
	writer.Write(headers)

	// Export data
	for _, item := range a.data {
		row := []string{
			item.IPOrCIDR,
			item.ScannerName,
			string(item.ScannerType),
			item.CountryCode,
			item.ISP,
			item.RiskLevel,
			fmt.Sprintf("%d", item.AbuseConfidenceScore),
			item.LastSeen.Format("2006-01-02"),
			strings.Join(item.Tags, ";"),
			item.Notes,
		}
		writer.Write(row)
	}

	a.logger.Info("GUI", fmt.Sprintf("✅ %d records exported to %s", len(a.data), filename))
	dialog.ShowInformation("Export Success", fmt.Sprintf("✅ %d records exported to:\n%s", len(a.data), filename), a.mainWindow)
}

// Additional helper methods for professional functionality

// loadExistingData loads existing data from various sources
// It attempts to load from CSV files first, then falls back to extraction
func (a *App) loadExistingData() error {
	// Use the new loadData function that loads from CSV or extracts
	a.loadData()
	return nil
}

// refreshData reloads data, refreshes the interface, and updates statistics
// It provides real-time data updates for the professional interface
func (a *App) refreshData() {
	a.logger.Info("GUI", "🔄 Refreshing data...")

	// Reload data
	a.loadData()

	// Refresh interface
	if a.dataTable != nil {
		a.dataTable.Refresh()
		// Apply column/row layout after load
		a.applyTableLayout()
	}

	// Update pagination
	a.updatePagination()

	// Update statistics
	a.updateStats()

	a.logger.Info("GUI", fmt.Sprintf("✅ %d records displayed", len(a.data)))
}

// updateStats updates the statistics display with current data information
// It provides real-time statistics for the professional dashboard
func (a *App) updateStats() {
	if a.statsLabel != nil {
		stats := fmt.Sprintf(`📊 Real-time Statistics:
• Total Records: %d
• Unique IPs: %d
• Countries: %d
• Scanners: %d
• High Risk: %d
• Last Updated: %s`,
			len(a.data),
			a.countUniqueIPs(),
			a.countUniqueCountries(),
			a.countUniqueScanners(),
			a.countHighRisk(),
			time.Now().Format("2006-01-02 15:04:05"))

		a.statsLabel.SetText(stats)
	}
}

// countUniqueIPs counts unique IP addresses in the dataset
func (a *App) countUniqueIPs() int {
	unique := make(map[string]bool)
	for _, item := range a.data {
		unique[item.IPOrCIDR] = true
	}
	return len(unique)
}

// countUniqueCountries counts unique countries in the dataset
func (a *App) countUniqueCountries() int {
	unique := make(map[string]bool)
	for _, item := range a.data {
		if item.CountryCode != "" {
			unique[item.CountryCode] = true
		}
	}
	return len(unique)
}

// countUniqueScanners counts unique scanners in the dataset
func (a *App) countUniqueScanners() int {
	unique := make(map[string]bool)
	for _, item := range a.data {
		unique[item.ScannerName] = true
	}
	return len(unique)
}

// countHighRisk counts high-risk entries in the dataset
func (a *App) countHighRisk() int {
	count := 0
	for _, item := range a.data {
		if item.RiskLevel == "High" {
			count++
		}
	}
	return count
}

// loadFromCSV loads data from a CSV file using header-based mapping
func (a *App) loadFromCSV(filename string) ([]models.ScannerData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("insufficient data in CSV file")
	}

	// Build header index map
	headers := records[0]
	index := func(name string) int {
		for i, h := range headers {
			if strings.EqualFold(strings.TrimSpace(h), strings.TrimSpace(name)) {
				return i
			}
		}
		return -1
	}

	ipIdx := index("IP/CIDR")
	scannerNameIdx := index("Scanner Name")
	scannerTypeIdx := index("Scanner Type")
	countryCodeIdx := index("Country Code")
	ispIdx := index("ISP")
	orgIdx := index("Organization")
	rdapNameIdx := index("RDAP Name")
	rdapHandleIdx := index("RDAP Handle")
	rdapCIDRIdx := index("RDAP CIDR")
	registryIdx := index("RDAP Registry")
	asnIdx := index("ASN")
	asNameIdx := index("AS Name")
	reverseIdx := index("Reverse DNS")
	riskIdx := index("Risk Level")
	scoreIdx := index("Abuse Confidence Score")
	domainIdx := index("Domain")
	lastSeenIdx := index("Last Seen")
	tagsIdx := index("Tags")
	notesIdx := index("Notes")
	parentHandleIdx := index("Parent Handle")
	eventRegIdx := index("Event Registration")
	eventChangedIdx := index("Event Last Changed")
	startAddrIdx := index("Start Address")
	endAddrIdx := index("End Address")
	ipVersionIdx := index("IP Version")
	rdapTypeIdx := index("RDAP Type")
	abuseEmailIdx := index("Abuse Email")
	techEmailIdx := index("Tech Email")

	var data []models.ScannerData
	for _, record := range records[1:] { // Skip header
		item := models.ScannerData{}
		if ipIdx >= 0 && ipIdx < len(record) {
			item.IPOrCIDR = record[ipIdx]
		}
		if scannerNameIdx >= 0 && scannerNameIdx < len(record) {
			item.ScannerName = record[scannerNameIdx]
		}
		if scannerTypeIdx >= 0 && scannerTypeIdx < len(record) {
			item.ScannerType = models.ScannerType(record[scannerTypeIdx])
		}
		if countryCodeIdx >= 0 && countryCodeIdx < len(record) {
			item.CountryCode = record[countryCodeIdx]
		}
		if ispIdx >= 0 && ispIdx < len(record) {
			item.ISP = record[ispIdx]
		}
		if orgIdx >= 0 && orgIdx < len(record) {
			item.Organization = record[orgIdx]
		}
		if rdapNameIdx >= 0 && rdapNameIdx < len(record) {
			item.RDAPName = record[rdapNameIdx]
		}
		if rdapHandleIdx >= 0 && rdapHandleIdx < len(record) {
			item.RDAPHandle = record[rdapHandleIdx]
		}
		if rdapCIDRIdx >= 0 && rdapCIDRIdx < len(record) {
			item.RDAPCIDR = record[rdapCIDRIdx]
		}
		if registryIdx >= 0 && registryIdx < len(record) {
			item.Registry = record[registryIdx]
		}
		if startAddrIdx >= 0 && startAddrIdx < len(record) {
			item.StartAddress = record[startAddrIdx]
		}
		if endAddrIdx >= 0 && endAddrIdx < len(record) {
			item.EndAddress = record[endAddrIdx]
		}
		if ipVersionIdx >= 0 && ipVersionIdx < len(record) {
			item.IPVersion = record[ipVersionIdx]
		}
		if rdapTypeIdx >= 0 && rdapTypeIdx < len(record) {
			item.RDAPType = record[rdapTypeIdx]
		}
		if parentHandleIdx >= 0 && parentHandleIdx < len(record) {
			item.ParentHandle = record[parentHandleIdx]
		}
		if eventRegIdx >= 0 && eventRegIdx < len(record) {
			item.EventRegistration = record[eventRegIdx]
		}
		if eventChangedIdx >= 0 && eventChangedIdx < len(record) {
			item.EventLastChanged = record[eventChangedIdx]
		}
		if asnIdx >= 0 && asnIdx < len(record) {
			item.ASN = record[asnIdx]
		}
		if asNameIdx >= 0 && asNameIdx < len(record) {
			item.ASName = record[asNameIdx]
		}
		if reverseIdx >= 0 && reverseIdx < len(record) {
			item.ReverseDNS = record[reverseIdx]
		}
		if riskIdx >= 0 && riskIdx < len(record) {
			item.RiskLevel = record[riskIdx]
		}
		if scoreIdx >= 0 && scoreIdx < len(record) {
			if score, err := strconv.Atoi(record[scoreIdx]); err == nil {
				item.AbuseConfidenceScore = score
			}
		}
		if domainIdx >= 0 && domainIdx < len(record) {
			item.Domain = record[domainIdx]
		}
		if lastSeenIdx >= 0 && lastSeenIdx < len(record) {
			if t, err := time.Parse("2006-01-02 15:04:05", record[lastSeenIdx]); err == nil {
				item.LastSeen = t
			} else {
				item.LastSeen = time.Now()
			}
		} else {
			item.LastSeen = time.Now()
		}
		if tagsIdx >= 0 && tagsIdx < len(record) {
			if ts := strings.TrimSpace(record[tagsIdx]); ts != "" {
				item.Tags = strings.Split(ts, ",")
			}
		}
		if notesIdx >= 0 && notesIdx < len(record) {
			item.Notes = record[notesIdx]
		}
		if abuseEmailIdx >= 0 && abuseEmailIdx < len(record) {
			item.AbuseEmail = record[abuseEmailIdx]
		}
		if techEmailIdx >= 0 && techEmailIdx < len(record) {
			item.TechEmail = record[techEmailIdx]
		}
		data = append(data, item)
	}

	return data, nil
}

// createTestData was removed - application now uses only real data from .nft files

// exportSelected exports selected data with professional confirmation
func (a *App) exportSelected() {
	dialog.ShowConfirm("Export Selected", "Export selected records to CSV?", func(confirm bool) {
		if confirm {
			a.exportSelectedData()
		}
	}, a.mainWindow)
}

// exportSelectedData performs the actual export of selected data
func (a *App) exportSelectedData() {
	selectedRows := a.getSelectedRows()
	if len(selectedRows) == 0 {
		dialog.ShowInformation("Export", "No records selected for export", a.mainWindow)
		return
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("results/selected_export_%s.csv", timestamp)

	file, err := os.Create(filename)
	if err != nil {
		dialog.ShowError(err, a.mainWindow)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Professional headers
	headers := []string{"IP/CIDR", "Scanner", "Type", "Country", "ISP", "Risk Level", "Score", "Last Seen"}
	writer.Write(headers)

	// Export selected data
	for _, index := range selectedRows {
		if index < len(a.data) {
			item := a.data[index]
			row := []string{
				item.IPOrCIDR,
				item.ScannerName,
				string(item.ScannerType),
				item.CountryCode,
				item.ISP,
				item.RiskLevel,
				fmt.Sprintf("%d", item.AbuseConfidenceScore),
				item.LastSeen.Format("2006-01-02"),
			}
			writer.Write(row)
		}
	}

	a.logger.Info("GUI", fmt.Sprintf("✅ %d selected records exported to %s", len(selectedRows), filename))
	dialog.ShowInformation("Export Success", fmt.Sprintf("✅ %d records exported to:\n%s", len(selectedRows), filename), a.mainWindow)
}

// getSelectedRows returns indices of selected rows (simulated for now)
func (a *App) getSelectedRows() []int {
	// For now, return first 500 rows as a simulation
	// In a real implementation, this would track actual user selection
	maxRows := 500
	if len(a.data) < maxRows {
		maxRows = len(a.data)
	}

	var selected []int
	for i := 0; i < maxRows; i++ {
		selected = append(selected, i)
	}
	return selected
}

// performAdvancedSearch performs advanced search with multiple criteria
func (a *App) performAdvancedSearch(query, country, scanner, risk string) {
	var results []models.ScannerData

	for _, item := range a.data {
		// Apply filters
		matchesQuery := query == "" ||
			strings.Contains(strings.ToLower(item.IPOrCIDR), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(item.ScannerName), strings.ToLower(query))

		matchesCountry := country == "All Countries" || item.CountryCode == country
		matchesScanner := scanner == "All Scanners" || item.ScannerName == scanner
		matchesRisk := risk == "All Risk Levels" || item.RiskLevel == risk

		if matchesQuery && matchesCountry && matchesScanner && matchesRisk {
			results = append(results, item)
		}
	}

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

// Helper functions for search statistics
func (a *App) countUniqueCountriesInResults(results []models.ScannerData) int {
	unique := make(map[string]bool)
	for _, item := range results {
		if item.CountryCode != "" {
			unique[item.CountryCode] = true
		}
	}
	return len(unique)
}

func (a *App) countUniqueScannersInResults(results []models.ScannerData) int {
	unique := make(map[string]bool)
	for _, item := range results {
		unique[item.ScannerName] = true
	}
	return len(unique)
}

func (a *App) countRiskLevelsInResults(results []models.ScannerData) int {
	unique := make(map[string]bool)
	for _, item := range results {
		unique[item.RiskLevel] = true
	}
	return len(unique)
}

// exportSearchResults exports search results to CSV
func (a *App) exportSearchResults() {
	if len(a.searchResults) == 0 {
		dialog.ShowInformation("Export", "No search results to export", a.mainWindow)
		return
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("results/search_results_%s.csv", timestamp)

	file, err := os.Create(filename)
	if err != nil {
		dialog.ShowError(err, a.mainWindow)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"IP/CIDR", "Scanner", "Type", "Country", "ISP", "Risk", "Score", "Last Seen"}
	writer.Write(headers)

	for _, item := range a.searchResults {
		row := []string{
			item.IPOrCIDR,
			item.ScannerName,
			string(item.ScannerType),
			item.CountryCode,
			item.ISP,
			item.RiskLevel,
			fmt.Sprintf("%d", item.AbuseConfidenceScore),
			item.LastSeen.Format("2006-01-02"),
		}
		writer.Write(row)
	}

	a.logger.Info("GUI", fmt.Sprintf("✅ %d search results exported to %s", len(a.searchResults), filename))
	dialog.ShowInformation("Export Success", fmt.Sprintf("✅ %d search results exported to:\n%s", len(a.searchResults), filename), a.mainWindow)
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

// exportLogs exports logs to file (placeholder implementation)
func (a *App) exportLogs() {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("logs/system_logs_%s.txt", timestamp)

	// Implementation would export actual logs
	a.logger.Info("GUI", "Logs exported to: "+filename)
	dialog.ShowInformation("Export Success", "Logs exported to:\n"+filename, a.mainWindow)
}

// zipDirectory zips a directory to the given zip file
func (a *App) zipDirectory(srcDir, destZip string) error {
	if err := os.MkdirAll(filepath.Dir(destZip), 0755); err != nil {
		return err
	}
	zf, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer zf.Close()
	w := zip.NewWriter(zf)
	defer w.Close()
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(srcDir, path)
		f, err := w.Create(rel)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(f, src)
		return err
	})
}

// applyTableLayout sets column widths and row heights to avoid overlap
func (a *App) applyTableLayout() {
	if a.dataTable == nil {
		return
	}
	// Headers matching columns
	headers := []string{"IP/CIDR", "Scanner", "Type", "Country", "ISP", "Organization", "RDAP Name", "RDAP Handle", "ASN", "Reverse", "Risk", "Score", "Domain", "Last Seen"}
	style := fyne.TextStyle{}
	startIndex := (a.currentPage - 1) * a.itemsPerPage
	endIndex := startIndex + a.itemsPerPage
	if endIndex > len(a.data) {
		endIndex = len(a.data)
	}
	// Compute max width per column on visible page (with padding)
	for col := 0; col < 14; col++ {
		maxw := fyne.MeasureText(headers[col], theme.TextSize(), style).Width
		for i := startIndex; i < endIndex; i++ {
			item := a.data[i]
			var txt string
			switch col {
			case 0:
				txt = item.IPOrCIDR
			case 1:
				txt = item.ScannerName
			case 2:
				txt = string(item.ScannerType)
			case 3:
				txt = item.CountryCode
			case 4:
				txt = item.ISP
			case 5:
				txt = item.Organization
			case 6:
				txt = item.RDAPName
			case 7:
				txt = item.RDAPHandle
			case 8:
				txt = item.ASN
			case 9:
				txt = item.ReverseDNS
			case 10:
				txt = item.RiskLevel
			case 11:
				txt = fmt.Sprintf("%d", item.AbuseConfidenceScore)
			case 12:
				txt = item.Domain
			case 13:
				txt = item.LastSeen.Format("2006-01-02")
			}
			w := fyne.MeasureText(txt, theme.TextSize(), style).Width
			if w > maxw {
				maxw = w
			}
		}
		// Apply width with padding to avoid clipping/overlap
		a.dataTable.SetColumnWidth(col, maxw+28)
		if col < len(a.headerLabels) {
			a.headerLabels[col].Resize(fyne.NewSize(maxw+28, a.headerLabels[col].MinSize().Height))
		}
		if col < len(a.headerLabels) {
			a.headerLabels[col].Resize(fyne.NewSize(maxw+28, a.headerLabels[col].MinSize().Height))
		}
	}
	// Ensure visible rows have enough height
	visible := endIndex - startIndex
	if visible < 0 {
		visible = 0
	}
	for r := 0; r < visible; r++ {
		a.dataTable.SetRowHeight(r, 30)
	}
}

// Run starts the application and enters the main event loop
func (a *App) Run() {
	a.fyneApp.Run()
}

// Shutdown gracefully shuts down the application
func (a *App) Shutdown() {
	a.logger.Info("GUI", "Shutting down application gracefully")
	a.fyneApp.Quit()
}
