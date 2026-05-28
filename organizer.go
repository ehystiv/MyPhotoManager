package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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

var managedFolders = map[string]bool{
	"raw": true, "altri": true, "senza_data": true,
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
}

// OrganizerStats raccoglie le statistiche dell'operazione.
type OrganizerStats struct {
	Moved   int
	Raw     int
	Altri   int
	Skipped int
	Dupes   int
	Cleaned int // cartelle vuote rimosse
	ByYear  map[int]int
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
	if tag, err := x.Get(exif.Make); err == nil {
		if s, err := tag.StringVal(); err == nil {
			s = strings.TrimSpace(s)
			if s != "" {
				parts = append(parts, s)
			}
		}
	}
	if tag, err := x.Get(exif.Model); err == nil {
		if s, err := tag.StringVal(); err == nil {
			s = strings.TrimSpace(s)
			if s != "" {
				parts = append(parts, s)
			}
		}
	}

	cam := strings.Join(parts, "_")
	cam = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '_'
	}, cam)
	cam = strings.Trim(cam, "_")
	if cam == "" {
		return "unknown"
	}
	return cam
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

func buildDestPath(outputDir string, dt time.Time, ext string, isRaw bool, camera string, opts OrganizerOptions) string {
	category := "altri"
	if isRaw {
		category = "raw"
	}

	folderFmt := opts.FolderFormat
	if folderFmt == "" {
		folderFmt = "2006_01_02"
	}

	tpl := opts.FileTemplate
	if tpl == "" {
		tpl = "photo_{date}_{time}"
	}

	return filepath.Join(outputDir, category, dt.Format(folderFmt), buildFilename(tpl, dt, camera, ext))
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
	photos, err := collectPhotos(inputDir, outputDir)
	if err != nil {
		return OrganizerStats{}, err
	}

	stats := OrganizerStats{ByYear: make(map[int]int)}

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

	for i, photo := range photos {
		if ctx.Err() != nil {
			fmt.Fprintln(w, "\n⚠ Operazione annullata.")
			return stats, nil
		}

		if progress != nil {
			progress(i+1, len(photos), filepath.Base(photo))
		}

		ext := strings.ToLower(filepath.Ext(photo))
		isRaw := rawExtensions[ext]
		name := filepath.Base(photo)

		if dc != nil {
			isDupe, firstSeen, hashErr := dc.check(photo)
			if hashErr == nil && isDupe {
				fmt.Fprintf(w, "  duplicato  %s\n         = %s\n\n", name, filepath.Base(firstSeen))
				stats.Dupes++
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

		categoria := "foto"
		if isRaw {
			categoria = "RAW "
		}

		// ── Rename-only mode ──────────────────────────────────────────────────
		if opts.RenameOnly {
			if !hasExif {
				fmt.Fprintf(w, "  saltato  %s  (nessuna data disponibile)\n\n", name)
				stats.Skipped++
				continue
			}

			camera := ""
			if needsCamera {
				camera = getExifCamera(photo)
			}

			var notes []string
			if opts.StripMeta && !isRaw && (ext == ".jpg" || ext == ".jpeg") {
				notes = append(notes, "metadati rimossi")
			}
			if modTimeUsed {
				notes = append(notes, "data da filesystem")
			}
			noteStr := ""
			if len(notes) > 0 {
				noteStr = "  [" + strings.Join(notes, ", ") + "]"
			}

			tpl := opts.FileTemplate
			if tpl == "" {
				tpl = "photo_{date}_{time}"
			}
			newName := buildFilename(tpl, dt, camera, ext)
			fmt.Fprintf(w, "  %s  %s%s\n         %s\n         → %s\n\n",
				categoria, name, noteStr, formatDatetimeHuman(dt), newName)

			if !opts.DryRun {
				if _, err := renameInPlace(photo, dt, camera, opts); err != nil {
					return stats, err
				}
			}
			stats.Moved++
			stats.ByYear[dt.Year()]++
			if isRaw {
				stats.Raw++
			} else {
				stats.Altri++
			}
			continue
		}

		// ── Organizzazione standard ───────────────────────────────────────────
		if !hasExif {
			dest := resolveConflict(filepath.Join(outputDir, "senza_data", name))
			rel, _ := filepath.Rel(outputDir, dest)
			fmt.Fprintf(w, "  senza data  %s\n         → %s\n\n", name, rel)
			if !opts.DryRun {
				if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
					return stats, err
				}
				if err := transferFile(photo, dest, opts); err != nil {
					return stats, err
				}
			}
			stats.Skipped++
			continue
		}

		camera := ""
		if needsCamera {
			camera = getExifCamera(photo)
		}

		dest := resolveConflict(buildDestPath(outputDir, dt, ext, isRaw, camera, opts))

		var notes []string
		if opts.StripMeta && !isRaw && (ext == ".jpg" || ext == ".jpeg") {
			notes = append(notes, "metadati rimossi")
		}
		if modTimeUsed {
			notes = append(notes, "data da filesystem")
		}
		noteStr := ""
		if len(notes) > 0 {
			noteStr = "  [" + strings.Join(notes, ", ") + "]"
		}

		rel, _ := filepath.Rel(outputDir, dest)
		fmt.Fprintf(w, "  %s  %s%s\n         %s\n         → %s\n\n",
			categoria, name, noteStr, formatDatetimeHuman(dt), rel)

		if !opts.DryRun {
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return stats, err
			}
			if err := transferFile(photo, dest, opts); err != nil {
				return stats, err
			}
		}

		stats.Moved++
		stats.ByYear[dt.Year()]++
		if isRaw {
			stats.Raw++
		} else {
			stats.Altri++
		}
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
