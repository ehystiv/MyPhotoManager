package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ── Chiavi preferenze ─────────────────────────────────────────────────────────

const (
	prefInputDir   = "inputDir"
	prefOutputDir  = "outputDir"
	prefDryRun     = "dryRun"
	prefCopyMode   = "copyMode"
	prefStripMeta  = "stripMeta"
	prefModTime    = "modTimeFallback"
	prefCheckDupes = "checkDupes"
	prefRenameOnly = "renameOnly"
	prefCleanDirs  = "cleanDirs"
	prefFolderFmt  = "folderFormat"
	prefFileTpl    = "fileTemplate"
)

// ── logWriter ─────────────────────────────────────────────────────────────────

type logWriter struct {
	mu  sync.Mutex
	buf strings.Builder
	b   binding.String
}

func (lw *logWriter) Write(p []byte) (int, error) {
	lw.mu.Lock()
	lw.buf.Write(p)
	text := lw.buf.String()
	lw.mu.Unlock()
	lw.b.Set(text)
	return len(p), nil
}

func (lw *logWriter) Reset() {
	lw.mu.Lock()
	lw.buf.Reset()
	lw.mu.Unlock()
	lw.b.Set("")
}

// ── Utilità ───────────────────────────────────────────────────────────────────

func pathFormItem(label string, pathBind binding.String, clearBtn fyne.CanvasObject, chooseBtn fyne.CanvasObject) *widget.FormItem {
	pathLabel := widget.NewLabelWithData(pathBind)
	pathLabel.Truncation = fyne.TextTruncateEllipsis
	var row fyne.CanvasObject
	if clearBtn != nil {
		row = container.NewBorder(nil, nil, nil, container.NewHBox(clearBtn, chooseBtn), pathLabel)
	} else {
		row = container.NewBorder(nil, nil, nil, chooseBtn, pathLabel)
	}
	return widget.NewFormItem(label, row)
}

func openInFileBrowser(path string) {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "explorer"
	default:
		cmd = "xdg-open"
	}
	exec.Command(cmd, path).Start() //nolint:errcheck
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	a := app.NewWithID("it.myphoto.manager")
	a.Settings().SetTheme(newPhotoTheme())
	prefs := a.Preferences()
	w := a.NewWindow("MyPhotoManager")
	w.Resize(fyne.NewSize(1060, 620))

	// ── Directory ─────────────────────────────────────────────────────────────

	inputBind := binding.NewString()
	outputBind := binding.NewString()

	var inputDir, outputDir string

	// Carica directory salvate
	if saved := prefs.String(prefInputDir); saved != "" {
		if info, err := os.Stat(saved); err == nil && info.IsDir() {
			inputDir = saved
			inputBind.Set(inputDir)
		}
	}
	if saved := prefs.String(prefOutputDir); saved != "" {
		if info, err := os.Stat(saved); err == nil && info.IsDir() {
			outputDir = saved
			outputBind.Set(outputDir)
		}
	}
	if outputDir == "" {
		outputBind.Set("stessa di input")
	}

	previewBind := binding.NewString()
	previewLabel := widget.NewLabelWithData(previewBind)
	previewLabel.TextStyle = fyne.TextStyle{Italic: true}

	var (
		scanMu     sync.Mutex
		scanCancel context.CancelFunc
	)

	startScan := func(dir string) {
		scanMu.Lock()
		if scanCancel != nil {
			scanCancel()
		}
		ctx, cancel := context.WithCancel(context.Background())
		scanCancel = cancel
		scanMu.Unlock()

		previewBind.Set("Scansione in corso…")
		go func() {
			photos, err := collectPhotos(dir, dir)
			if err != nil || ctx.Err() != nil {
				return
			}
			if len(photos) == 0 {
				previewBind.Set("Nessuna foto trovata.")
				return
			}
			rawCount, noExif := 0, 0
			for _, p := range photos {
				if ctx.Err() != nil {
					return
				}
				if rawExtensions[strings.ToLower(filepath.Ext(p))] {
					rawCount++
				}
				if _, ok := getExifDatetime(p); !ok {
					noExif++
				}
			}
			msg := fmt.Sprintf("%d foto  ·  %d RAW  ·  %d altri", len(photos), rawCount, len(photos)-rawCount)
			if noExif > 0 {
				msg += fmt.Sprintf("  ·  %d senza EXIF", noExif)
			}
			previewBind.Set(msg)
		}()
	}

	// Avvia scansione se la directory era salvata
	if inputDir != "" {
		go startScan(inputDir)
	}

	chooseInputBtn := widget.NewButtonWithIcon("Scegli…", theme.FolderOpenIcon(), func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			inputDir = uri.Path()
			inputBind.Set(inputDir)
			prefs.SetString(prefInputDir, inputDir)
			startScan(inputDir)
		}, w)
	})

	clearOutputBtn := widget.NewButtonWithIcon("", theme.ContentClearIcon(), func() {
		outputDir = ""
		outputBind.Set("stessa di input")
		prefs.SetString(prefOutputDir, "")
	})
	clearOutputBtn.Importance = widget.LowImportance

	chooseOutputBtn := widget.NewButtonWithIcon("Scegli…", theme.FolderOpenIcon(), func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			outputDir = uri.Path()
			outputBind.Set(outputDir)
			prefs.SetString(prefOutputDir, outputDir)
		}, w)
	})

	dropHint := widget.NewLabel("(o trascina una cartella qui)")
	dropHint.TextStyle = fyne.TextStyle{Italic: true}

	dirForm := widget.NewForm(
		pathFormItem("Input", inputBind, nil, chooseInputBtn),
		pathFormItem("Output", outputBind, clearOutputBtn, chooseOutputBtn),
	)
	sorgenteCard := widget.NewCard("Cartelle", "", container.NewVBox(dirForm, previewLabel, dropHint))

	// ── Drag & drop ───────────────────────────────────────────────────────────

	w.SetOnDropped(func(_ fyne.Position, uris []fyne.URI) {
		for _, uri := range uris {
			if uri.Scheme() != "file" {
				continue
			}
			path := uri.Path()
			if info, err := os.Stat(path); err == nil && info.IsDir() {
				fyne.Do(func() {
					inputDir = path
					inputBind.Set(path)
					prefs.SetString(prefInputDir, path)
					startScan(path)
				})
				return
			}
		}
	})

	// ── Opzioni ───────────────────────────────────────────────────────────────

	saveBool := func(key string, val bool) { prefs.SetBool(key, val) }

	dryRunCheck := widget.NewCheck("Dry-run  (anteprima, nessuna modifica)", nil)
	dryRunCheck.SetChecked(prefs.Bool(prefDryRun))
	dryRunCheck.OnChanged = func(v bool) { saveBool(prefDryRun, v) }

	copyModeCheck := widget.NewCheck("Copia  (mantieni gli originali)", nil)
	copyModeCheck.SetChecked(prefs.Bool(prefCopyMode))
	copyModeCheck.OnChanged = func(v bool) { saveBool(prefCopyMode, v) }

	stripMetaCheck := widget.NewCheck("Rimuovi EXIF dai JPEG", nil)
	stripMetaCheck.SetChecked(prefs.Bool(prefStripMeta))
	stripMetaCheck.OnChanged = func(v bool) { saveBool(prefStripMeta, v) }

	modTimeCheck := widget.NewCheck("Data da filesystem se EXIF mancante", nil)
	modTimeCheck.SetChecked(prefs.Bool(prefModTime))
	modTimeCheck.OnChanged = func(v bool) { saveBool(prefModTime, v) }

	dupesCheck := widget.NewCheck("Salta duplicati  (SHA-256)", nil)
	dupesCheck.SetChecked(prefs.Bool(prefCheckDupes))
	dupesCheck.OnChanged = func(v bool) { saveBool(prefCheckDupes, v) }

	renameOnlyCheck := widget.NewCheck("Rinomina in-place  (non sposta)", nil)
	renameOnlyCheck.SetChecked(prefs.Bool(prefRenameOnly))

	cleanDirsCheck := widget.NewCheck("Rimuovi cartelle vuote", nil)
	cleanDirsCheck.SetChecked(prefs.Bool(prefCleanDirs))
	cleanDirsCheck.OnChanged = func(v bool) { saveBool(prefCleanDirs, v) }

	// ── Template struttura ────────────────────────────────────────────────────

	folderFmtEntry := widget.NewEntry()
	savedFolderFmt := prefs.StringWithFallback(prefFolderFmt, "2006_01_02")
	folderFmtEntry.SetText(savedFolderFmt)
	folderFmtEntry.SetPlaceHolder("formato Go time, es. 2006/01/02")
	folderFmtEntry.OnChanged = func(v string) { prefs.SetString(prefFolderFmt, v) }

	fileTplEntry := widget.NewEntry()
	savedFileTpl := prefs.StringWithFallback(prefFileTpl, "photo_{date}_{time}")
	fileTplEntry.SetText(savedFileTpl)
	fileTplEntry.SetPlaceHolder("{date} {time} {datetime} {year} {month} {day} {camera}")
	fileTplEntry.OnChanged = func(v string) { prefs.SetString(prefFileTpl, v) }

	tokenHint := widget.NewLabel("Token: {date}  {time}  {datetime}  {year}  {month}  {day}  {camera}")
	tokenHint.TextStyle = fyne.TextStyle{Italic: true}

	// Rename-only disabilita folder format e copyMode (irrilevanti)
	renameOnlyCheck.OnChanged = func(v bool) {
		saveBool(prefRenameOnly, v)
		if v {
			folderFmtEntry.Disable()
			copyModeCheck.Disable()
			cleanDirsCheck.Disable()
		} else {
			folderFmtEntry.Enable()
			copyModeCheck.Enable()
			cleanDirsCheck.Enable()
		}
	}
	if renameOnlyCheck.Checked {
		folderFmtEntry.Disable()
		copyModeCheck.Disable()
		cleanDirsCheck.Disable()
	}

	opzioniCard := widget.NewCard("Opzioni", "", container.NewGridWithColumns(2,
		dryRunCheck, copyModeCheck,
		stripMetaCheck, modTimeCheck,
		dupesCheck, renameOnlyCheck,
		cleanDirsCheck, widget.NewLabel(""),
	))

	strutturaCard := widget.NewCard("Struttura", "", container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Cartelle", folderFmtEntry),
			widget.NewFormItem("Nome file", fileTplEntry),
		),
		tokenHint,
	))

	// ── Watch ─────────────────────────────────────────────────────────────────

	var watcher *Watcher
	watchStatusBind := binding.NewString()
	watchStatusLabel := widget.NewLabelWithData(watchStatusBind)
	watchStatusLabel.TextStyle = fyne.TextStyle{Italic: true}

	// ── Log & statistiche ─────────────────────────────────────────────────────

	logBind := binding.NewString()
	lw := &logWriter{b: logBind}

	logEntry := widget.NewEntryWithData(logBind)
	logEntry.MultiLine = true
	logEntry.Disable()
	logCard := widget.NewCard("Log", "", container.NewScroll(logEntry))

	statsLabel := widget.NewLabel("")
	openBtn := widget.NewButtonWithIcon("Apri cartella nel Finder", theme.FolderOpenIcon(), nil)
	openBtn.Hide()
	statsCard := widget.NewCard("Statistiche per anno", "", container.NewVBox(statsLabel, openBtn))
	statsCard.Hide()

	var lastOutputDir string

	showStats := func(stats OrganizerStats, outDir string) {
		lastOutputDir = outDir
		fyne.Do(func() {
			openBtn.OnTapped = func() { openInFileBrowser(lastOutputDir) }
			openBtn.Show()

			if len(stats.ByYear) == 0 {
				statsCard.Hide()
				return
			}
			years := make([]int, 0, len(stats.ByYear))
			for y := range stats.ByYear {
				years = append(years, y)
			}
			sort.Sort(sort.Reverse(sort.IntSlice(years)))
			var sb strings.Builder
			for _, y := range years {
				sb.WriteString(fmt.Sprintf("%d  →  %d foto\n", y, stats.ByYear[y]))
			}
			statsLabel.SetText(strings.TrimRight(sb.String(), "\n"))
			statsCard.Show()
		})
	}

	// ── Avanzamento ───────────────────────────────────────────────────────────

	progressBar := widget.NewProgressBar()
	progressBar.Hide()
	progressInfoBind := binding.NewString()
	progressInfoLabel := widget.NewLabelWithData(progressInfoBind)
	progressInfoLabel.Hide()

	onProgress := func(cur, tot int, name string) {
		fyne.Do(func() {
			progressBar.SetValue(float64(cur) / float64(tot))
			progressInfoBind.Set(fmt.Sprintf("%d / %d  —  %s", cur, tot, name))
		})
	}

	// ── Pulsanti ──────────────────────────────────────────────────────────────

	startBtn := widget.NewButtonWithIcon("  Avvia  ", theme.MediaPlayIcon(), nil)
	startBtn.Importance = widget.HighImportance

	stopBtn := widget.NewButtonWithIcon("Stop", theme.MediaStopIcon(), nil)
	stopBtn.Importance = widget.DangerImportance
	stopBtn.Disable()

	var (
		opMu     sync.Mutex
		opCancel context.CancelFunc
	)

	cancelCurrentOp := func() {
		opMu.Lock()
		if opCancel != nil {
			opCancel()
			opCancel = nil
		}
		opMu.Unlock()
	}

	stopWatcher := func() {
		if watcher != nil {
			watcher.Stop()
			watcher = nil
		}
		watchStatusBind.Set("")
	}

	getOpts := func() OrganizerOptions {
		return OrganizerOptions{
			DryRun:          dryRunCheck.Checked,
			StripMeta:       stripMetaCheck.Checked,
			CopyMode:        copyModeCheck.Checked,
			ModTimeFallback: modTimeCheck.Checked,
			CheckDupes:      dupesCheck.Checked,
			RenameOnly:      renameOnlyCheck.Checked,
			CleanEmptyDirs:  cleanDirsCheck.Checked,
			FolderFormat:    folderFmtEntry.Text,
			FileTemplate:    fileTplEntry.Text,
		}
	}

	watchCheck := widget.NewCheck("Monitora cartella automaticamente  (ogni 10 s)", func(checked bool) {
		if checked {
			if inputDir == "" {
				dialog.ShowError(&valErr{"Seleziona prima una directory di input."}, w)
				watchStatusBind.Set("")
				return
			}
			outDir := outputDir
			if outDir == "" {
				outDir = inputDir
			}
			lw.Reset()
			statsCard.Hide()
			openBtn.Hide()
			startBtn.Disable()
			stopBtn.Enable()

			watcher = StartWatch(inputDir, outDir, getOpts(), lw, onProgress, func(stats OrganizerStats) {
				watchStatusBind.Set(fmt.Sprintf("Ultima scansione: %d file elaborati", stats.Moved+stats.Skipped))
				showStats(stats, outDir)
			})
			watchStatusBind.Set("Watch attivo — in attesa di nuovi file…")
		} else {
			stopWatcher()
			startBtn.Enable()
			stopBtn.Disable()
		}
	})

	watchCard := widget.NewCard("Watch", "", container.NewVBox(watchCheck, watchStatusLabel))

	stopBtn.OnTapped = func() {
		cancelCurrentOp()
		if watcher != nil {
			stopWatcher()
			watchCheck.SetChecked(false)
		}
		fyne.Do(func() {
			startBtn.Enable()
			stopBtn.Disable()
			progressBar.Hide()
			progressInfoLabel.Hide()
		})
	}

	startBtn.OnTapped = func() {
		if inputDir == "" {
			dialog.ShowError(&valErr{"Seleziona prima una directory di input."}, w)
			return
		}
		lw.Reset()
		statsCard.Hide()
		openBtn.Hide()
		startBtn.Disable()
		stopBtn.Enable()
		progressBar.SetValue(0)
		progressBar.Show()
		progressInfoBind.Set("")
		progressInfoLabel.Show()

		ctx, cancel := context.WithCancel(context.Background())
		opMu.Lock()
		opCancel = cancel
		opMu.Unlock()

		outDir := outputDir
		if outDir == "" {
			outDir = inputDir
		}
		opts := getOpts()

		go func() {
			stats, err := organizePhotos(ctx, inputDir, outDir, opts, onProgress, lw)
			fyne.Do(func() {
				startBtn.Enable()
				stopBtn.Disable()
				progressBar.Hide()
				progressInfoLabel.Hide()
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				showStats(stats, outDir)
				// Notifica di sistema
				if !opts.DryRun && ctx.Err() == nil {
					msg := fmt.Sprintf("%d file elaborati", stats.Moved+stats.Skipped)
					if stats.Dupes > 0 {
						msg += fmt.Sprintf(", %d duplicati ignorati", stats.Dupes)
					}
					a.SendNotification(&fyne.Notification{
						Title:   "MyPhotoManager — completato",
						Content: msg,
					})
				}
			})
		}()
	}

	// ── Layout ────────────────────────────────────────────────────────────────

	actionRow := container.NewBorder(
		nil, nil, nil,
		container.NewHBox(stopBtn, startBtn),
		container.NewVBox(progressBar, progressInfoLabel),
	)

	controls := container.NewVBox(
		sorgenteCard,
		opzioniCard,
		strutturaCard,
		watchCard,
		actionRow,
	)

	rightPanel := container.NewPadded(container.NewBorder(nil, statsCard, nil, nil, logCard))

	split := container.NewHSplit(
		container.NewPadded(controls),
		rightPanel,
	)
	split.Offset = 0.42

	w.SetContent(split)
	w.ShowAndRun()
}

type valErr struct{ msg string }

func (e *valErr) Error() string { return e.msg }
