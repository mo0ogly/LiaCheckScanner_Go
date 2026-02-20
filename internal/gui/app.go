// Package gui provides the graphical user interface for LiaCheckScanner.
// It handles all UI components, data display, and user interactions.
package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/lia/liacheckscanner_go/internal/extractor"
	"github.com/lia/liacheckscanner_go/internal/logger"
	"github.com/lia/liacheckscanner_go/internal/models"
)

// App represents the main application structure, managing the GUI, data, and user interactions.
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

// NewApp creates a new App instance, initializing the GUI window, extractor, and user interface.
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

// updatePagination updates pagination state and refreshes the interface
// It calculates page numbers, validates current page, and updates the display
func (a *App) updatePagination() {
	totalPages, validPage, startIndex, endIndex := CalculatePagination(
		len(a.data), a.itemsPerPage, a.currentPage,
	)
	a.totalPages = totalPages
	a.currentPage = validPage

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
func (a *App) countUniqueIPs() int { return CountUniqueIPs(a.data) }

// countUniqueCountries counts unique countries in the dataset
func (a *App) countUniqueCountries() int { return CountUniqueCountries(a.data) }

// countUniqueScanners counts unique scanners in the dataset
func (a *App) countUniqueScanners() int { return CountUniqueScanners(a.data) }

// countHighRisk counts high-risk entries in the dataset
func (a *App) countHighRisk() int { return CountHighRisk(a.data) }

// loadFromCSV loads data from a CSV file using header-based mapping
func (a *App) loadFromCSV(filename string) ([]models.ScannerData, error) {
	return LoadCSVData(filename)
}

// Run starts the application and enters the main event loop.
func (a *App) Run() {
	a.fyneApp.Run()
}

// Shutdown gracefully shuts down the application.
func (a *App) Shutdown() {
	a.logger.Info("GUI", "Shutting down application gracefully")
	a.fyneApp.Quit()
}
