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

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type Prefs struct {
	InputDir   string `json:"inputDir"`
	OutputDir  string `json:"outputDir"`
	DryRun     bool   `json:"dryRun"`
	CopyMode   bool   `json:"copyMode"`
	StripMeta  bool   `json:"stripMeta"`
	ModTime    bool   `json:"modTime"`
	CheckDupes bool   `json:"checkDupes"`
	RenameOnly bool   `json:"renameOnly"`
	CleanDirs  bool   `json:"cleanDirs"`
	FolderFmt  string `json:"folderFmt"`
	FileTpl    string `json:"fileTpl"`
}

type ScanResult struct {
	Total  int `json:"total"`
	Raw    int `json:"raw"`
	Others int `json:"others"`
	NoExif int `json:"noExif"`
}

type YearStat struct {
	Year  int `json:"year"`
	Count int `json:"count"`
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
	Moved   int        `json:"moved"`
	Raw     int        `json:"raw"`
	Others  int        `json:"others"`
	Skipped int        `json:"skipped"`
	Dupes   int        `json:"dupes"`
	Cleaned int        `json:"cleaned"`
	ByYear  []YearStat `json:"byYear"`
	Err     string     `json:"err,omitempty"`
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
	a.savePrefs()
	return dir
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
		if _, ok := getExifDatetime(p); !ok {
			result.NoExif++
		}
	}
	return result
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
	}
}

func (a *App) buildResult(stats OrganizerStats) OrganizeResult {
	result := OrganizeResult{
		Moved:   stats.Moved,
		Raw:     stats.Raw,
		Others:  stats.Altri,
		Skipped: stats.Skipped,
		Dupes:   stats.Dupes,
		Cleaned: stats.Cleaned,
	}
	years := make([]int, 0, len(stats.ByYear))
	for y := range stats.ByYear {
		years = append(years, y)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	for _, y := range years {
		result.ByYear = append(result.ByYear, YearStat{Year: y, Count: stats.ByYear[y]})
	}
	return result
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
		onProgress := func(cur, tot int, name string) {
			wailsruntime.EventsEmit(a.ctx, "progress:update", map[string]interface{}{
				"current": cur, "total": tot, "filename": name,
			})
		}

		stats, err := organizePhotos(ctx, inputDir, outputDir, a.buildOrgOpts(opts), onProgress, lw)
		result := a.buildResult(stats)
		if err != nil && ctx.Err() == nil {
			result.Err = err.Error()
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
	onProgress := func(cur, tot int, name string) {
		wailsruntime.EventsEmit(a.ctx, "progress:update", map[string]interface{}{
			"current": cur, "total": tot, "filename": name,
		})
	}

	a.watcher = StartWatch(opts.InputDir, outputDir, a.buildOrgOpts(opts), lw, onProgress, func(stats OrganizerStats) {
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
		onProgress := func(cur, tot int, name string) {
			wailsruntime.EventsEmit(a.ctx, "progress:update", map[string]interface{}{
				"current": cur, "total": tot, "filename": name,
			})
		}
		stats, err := dedupePhotos(ctx, inputDir, dryRun, onProgress, lw)
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
