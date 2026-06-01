package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

var rawExtensions = map[string]bool{
	".arw": true, ".cr2": true, ".cr3": true, ".nef": true,
	".dng": true, ".raf": true, ".rw2": true, ".orf": true,
	".pef": true, ".srw": true, ".x3f": true, ".3fr": true,
	".mef": true, ".mrw": true, ".nrw": true, ".rwl": true,
	".iiq": true, ".erf": true,
}

var otherExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".tiff": true,
	".tif": true, ".heic": true, ".heif": true, ".bmp": true,
	".webp": true,
}

// otherCategory restituisce il nome della cartella per un file non-RAW,
// derivato dall'estensione (senza punto). Alcune estensioni equivalenti
// vengono normalizzate sotto un'unica cartella.
func otherCategory(ext string) string {
	e := strings.TrimPrefix(strings.ToLower(ext), ".")
	switch e {
	case "jpeg":
		return "jpg"
	case "tif":
		return "tiff"
	case "heif":
		return "heic"
	}
	return e
}

// categoryFor restituisce la cartella di destinazione per un file:
// "raw" per i formati RAW (uniti), altrimenti il nome derivato dall'estensione.
func categoryFor(ext string, isRaw bool) string {
	if isRaw {
		return "raw"
	}
	return otherCategory(ext)
}

// managedFolders elenca i nomi delle cartelle gestite dall'app, escluse dalla
// raccolta dei sorgenti e dalla pulizia delle cartelle vuote. Include "raw",
// "senza_data", la storica "altri" (compatibilità) e una cartella per ogni
// estensione non-RAW supportata (jpg, png, heic…).
var managedFolders = buildManagedFolders()

func buildManagedFolders() map[string]bool {
	m := map[string]bool{
		"raw": true, "altri": true, "senza_data": true,
	}
	for ext := range otherExtensions {
		m[otherCategory(ext)] = true
	}
	return m
}

var giorni = []string{"lunedì", "martedì", "mercoledì", "giovedì", "venerdì", "sabato", "domenica"}
var mesi = []string{
	"gennaio", "febbraio", "marzo", "aprile", "maggio", "giugno",
	"luglio", "agosto", "settembre", "ottobre", "novembre", "dicembre",
}

// OrganizerOptions raccoglie tutte le opzioni di organizzazione.
type OrganizerOptions struct {
	DryRun          bool
	StripMeta       bool
	CopyMode        bool   // copia invece di sposta
	ModTimeFallback bool   // usa data modifica file se EXIF mancante
	CheckDupes      bool   // salta duplicati identici (SHA-256)
	RenameOnly      bool   // rinomina in-place senza spostare in sottocartelle
	CleanEmptyDirs  bool   // rimuovi cartelle vuote dopo lo spostamento
	FolderFormat    string // formato Go time per la cartella data (es. "2006_01_02")
	FileTemplate    string // template nome file (es. "photo_{date}_{time}")
	RawSplit        string // suddivisione extra dei soli RAW per metadato: ""|camera|lens|iso|camera_lens
	Workers         int    // goroutine parallele (0 = auto)
}

// OrganizerStats raccoglie le statistiche dell'operazione.
type OrganizerStats struct {
	Moved      int
	Raw        int
	Altri      int
	Skipped    int
	Dupes      int
	Cleaned    int            // cartelle vuote rimosse
	Migrated   int            // file migrati dalla storica cartella altri/
	ByYear     map[int]int
	ByCategory map[string]int // conteggio per cartella di destinazione (raw, jpg, png…)
}

// ProgressFunc viene chiamata ad ogni file elaborato.
type ProgressFunc func(current, total int, filename string)

func parseExifTime(raw string) (time.Time, error) {
	return time.ParseInLocation("2006:01:02 15:04:05", raw, time.Local)
}

func getExifDatetime(path string) (time.Time, bool) {
	f, err := os.Open(path)
	if err != nil {
		return time.Time{}, false
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return time.Time{}, false
	}

	for _, tagName := range []exif.FieldName{exif.DateTimeOriginal, exif.DateTimeDigitized, exif.DateTime} {
		tag, err := x.Get(tagName)
		if err != nil {
			continue
		}
		raw, err := tag.StringVal()
		if err != nil {
			continue
		}
		t, err := parseExifTime(raw)
		if err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// sanitizeFolder normalizza una stringa in un nome cartella sicuro:
// mantiene lettere/cifre/'-', sostituisce il resto con '_' e comprime le ripetizioni.
func sanitizeFolder(s string) string {
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '_'
	}, s)
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	return strings.Trim(s, "_")
}

// exifString legge un tag EXIF come stringa (trimmata), "" se assente.
func exifString(x *exif.Exif, name exif.FieldName) string {
	tag, err := x.Get(name)
	if err != nil {
		return ""
	}
	s, err := tag.StringVal()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(s)
}

// exifInt legge un tag EXIF come intero, 0 se assente.
func exifInt(x *exif.Exif, name exif.FieldName) int {
	tag, err := x.Get(name)
	if err != nil {
		return 0
	}
	v, err := tag.Int(0)
	if err != nil {
		return 0
	}
	return v
}

func getExifCamera(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return ""
	}

	var parts []string
	if s := exifString(x, exif.Make); s != "" {
		parts = append(parts, s)
	}
	if s := exifString(x, exif.Model); s != "" {
		parts = append(parts, s)
	}

	cam := sanitizeFolder(strings.Join(parts, "_"))
	if cam == "" {
		return "unknown"
	}
	return cam
}

// isoBand raggruppa il valore ISO in una fascia, per non generare una cartella per valore.
func isoBand(iso int) string {
	switch {
	case iso <= 0:
		return "sconosciuto"
	case iso <= 200:
		return "iso_0-200"
	case iso <= 800:
		return "iso_200-800"
	case iso <= 3200:
		return "iso_800-3200"
	case iso <= 12800:
		return "iso_3200-12800"
	default:
		return "iso_12800+"
	}
}

// rawSplitValue legge dall'EXIF il nome della sotto-cartella per la suddivisione
// extra dei RAW secondo il criterio scelto (camera|lens|iso|camera_lens).
// Restituisce "sconosciuto" se il metadato è assente, "" se split è disattivato.
func rawSplitValue(path, split string) string {
	if split == "" {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return "sconosciuto"
	}
	defer f.Close()
	x, err := exif.Decode(f)
	if err != nil {
		return "sconosciuto"
	}

	camera := func() string {
		c := sanitizeFolder(strings.Join(nonEmpty(exifString(x, exif.Make), exifString(x, exif.Model)), "_"))
		if c == "" {
			return "sconosciuto"
		}
		return c
	}
	lens := func() string {
		l := sanitizeFolder(exifString(x, exif.LensModel))
		if l == "" {
			return "sconosciuto"
		}
		return l
	}

	switch split {
	case "camera":
		return camera()
	case "lens":
		return lens()
	case "iso":
		return isoBand(exifInt(x, exif.ISOSpeedRatings))
	case "camera_lens":
		return filepath.Join(camera(), lens())
	}
	return "sconosciuto"
}

// nonEmpty restituisce solo le stringhe non vuote, nell'ordine dato.
func nonEmpty(ss ...string) []string {
	var out []string
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func buildFilename(template string, dt time.Time, camera, ext string) string {
	r := template
	r = strings.ReplaceAll(r, "{date}", dt.Format("2006_01_02"))
	r = strings.ReplaceAll(r, "{time}", dt.Format("150405"))
	r = strings.ReplaceAll(r, "{datetime}", dt.Format("2006_01_02_150405"))
	r = strings.ReplaceAll(r, "{year}", dt.Format("2006"))
	r = strings.ReplaceAll(r, "{month}", dt.Format("01"))
	r = strings.ReplaceAll(r, "{day}", dt.Format("02"))
	r = strings.ReplaceAll(r, "{camera}", camera)
	return r + strings.ToLower(ext)
}

func buildDestPath(outputDir string, dt time.Time, ext string, isRaw bool, camera, rawSub string, opts OrganizerOptions) string {
	category := categoryFor(ext, isRaw)

	folderFmt := opts.FolderFormat
	if folderFmt == "" {
		folderFmt = "2006_01_02"
	}

	tpl := opts.FileTemplate
	if tpl == "" {
		tpl = "photo_{date}_{time}"
	}

	parts := []string{outputDir, category}
	if rawSub != "" {
		parts = append(parts, rawSub) // livello extra per i RAW (es. fotocamera)
	}
	parts = append(parts, dt.Format(folderFmt), buildFilename(tpl, dt, camera, ext))
	return filepath.Join(parts...)
}

func resolveConflict(dest string) string {
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return dest
	}
	ext := filepath.Ext(dest)
	stem := strings.TrimSuffix(dest, ext)
	for counter := 1; ; counter++ {
		candidate := fmt.Sprintf("%s_%d%s", stem, counter, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

// dupeChecker rileva file identici tramite hash SHA-256.
type dupeChecker struct {
	mu     sync.Mutex
	hashes map[string]string
}

func (dc *dupeChecker) check(path string) (isDupe bool, firstSeen string, err error) {
	h, err := fileHash(path)
	if err != nil {
		return false, "", err
	}
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if existing, ok := dc.hashes[h]; ok {
		return true, existing, nil
	}
	dc.hashes[h] = path
	return false, "", nil
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// moveToTrash sposta path nel cestino macOS (~/.Trash). Su altri OS rimuove definitivamente.
// Restituisce errore solo per problemi gravi; conflitti di nome vengono risolti automaticamente.
func moveToTrash(path string) error {
	if runtime.GOOS != "darwin" {
		return os.Remove(path)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return os.Remove(path)
	}
	trash := filepath.Join(home, ".Trash")
	if err := os.MkdirAll(trash, 0o755); err != nil {
		return os.Remove(path)
	}
	name := filepath.Base(path)
	dst := filepath.Join(trash, name)
	if _, err := os.Stat(dst); err == nil {
		ext := filepath.Ext(name)
		stem := strings.TrimSuffix(name, ext)
		ts := time.Now().Format("20060102_150405")
		for i := 0; ; i++ {
			var candidate string
			if i == 0 {
				candidate = filepath.Join(trash, fmt.Sprintf("%s %s%s", stem, ts, ext))
			} else {
				candidate = filepath.Join(trash, fmt.Sprintf("%s %s_%d%s", stem, ts, i, ext))
			}
			if _, err := os.Stat(candidate); os.IsNotExist(err) {
				dst = candidate
				break
			}
		}
	}
	if err := os.Rename(path, dst); err != nil {
		// Cross-volume: copia + rimuovi.
		if copyErr := copyFile(path, dst); copyErr != nil {
			return copyErr
		}
		return os.Remove(path)
	}
	return nil
}

func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyFile(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// stripJPEGExif copia src in dst saltando i segmenti APP1 (EXIF/XMP) e APP13 (IPTC).
func stripJPEGExif(src io.Reader, dst io.Writer) error {
	data, err := io.ReadAll(src)
	if err != nil {
		return err
	}
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		_, err = dst.Write(data)
		return err
	}

	out := make([]byte, 0, len(data))
	out = append(out, 0xFF, 0xD8)
	i := 2

	for i < len(data) {
		if i+1 >= len(data) {
			break
		}
		if data[i] != 0xFF {
			out = append(out, data[i:]...)
			break
		}
		marker := data[i+1]
		if marker == 0xFF {
			i++
			continue
		}
		if marker == 0xD8 || marker == 0xD9 || (marker >= 0xD0 && marker <= 0xD7) {
			out = append(out, data[i:i+2]...)
			i += 2
			continue
		}
		if marker == 0xDA {
			out = append(out, data[i:]...)
			break
		}
		if i+3 >= len(data) {
			out = append(out, data[i:]...)
			break
		}
		segLen := int(data[i+2])<<8 | int(data[i+3])
		end := i + 2 + segLen
		if end > len(data) {
			end = len(data)
		}
		if marker == 0xE1 || marker == 0xED {
			i = end
			continue
		}
		out = append(out, data[i:end]...)
		i = end
	}

	_, err = dst.Write(out)
	return err
}

func transferStripped(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		in.Close()
		return err
	}
	stripErr := stripJPEGExif(in, out)
	syncErr := out.Sync()
	out.Close()
	in.Close()
	if stripErr != nil {
		os.Remove(dst)
		return stripErr
	}
	if syncErr != nil {
		os.Remove(dst)
		return syncErr
	}
	return nil
}

func transferFile(src, dst string, opts OrganizerOptions) error {
	ext := strings.ToLower(filepath.Ext(src))
	willStrip := opts.StripMeta && (ext == ".jpg" || ext == ".jpeg")

	if opts.CopyMode {
		if willStrip {
			return transferStripped(src, dst)
		}
		return copyFile(src, dst)
	}
	if willStrip {
		if err := transferStripped(src, dst); err != nil {
			return err
		}
		return os.Remove(src)
	}
	return moveFile(src, dst)
}

// renameInPlace rinomina src applicando il template, restando nella stessa cartella.
func renameInPlace(src string, dt time.Time, camera string, opts OrganizerOptions) (string, error) {
	ext := strings.ToLower(filepath.Ext(src))
	tpl := opts.FileTemplate
	if tpl == "" {
		tpl = "photo_{date}_{time}"
	}
	dst := resolveConflict(filepath.Join(filepath.Dir(src), buildFilename(tpl, dt, camera, ext)))

	willStrip := opts.StripMeta && (ext == ".jpg" || ext == ".jpeg")
	if willStrip {
		if err := transferStripped(src, dst); err != nil {
			return "", err
		}
		return dst, os.Remove(src)
	}
	return dst, os.Rename(src, dst)
}

// removeEmptyDirs rimuove ricorsivamente le cartelle vuote sotto root,
// saltando le managed folders (raw, altri, senza_data).
func removeEmptyDirs(root string, logW io.Writer) int {
	var dirs []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() || path == root {
			return nil
		}
		dirs = append(dirs, path)
		return nil
	})

	// Processa prima le directory più profonde
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	removed := 0
	for _, dir := range dirs {
		if managedFolders[filepath.Base(dir)] {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 0 {
			continue
		}
		if os.Remove(dir) == nil {
			rel, _ := filepath.Rel(root, dir)
			fmt.Fprintf(logW, "  rimossa cartella vuota: %s\n", rel)
			removed++
		}
	}
	return removed
}

// migrateAltri ridistribuisce il contenuto di una eventuale cartella outputDir/altri
// (struttura precedente) nelle nuove cartelle per tipo, preservando le sottocartelle
// per data. Al termine rimuove le cartelle vuote e la cartella altri stessa.
// Restituisce il numero di file migrati. In dry-run logga senza spostare.
func migrateAltri(outputDir string, dryRun bool, logW io.Writer) int {
	altriDir := filepath.Join(outputDir, "altri")
	info, err := os.Stat(altriDir)
	if err != nil || !info.IsDir() {
		return 0
	}

	var files []string
	filepath.Walk(altriDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		if strings.HasPrefix(fi.Name(), "._") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(fi.Name()))
		if !rawExtensions[ext] && !otherExtensions[ext] {
			return nil
		}
		files = append(files, path)
		return nil
	})

	if len(files) == 0 {
		if !dryRun {
			os.Remove(altriDir) // rimuove se vuota
		}
		return 0
	}
	sort.Strings(files)

	fmt.Fprintf(logW, "Migrazione struttura: %d file da altri/ alle cartelle per tipo.\n", len(files))
	migrated := 0
	for _, src := range files {
		ext := strings.ToLower(filepath.Ext(src))
		category := categoryFor(ext, rawExtensions[ext])
		rel, _ := filepath.Rel(altriDir, src) // <cartella_data>/<file>
		dest := resolveConflict(filepath.Join(outputDir, category, rel))
		relSrc, _ := filepath.Rel(outputDir, src)
		relDest, _ := filepath.Rel(outputDir, dest)
		if !dryRun {
			if mkErr := os.MkdirAll(filepath.Dir(dest), 0o755); mkErr != nil {
				fmt.Fprintf(logW, "  errore  %s: %v\n", relSrc, mkErr)
				continue
			}
			if mvErr := moveFile(src, dest); mvErr != nil {
				fmt.Fprintf(logW, "  errore  %s: %v\n", relSrc, mvErr)
				continue
			}
		}
		fmt.Fprintf(logW, "  %s → %s\n", relSrc, relDest)
		migrated++
	}

	if !dryRun {
		removeEmptyDirs(altriDir, logW)
		if entries, rdErr := os.ReadDir(altriDir); rdErr == nil && len(entries) == 0 {
			os.Remove(altriDir)
		}
	}
	fmt.Fprintln(logW)
	return migrated
}

func isExcluded(path string, excluded []string) bool {
	for _, exc := range excluded {
		rel, err := filepath.Rel(exc, path)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return true
		}
	}
	return false
}

func collectPhotos(inputDir, outputDir string) ([]string, error) {
	var excluded []string
	for _, dir := range []string{inputDir, outputDir} {
		for folder := range managedFolders {
			candidate := filepath.Join(dir, folder)
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				abs, err := filepath.Abs(candidate)
				if err == nil {
					excluded = append(excluded, abs)
				}
			}
		}
	}

	var photos []string
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name()
		if strings.HasPrefix(name, "._") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(name))
		if !rawExtensions[ext] && !otherExtensions[ext] {
			return nil
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil
		}
		if isExcluded(abs, excluded) {
			return nil
		}
		photos = append(photos, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(photos)
	return photos, nil
}

func formatDatetimeHuman(dt time.Time) string {
	giorno := giorni[int(dt.Weekday()+6)%7]
	mese := mesi[dt.Month()-1]
	return fmt.Sprintf("%s %d %s %d alle %s", giorno, dt.Day(), mese, dt.Year(), dt.Format("15:04:05"))
}

func organizePhotos(ctx context.Context, inputDir, outputDir string, opts OrganizerOptions, progress ProgressFunc, w io.Writer) (OrganizerStats, error) {
	stats := OrganizerStats{ByYear: make(map[int]int), ByCategory: make(map[string]int)}

	// Migra una eventuale struttura precedente (outputDir/altri) verso le cartelle per tipo.
	if !opts.RenameOnly {
		stats.Migrated = migrateAltri(outputDir, opts.DryRun, w)
	}

	photos, err := collectPhotos(inputDir, outputDir)
	if err != nil {
		return stats, err
	}

	if len(photos) == 0 {
		fmt.Fprintln(w, "Nessuna foto trovata.")
		return stats, nil
	}

	fmt.Fprintf(w, "Trovate %d foto da elaborare.\n\n", len(photos))

	var dc *dupeChecker
	if opts.CheckDupes {
		dc = &dupeChecker{hashes: make(map[string]string)}
	}

	needsCamera := strings.Contains(opts.FileTemplate, "{camera}")

	numWorkers := opts.Workers
	if numWorkers <= 0 {
		numWorkers = min(runtime.NumCPU(), 8)
	}

	jobs := make(chan string, numWorkers*2)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex  // guarda w + stats
		ioMu    sync.Mutex  // serializza resolveConflict + mkdir + transfer
		counter atomic.Int32
	)

	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for photo := range jobs {
				if ctx.Err() != nil {
					return
				}

				ext := strings.ToLower(filepath.Ext(photo))
				isRaw := rawExtensions[ext]
				name := filepath.Base(photo)

				// ── Phase 1: letture parallele (nessun lock esterno) ──────────────────
				if dc != nil {
					isDupe, firstSeen, hashErr := dc.check(photo)
					if hashErr == nil && isDupe {
						cur := int(counter.Add(1))
						if progress != nil {
							progress(cur, len(photos), name)
						}
						mu.Lock()
						fmt.Fprintf(w, "  duplicato  %s\n         = %s\n\n", name, filepath.Base(firstSeen))
						stats.Dupes++
						mu.Unlock()
						continue
					}
				}

				dt, hasExif := getExifDatetime(photo)
				modTimeUsed := false
				if !hasExif && opts.ModTimeFallback {
					if info, statErr := os.Stat(photo); statErr == nil {
						dt = info.ModTime()
						hasExif = true
						modTimeUsed = true
					}
				}

				camera := ""
				if needsCamera && hasExif {
					camera = getExifCamera(photo)
				}

				cur := int(counter.Add(1))
				if progress != nil {
					progress(cur, len(photos), name)
				}

				categoria := "foto"
				if isRaw {
					categoria = "RAW "
				}

				noteStr := buildNoteStr(opts, isRaw, ext, modTimeUsed)

				// ── Rename-only mode ──────────────────────────────────────────────────
				if opts.RenameOnly {
					if !hasExif {
						mu.Lock()
						fmt.Fprintf(w, "  saltato  %s  (nessuna data disponibile)\n\n", name)
						stats.Skipped++
						mu.Unlock()
						continue
					}
					tpl := opts.FileTemplate
					if tpl == "" {
						tpl = "photo_{date}_{time}"
					}
					newName := buildFilename(tpl, dt, camera, ext)
					logLine := fmt.Sprintf("  %s  %s%s\n         %s\n         → %s\n\n",
						categoria, name, noteStr, formatDatetimeHuman(dt), newName)
					if !opts.DryRun {
						ioMu.Lock()
						_, renErr := renameInPlace(photo, dt, camera, opts)
						ioMu.Unlock()
						if renErr != nil {
							logLine = fmt.Sprintf("  errore  %s: %v\n\n", name, renErr)
						}
					}
					mu.Lock()
					fmt.Fprint(w, logLine)
					stats.Moved++
					stats.ByYear[dt.Year()]++
					stats.ByCategory[categoryFor(ext, isRaw)]++
					if isRaw {
						stats.Raw++
					} else {
						stats.Altri++
					}
					mu.Unlock()
					continue
				}

				// ── Organizzazione standard ───────────────────────────────────────────
				if !hasExif {
					ioMu.Lock()
					dest := resolveConflict(filepath.Join(outputDir, "senza_data", name))
					rel, _ := filepath.Rel(outputDir, dest)
					if !opts.DryRun {
						os.MkdirAll(filepath.Dir(dest), 0o755)   //nolint:errcheck
						transferFile(photo, dest, opts)           //nolint:errcheck
					}
					ioMu.Unlock()
					mu.Lock()
					fmt.Fprintf(w, "  senza data  %s\n         → %s\n\n", name, rel)
					stats.Skipped++
					mu.Unlock()
					continue
				}

				rawSub := ""
				if isRaw && opts.RawSplit != "" {
					rawSub = rawSplitValue(photo, opts.RawSplit)
				}

				dest := buildDestPath(outputDir, dt, ext, isRaw, camera, rawSub, opts)

				ioMu.Lock()
				dest = resolveConflict(dest)
				rel, _ := filepath.Rel(outputDir, dest)
				if !opts.DryRun {
					os.MkdirAll(filepath.Dir(dest), 0o755) //nolint:errcheck
					transferFile(photo, dest, opts)        //nolint:errcheck
				}
				ioMu.Unlock()

				mu.Lock()
				fmt.Fprintf(w, "  %s  %s%s\n         %s\n         → %s\n\n",
					categoria, name, noteStr, formatDatetimeHuman(dt), rel)
				stats.Moved++
				stats.ByYear[dt.Year()]++
				stats.ByCategory[categoryFor(ext, isRaw)]++
				if isRaw {
					stats.Raw++
				} else {
					stats.Altri++
				}
				mu.Unlock()
			}
		}()
	}

	// Invia job; interrompe se il context viene cancellato
	for _, p := range photos {
		select {
		case jobs <- p:
		case <-ctx.Done():
			goto drain
		}
	}
drain:
	close(jobs)
	wg.Wait()

	if ctx.Err() != nil {
		fmt.Fprintln(w, "\n⚠ Operazione annullata.")
		return stats, nil
	}

	// ── Pulizia cartelle vuote ────────────────────────────────────────────────
	if opts.CleanEmptyDirs && !opts.DryRun && !opts.RenameOnly {
		fmt.Fprintln(w, "")
		n := removeEmptyDirs(inputDir, w)
		stats.Cleaned = n
	}

	// ── Riepilogo ─────────────────────────────────────────────────────────────
	fmt.Fprintln(w, strings.Repeat("─", 50))

	verbo := "da spostare"
	switch {
	case opts.RenameOnly && opts.DryRun:
		verbo = "da rinominare"
	case opts.RenameOnly:
		verbo = "rinominati"
	case opts.CopyMode && opts.DryRun:
		verbo = "da copiare"
	case opts.CopyMode:
		verbo = "copiati"
	case opts.DryRun:
		verbo = "da spostare"
	default:
		verbo = "spostati"
	}

	fmt.Fprintf(w, "  %d file %s  (%d RAW, %d altri)\n", stats.Moved, verbo, stats.Raw, stats.Altri)

	// Dettaglio per tipo (cartella di destinazione), ordinato per conteggio decrescente.
	if len(stats.ByCategory) > 0 {
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
			fmt.Fprintf(w, "      · %s/  %d\n", c, stats.ByCategory[c])
		}
	}

	if stats.Migrated > 0 {
		fmt.Fprintf(w, "  %d file migrati da altri/ alle cartelle per tipo\n", stats.Migrated)
	}
	if stats.Skipped > 0 {
		if opts.RenameOnly {
			fmt.Fprintf(w, "  %d saltati  (nessuna data disponibile)\n", stats.Skipped)
		} else {
			fmt.Fprintf(w, "  %d → senza_data/  (EXIF mancante)\n", stats.Skipped)
		}
	}
	if stats.Dupes > 0 {
		fmt.Fprintf(w, "  %d duplicati ignorati\n", stats.Dupes)
	}
	if stats.Cleaned > 0 {
		fmt.Fprintf(w, "  %d cartelle vuote rimosse\n", stats.Cleaned)
	}

	return stats, nil
}

// DedupeStats raccoglie le statistiche dell'operazione di deduplicazione.
type DedupeStats struct {
	Scanned    int
	Groups     int
	Removed    int
	FreedBytes int64
}

type fileData struct {
	path string
	size int64
	hash string
	ext  string // normalizzata, es. ".jpg"
}

func dedupePhotos(ctx context.Context, inputDir string, dryRun bool, progress ProgressFunc, w io.Writer) (DedupeStats, error) {
	var photos []string
	filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		if strings.HasPrefix(name, "._") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(name))
		if rawExtensions[ext] || otherExtensions[ext] {
			abs, absErr := filepath.Abs(path)
			if absErr == nil {
				photos = append(photos, abs)
			}
		}
		return nil
	})

	stats := DedupeStats{Scanned: len(photos)}
	if len(photos) == 0 {
		fmt.Fprintln(w, "Nessuna foto trovata.")
		return stats, nil
	}

	fmt.Fprintf(w, "Scansione di %d foto in corso…\n\n", len(photos))

	collected := make([]fileData, 0, len(photos))
	var collMu sync.Mutex
	var wg sync.WaitGroup
	var ctr atomic.Int32

	numWorkers := min(runtime.NumCPU(), 8)
	jobs := make(chan string, numWorkers*2)

	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				if ctx.Err() != nil {
					return
				}
				h, err := fileHash(path)
				if err != nil {
					ctr.Add(1)
					continue
				}
				var size int64
				if info, statErr := os.Stat(path); statErr == nil {
					size = info.Size()
				}
				ext := strings.ToLower(filepath.Ext(path))
				collMu.Lock()
				collected = append(collected, fileData{path, size, h, ext})
				collMu.Unlock()
				cur := int(ctr.Add(1))
				if progress != nil {
					progress(cur, len(photos), filepath.Base(path))
				}
			}
		}()
	}

	for _, p := range photos {
		if ctx.Err() != nil {
			break
		}
		jobs <- p
	}
	close(jobs)
	wg.Wait()

	if ctx.Err() != nil {
		fmt.Fprintln(w, "\n⚠ Operazione annullata.")
		return stats, nil
	}

	// Due file sono duplicati se hanno stesso hash e stessa estensione.
	// (stesso hash implica stesso contenuto, quindi stessi metadati EXIF)
	groups := make(map[string][]fileData)
	for _, fd := range collected {
		key := fd.hash + "|" + fd.ext
		groups[key] = append(groups[key], fd)
	}

	// Ordine deterministico
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		files := groups[k]
		if len(files) < 2 {
			continue
		}
		stats.Groups++

		sort.Slice(files, func(i, j int) bool {
			return files[i].path < files[j].path
		})

		keep := files[0]
		relKeep, _ := filepath.Rel(inputDir, keep.path)
		fmt.Fprintf(w, "  mantieni  %s\n", relKeep)

		for _, dup := range files[1:] {
			relDup, _ := filepath.Rel(inputDir, dup.path)
			if dryRun {
				stats.Removed++
				stats.FreedBytes += dup.size
				fmt.Fprintf(w, "  ↳ dupl.   %s\n", relDup)
			} else {
				if err := moveToTrash(dup.path); err != nil {
					fmt.Fprintf(w, "  errore    %s: %v\n", relDup, err)
					continue
				}
				stats.Removed++
				stats.FreedBytes += dup.size
				fmt.Fprintf(w, "  → cestino %s\n", relDup)
			}
		}
		fmt.Fprintln(w)
	}

	if stats.Groups == 0 {
		fmt.Fprintln(w, "Nessun duplicato trovato.")
		return stats, nil
	}

	fmt.Fprintln(w, strings.Repeat("─", 50))
	verb := "da spostare nel cestino"
	if !dryRun {
		verb = "spostati nel cestino"
	}
	fmt.Fprintf(w, "  %d gruppi  ·  %d file %s  ·  %s liberati\n",
		stats.Groups, stats.Removed, verb, formatBytes(stats.FreedBytes))

	return stats, nil
}

func formatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func buildNoteStr(opts OrganizerOptions, isRaw bool, ext string, modTimeUsed bool) string {
	var notes []string
	if opts.StripMeta && !isRaw && (ext == ".jpg" || ext == ".jpeg") {
		notes = append(notes, "metadati rimossi")
	}
	if modTimeUsed {
		notes = append(notes, "data da filesystem")
	}
	if len(notes) == 0 {
		return ""
	}
	return "  [" + strings.Join(notes, ", ") + "]"
}
