package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const maxRecents = 8

type Prefs struct {
	InputDir            string   `json:"inputDir"`
	OutputDir           string   `json:"outputDir"`
	DryRun              bool     `json:"dryRun"`
	CopyMode            bool     `json:"copyMode"`
	StripMeta           bool     `json:"stripMeta"`
	ModTime             bool     `json:"modTime"`
	CheckDupes          bool     `json:"checkDupes"`
	RenameOnly          bool     `json:"renameOnly"`
	CleanDirs           bool     `json:"cleanDirs"`
	FolderFmt           string   `json:"folderFmt"`
	FileTpl             string   `json:"fileTpl"`
	RawSplit            string   `json:"rawSplit"`
	Recents             []string `json:"recents"`
	ConfirmedUnsafeOnce bool     `json:"confirmedUnsafeOnce"`
}

type FormatPreviewResult struct {
	Folder string `json:"folder"`
	File   string `json:"file"`
	Full   string `json:"full"`
	Error  string `json:"error,omitempty"`
}

type ScanResult struct {
	Total      int   `json:"total"`
	Raw        int   `json:"raw"`
	Others     int   `json:"others"`
	NoExif     int   `json:"noExif"`
	TotalBytes int64 `json:"totalBytes"`
}

type YearStat struct {
	Year  int `json:"year"`
	Count int `json:"count"`
}

type CategoryStat struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

type DedupeResult struct {
	Scanned int    `json:"scanned"`
	Groups  int    `json:"groups"`
	Removed int    `json:"removed"`
	Freed   int64  `json:"freed"`
	DryRun  bool   `json:"dryRun"`
	Err     string `json:"err,omitempty"`
}

type OrganizeResult struct {
	Moved      int            `json:"moved"`
	Raw        int            `json:"raw"`
	Others     int            `json:"others"`
	Skipped    int            `json:"skipped"`
	Dupes      int            `json:"dupes"`
	Cleaned    int            `json:"cleaned"`
	Migrated   int            `json:"migrated"`
	ByYear     []YearStat     `json:"byYear"`
	ByCategory []CategoryStat `json:"byCategory"`
	Err        string         `json:"err,omitempty"`
}

type App struct {
	ctx       context.Context
	prefs     Prefs
	prefsPath string
	watcher   *Watcher
	cancelOp  context.CancelFunc
	opMu      sync.Mutex
	logBuf    strings.Builder
	logMu     sync.Mutex
}

func NewApp() *App {
	a := &App{}
	if home, err := os.UserHomeDir(); err == nil {
		dir := filepath.Join(home, ".myphoto")
		os.MkdirAll(dir, 0o755)
		a.prefsPath = filepath.Join(dir, "prefs.json")
	}
	a.loadPrefs()
	return a
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) loadPrefs() {
	a.prefs = Prefs{FolderFmt: "2006_01_02", FileTpl: "photo_{date}_{time}"}
	if a.prefsPath == "" {
		return
	}
	data, err := os.ReadFile(a.prefsPath)
	if err != nil {
		return
	}
	json.Unmarshal(data, &a.prefs)
	if a.prefs.FolderFmt == "" {
		a.prefs.FolderFmt = "2006_01_02"
	}
	if a.prefs.FileTpl == "" {
		a.prefs.FileTpl = "photo_{date}_{time}"
	}
}

func (a *App) savePrefs() {
	if a.prefsPath == "" {
		return
	}
	data, _ := json.Marshal(a.prefs)
	os.WriteFile(a.prefsPath, data, 0o644)
}

func (a *App) GetPreferences() Prefs {
	return a.prefs
}

func (a *App) SavePreferences(p Prefs) {
	a.prefs = p
	a.savePrefs()
}

func (a *App) ChooseInputDir() string {
	dir, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Seleziona cartella di input",
	})
	if err != nil || dir == "" {
		return ""
	}
	a.prefs.InputDir = dir
	a.pushRecent(dir)
	a.savePrefs()
	return dir
}

func (a *App) pushRecent(dir string) {
	if dir == "" {
		return
	}
	out := []string{dir}
	for _, r := range a.prefs.Recents {
		if r == dir || r == "" {
			continue
		}
		out = append(out, r)
		if len(out) >= maxRecents {
			break
		}
	}
	a.prefs.Recents = out
}

func (a *App) ClearRecents() {
	a.prefs.Recents = nil
	a.savePrefs()
}

func (a *App) ResetPreferences() Prefs {
	a.prefs = Prefs{
		FolderFmt: "2006_01_02",
		FileTpl:   "photo_{date}_{time}",
		Recents:   a.prefs.Recents,
	}
	a.savePrefs()
	return a.prefs
}

// FormatPreview restituisce un esempio del path risultante per i valori dati.
func (a *App) FormatPreview(folderFmt, fileTpl string) FormatPreviewResult {
	if folderFmt == "" {
		folderFmt = "2006_01_02"
	}
	if fileTpl == "" {
		fileTpl = "photo_{date}_{time}"
	}
	dt := time.Date(2026, 5, 30, 15, 45, 32, 0, time.Local)
	folder := dt.Format(folderFmt)
	file := buildFilename(fileTpl, dt, "canon_eos_r5", ".jpg")
	return FormatPreviewResult{
		Folder: folder,
		File:   file,
		Full:   filepath.Join(otherCategory(".jpg"), folder, file),
	}
}

func (a *App) ChooseOutputDir() string {
	dir, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Seleziona cartella di output",
	})
	if err != nil || dir == "" {
		return ""
	}
	a.prefs.OutputDir = dir
	a.savePrefs()
	return dir
}

func (a *App) ScanPhotos(inputDir string) ScanResult {
	if inputDir == "" {
		return ScanResult{}
	}
	photos, err := collectPhotos(inputDir, inputDir)
	if err != nil {
		return ScanResult{}
	}
	var result ScanResult
	result.Total = len(photos)
	for _, p := range photos {
		if rawExtensions[strings.ToLower(filepath.Ext(p))] {
			result.Raw++
		} else {
			result.Others++
		}
		if info, statErr := os.Stat(p); statErr == nil {
			result.TotalBytes += info.Size()
		}
		if _, ok := getExifDatetime(p); !ok {
			result.NoExif++
		}
	}
	return result
}

// NotifyDesktop mostra una notifica desktop macOS via osascript.
func (a *App) NotifyDesktop(title, body string) {
	if runtime.GOOS != "darwin" {
		return
	}
	// Escape doppi apici per osascript.
	esc := func(s string) string {
		return strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`)
	}
	script := fmt.Sprintf(`display notification "%s" with title "%s"`, esc(body), esc(title))
	exec.Command("osascript", "-e", script).Start() //nolint:errcheck
}

// ShowAbout mostra il dialog "Informazioni" via Wails.
func (a *App) ShowAbout() {
	if a.ctx == nil {
		return
	}
	wailsruntime.MessageDialog(a.ctx, wailsruntime.MessageDialogOptions{
		Type:    wailsruntime.InfoDialog,
		Title:   "MyPhotoManager",
		Message: "Organizza, rinomina e deduplica le tue foto in modo automatico.\n\nFatto con Wails + Vue 3.",
		Buttons: []string{"OK"},
	}) //nolint:errcheck
}

type eventLogWriter struct {
	app *App
}

func (w *eventLogWriter) Write(p []byte) (int, error) {
	w.app.logMu.Lock()
	w.app.logBuf.Write(p)
	text := w.app.logBuf.String()
	w.app.logMu.Unlock()
	wailsruntime.EventsEmit(w.app.ctx, "log:update", text)
	return len(p), nil
}

func (a *App) resetLog() {
	a.logMu.Lock()
	a.logBuf.Reset()
	a.logMu.Unlock()
	wailsruntime.EventsEmit(a.ctx, "log:update", "")
}

func (a *App) buildOrgOpts(p Prefs) OrganizerOptions {
	return OrganizerOptions{
		DryRun:          p.DryRun,
		StripMeta:       p.StripMeta,
		CopyMode:        p.CopyMode,
		ModTimeFallback: p.ModTime,
		CheckDupes:      p.CheckDupes,
		RenameOnly:      p.RenameOnly,
		CleanEmptyDirs:  p.CleanDirs,
		FolderFormat:    p.FolderFmt,
		FileTemplate:    p.FileTpl,
		RawSplit:        p.RawSplit,
	}
}

func (a *App) buildResult(stats OrganizerStats) OrganizeResult {
	result := OrganizeResult{
		Moved:    stats.Moved,
		Raw:      stats.Raw,
		Others:   stats.Altri,
		Skipped:  stats.Skipped,
		Dupes:    stats.Dupes,
		Cleaned:  stats.Cleaned,
		Migrated: stats.Migrated,
	}
	years := make([]int, 0, len(stats.ByYear))
	for y := range stats.ByYear {
		years = append(years, y)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	for _, y := range years {
		result.ByYear = append(result.ByYear, YearStat{Year: y, Count: stats.ByYear[y]})
	}
	cats := make([]string, 0, len(stats.ByCategory))
	for c := range stats.ByCategory {
		cats = append(cats, c)
	}
	sort.Slice(cats, func(i, j int) bool {
		if stats.ByCategory[cats[i]] != stats.ByCategory[cats[j]] {
			return stats.ByCategory[cats[i]] > stats.ByCategory[cats[j]]
		}
		return cats[i] < cats[j]
	})
	for _, c := range cats {
		result.ByCategory = append(result.ByCategory, CategoryStat{Category: c, Count: stats.ByCategory[c]})
	}
	return result
}

// progressEmitter restituisce una ProgressFunc che calcola throughput ed ETA
// e emette progress:update verso il frontend.
func (a *App) progressEmitter() ProgressFunc {
	start := time.Now()
	var lastEmit time.Time
	return func(cur, tot int, name string) {
		now := time.Now()
		// Throttle: max ~20 eventi/s, ma sempre primo e ultimo.
		if cur > 1 && cur < tot && now.Sub(lastEmit) < 50*time.Millisecond {
			return
		}
		lastEmit = now
		elapsed := now.Sub(start).Seconds()
		var throughput float64
		var etaSec float64
		if elapsed > 0.2 {
			throughput = float64(cur) / elapsed
			if throughput > 0 && cur < tot {
				etaSec = float64(tot-cur) / throughput
			}
		}
		wailsruntime.EventsEmit(a.ctx, "progress:update", map[string]interface{}{
			"current":    cur,
			"total":      tot,
			"filename":   name,
			"throughput": throughput,
			"etaSec":     etaSec,
			"elapsedSec": elapsed,
		})
	}
}

func (a *App) Organize(opts Prefs) {
	a.opMu.Lock()
	if a.cancelOp != nil {
		a.cancelOp()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancelOp = cancel
	a.opMu.Unlock()

	a.resetLog()
	wailsruntime.EventsEmit(a.ctx, "organize:start", nil)

	go func() {
		inputDir := opts.InputDir
		outputDir := opts.OutputDir
		if outputDir == "" {
			outputDir = inputDir
		}

		lw := &eventLogWriter{app: a}
		stats, err := organizePhotos(ctx, inputDir, outputDir, a.buildOrgOpts(opts), a.progressEmitter(), lw)
		result := a.buildResult(stats)
		if err != nil && ctx.Err() == nil {
			result.Err = err.Error()
		}
		if ctx.Err() == nil && !opts.DryRun && result.Moved > 0 {
			a.appendHistory(HistoryEntry{
				RunAt:    time.Now(),
				InputDir: opts.InputDir,
				Moved:    result.Moved,
				Raw:      result.Raw,
				Others:   result.Others,
				Skipped:  result.Skipped,
				Dupes:    result.Dupes,
			})
		}
		wailsruntime.EventsEmit(a.ctx, "organize:done", result)
	}()
}

func (a *App) StopOperation() {
	a.opMu.Lock()
	if a.cancelOp != nil {
		a.cancelOp()
		a.cancelOp = nil
	}
	a.opMu.Unlock()
}

func (a *App) BeginWatch(opts Prefs) {
	if opts.InputDir == "" {
		return
	}
	a.StopCurrentWatch()

	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = opts.InputDir
	}

	a.resetLog()
	lw := &eventLogWriter{app: a}

	a.watcher = StartWatch(opts.InputDir, outputDir, a.buildOrgOpts(opts), lw, a.progressEmitter(), func(stats OrganizerStats) {
		msg := fmt.Sprintf("Ultima scansione: %d file elaborati", stats.Moved+stats.Skipped)
		wailsruntime.EventsEmit(a.ctx, "watch:status", msg)
		wailsruntime.EventsEmit(a.ctx, "organize:done", a.buildResult(stats))
	})

	wailsruntime.EventsEmit(a.ctx, "watch:status", "Watch attivo — in attesa di nuovi file…")
}

func (a *App) StopCurrentWatch() {
	if a.watcher != nil {
		a.watcher.Stop()
		a.watcher = nil
	}
	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "watch:status", "")
	}
}

// HandleDrop riceve i path droppati sulla finestra e restituisce il primo che è una directory.
func (a *App) HandleDrop(paths []string) string {
	for _, p := range paths {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			a.prefs.InputDir = p
			a.pushRecent(p)
			a.savePrefs()
			return p
		}
	}
	return ""
}

func (a *App) Dedupe(inputDir string, dryRun bool) {
	a.opMu.Lock()
	if a.cancelOp != nil {
		a.cancelOp()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancelOp = cancel
	a.opMu.Unlock()

	a.resetLog()
	wailsruntime.EventsEmit(a.ctx, "organize:start", nil)

	go func() {
		if inputDir == "" {
			inputDir = a.prefs.InputDir
		}
		lw := &eventLogWriter{app: a}
		stats, err := dedupePhotos(ctx, inputDir, dryRun, a.progressEmitter(), lw)
		result := DedupeResult{
			Scanned: stats.Scanned,
			Groups:  stats.Groups,
			Removed: stats.Removed,
			Freed:   stats.FreedBytes,
			DryRun:  dryRun,
		}
		if err != nil && ctx.Err() == nil {
			result.Err = err.Error()
		}
		wailsruntime.EventsEmit(a.ctx, "dedupe:done", result)
		wailsruntime.EventsEmit(a.ctx, "organize:done", map[string]interface{}{})
	}()
}

func (a *App) OpenInFinder(path string) {
	if path == "" {
		return
	}
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
