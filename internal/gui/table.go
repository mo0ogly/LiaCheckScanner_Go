// Package gui provides the graphical user interface for LiaCheckScanner.
// This file contains table creation, column headers, data binding, and layout.
package gui

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/lia/liacheckscanner_go/internal/models"
)

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
