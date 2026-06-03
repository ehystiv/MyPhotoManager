package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rwcarlsen/goexif/exif"
)

// viewableExtensions sono i formati immagine renderizzabili nativamente dal
// browser, usati dalla sezione di revisione (culling).
var viewableExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
}

// rawCullingExtensions sono i formati RAW da cui si può estrarre la miniatura
// JPEG incorporata nell'EXIF per la visualizzazione nel culling.
var rawCullingExtensions = map[string]bool{
	".arw": true, ".cr2": true, ".cr3": true, ".nef": true,
	".dng": true, ".raf": true, ".rw2": true, ".orf": true,
	".pef": true, ".srw": true, ".x3f": true,
}

func isCullingExt(ext string) bool {
	return viewableExtensions[ext] || rawCullingExtensions[ext]
}

// reviewFolder è la sottocartella della cartella di output in cui finiscono le
// foto marcate "review" quando si applicano le decisioni di revisione.
const reviewFolder = "_da_correggere"

// CullingMark è lo stato assegnato a una foto durante la revisione.
type CullingMark string

const (
	MarkDelete CullingMark = "delete"
	MarkReview CullingMark = "review"
	MarkOk     CullingMark = "ok"
)

func validMark(m string) bool {
	switch CullingMark(m) {
	case MarkDelete, MarkReview, MarkOk:
		return true
	}
	return false
}

// CullingPhoto descrive una foto rivedibile e l'eventuale marcatura corrente.
// Il contenuto immagine è caricato a parte e su richiesta tramite PhotoData.
type CullingPhoto struct {
	Path string `json:"path"` // path assoluto, chiave delle marcature
	Name string `json:"name"` // nome del file
	Rel  string `json:"rel"`  // path relativo alla radice, per la visualizzazione
	Mark string `json:"mark"` // "delete"|"review"|"ok"|"" se non marcata
}

// CullingListResult è il risultato di ListCullingPhotos.
type CullingListResult struct {
	Root   string         `json:"root"`
	Photos []CullingPhoto `json:"photos"`
	Err    string         `json:"err,omitempty"`
}

// CullingApplyResult riepiloga l'esito di ApplyCulling.
type CullingApplyResult struct {
	Deleted int    `json:"deleted"`
	Moved   int    `json:"moved"`
	Kept    int    `json:"kept"`
	Errors  int    `json:"errors"`
	DryRun  bool   `json:"dryRun"`
	Err     string `json:"err,omitempty"`
}

// cullingRoot restituisce la cartella su cui opera la revisione: l'output se
// impostato, altrimenti l'input.
func (a *App) cullingRoot() string {
	if a.prefs.OutputDir != "" {
		return a.prefs.OutputDir
	}
	return a.prefs.InputDir
}

// cullingMarksFile è il percorso del file di persistenza delle marcature,
// accanto a prefs.json (~/.myphoto/culling.json).
func (a *App) cullingMarksFile() string {
	if a.prefsPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(a.prefsPath), "culling.json")
}

func (a *App) loadCullingMarks() map[string]string {
	marks := map[string]string{}
	path := a.cullingMarksFile()
	if path == "" {
		return marks
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return marks
	}
	json.Unmarshal(data, &marks) //nolint:errcheck
	return marks
}

func (a *App) saveCullingMarks(marks map[string]string) {
	path := a.cullingMarksFile()
	if path == "" {
		return
	}
	data, _ := json.Marshal(marks)
	os.WriteFile(path, data, 0o644) //nolint:errcheck
}

// listCullablePhotos percorre la cartella di revisione raccogliendo i soli file
// in formato visualizzabile (jpg/png/webp), saltando la sottocartella di
// revisione _da_correggere e i file temporanei macOS.
func (a *App) listCullablePhotos() ([]CullingPhoto, error) {
	root := a.cullingRoot()
	if root == "" {
		return nil, nil
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	reviewAbs := filepath.Join(rootAbs, reviewFolder)
	marks := a.loadCullingMarks()

	var photos []CullingPhoto
	walkErr := filepath.Walk(rootAbs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			// Salta l'intera cartella di revisione.
			if path == reviewAbs {
				return filepath.SkipDir
			}
			return nil
		}
		name := info.Name()
		if strings.HasPrefix(name, "._") {
			return nil
		}
		if !isCullingExt(strings.ToLower(filepath.Ext(name))) {
			return nil
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(rootAbs, abs)
		if err != nil {
			rel = name
		}
		photos = append(photos, CullingPhoto{
			Path: abs,
			Name: name,
			Rel:  rel,
			Mark: marks[abs],
		})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	sort.Slice(photos, func(i, j int) bool { return photos[i].Path < photos[j].Path })
	return photos, nil
}

// applyCulling esegue le decisioni di revisione: cestina le "delete", sposta le
// "review" in _da_correggere, lascia le "ok". In dry-run conta soltanto. Al
// termine (solo non dry-run) conserva nel file le sole marcature non applicate.
func (a *App) applyCulling(dryRun bool) CullingApplyResult {
	res := CullingApplyResult{DryRun: dryRun}
	root := a.cullingRoot()
	if root == "" {
		res.Err = "nessuna cartella selezionata"
		return res
	}
	marks := a.loadCullingMarks()
	if len(marks) == 0 {
		return res
	}
	reviewDir := filepath.Join(root, reviewFolder)
	remaining := map[string]string{}

	for path, mark := range marks {
		// Salta marcature riferite a file non più esistenti.
		if _, err := os.Stat(path); err != nil {
			continue
		}
		switch CullingMark(mark) {
		case MarkDelete:
			if dryRun {
				res.Deleted++
				continue
			}
			if err := moveToTrash(path); err != nil {
				res.Errors++
				remaining[path] = mark
				continue
			}
			res.Deleted++
		case MarkReview:
			if dryRun {
				res.Moved++
				continue
			}
			if err := os.MkdirAll(reviewDir, 0o755); err != nil {
				res.Errors++
				remaining[path] = mark
				continue
			}
			dest := resolveConflict(filepath.Join(reviewDir, filepath.Base(path)))
			if err := moveFile(path, dest); err != nil {
				res.Errors++
				remaining[path] = mark
				continue
			}
			res.Moved++
		case MarkOk:
			res.Kept++
		}
	}

	if !dryRun {
		a.saveCullingMarks(remaining)
	}
	return res
}

// resolveCulling valida che path sia un file immagine (nativo o RAW) dentro la
// cartella di revisione e ne restituisce il percorso assoluto. Evita letture arbitrarie.
func (a *App) resolveCulling(path string) (string, bool) {
	if path == "" {
		return "", false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	root := a.cullingRoot()
	if root == "" {
		return "", false
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}
	if !isCullingExt(strings.ToLower(filepath.Ext(abs))) {
		return "", false
	}
	return abs, true
}

// PhotoMetaResult contiene i metadati EXIF principali di una foto.
type PhotoMetaResult struct {
	Date     string `json:"date,omitempty"`
	Camera   string `json:"camera,omitempty"`
	Lens     string `json:"lens,omitempty"`
	Focal    string `json:"focal,omitempty"`
	Aperture string `json:"aperture,omitempty"`
	Shutter  string `json:"shutter,omitempty"`
	ISO      string `json:"iso,omitempty"`
	Flash    bool   `json:"flash"`
	GPS      string `json:"gps,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Bias     string `json:"bias,omitempty"`
	Program  string `json:"program,omitempty"`
	Metering string `json:"metering,omitempty"`
	MaxAp    string `json:"maxAp,omitempty"`
}

// PhotoMeta legge i metadati EXIF di un file immagine nella cartella di revisione.
// Esposto al frontend.
func (a *App) PhotoMeta(path string) PhotoMetaResult {
	var res PhotoMetaResult
	abs, ok := a.resolveCulling(path)
	if !ok {
		return res
	}
	f, err := os.Open(abs)
	if err != nil {
		return res
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return res
	}

	if t, err := x.DateTime(); err == nil {
		res.Date = t.Format("02/01/2006 15:04")
	}

	make_ := strings.TrimSpace(exifString(x, exif.Make))
	model := strings.TrimSpace(exifString(x, exif.Model))
	// Rimuove il prefisso marca dal modello se già incluso (es. "SONY ILCE-7M3" → "ILCE-7M3").
	if make_ != "" && strings.HasPrefix(strings.ToUpper(model), strings.ToUpper(make_)) {
		model = strings.TrimSpace(model[len(make_):])
	}
	res.Camera = strings.Join(nonEmpty(make_, model), " ")
	res.Lens = exifString(x, exif.LensModel)

	if tag, err := x.Get(exif.FocalLength); err == nil {
		if mm, err := tag.Float(0); err == nil && mm > 0 {
			if mm == math.Trunc(mm) {
				res.Focal = fmt.Sprintf("%dmm", int(mm))
			} else {
				res.Focal = fmt.Sprintf("%.1fmm", mm)
			}
		}
	}

	if tag, err := x.Get(exif.FNumber); err == nil {
		if fn, err := tag.Float(0); err == nil && fn > 0 {
			res.Aperture = fmt.Sprintf("f/%.1g", fn)
		}
	}

	if tag, err := x.Get(exif.ExposureTime); err == nil {
		if secs, err := tag.Float(0); err == nil && secs > 0 {
			if secs >= 1 {
				res.Shutter = fmt.Sprintf("%gs", secs)
			} else {
				res.Shutter = fmt.Sprintf("1/%ds", int(math.Round(1/secs)))
			}
		}
	}

	if iso := exifInt(x, exif.ISOSpeedRatings); iso > 0 {
		res.ISO = fmt.Sprintf("%d", iso)
	}

	if tag, err := x.Get(exif.Flash); err == nil {
		if v, err := tag.Int(0); err == nil {
			res.Flash = (v & 0x1) != 0
		}
	}

	if lat, lon, err := x.LatLong(); err == nil {
		res.GPS = fmt.Sprintf("%.5f, %.5f", lat, lon)
	}

	res.Width = exifInt(x, exif.PixelXDimension)
	res.Height = exifInt(x, exif.PixelYDimension)

	if tag, err := x.Get(exif.ExposureBiasValue); err == nil {
		if v, err := tag.Float(0); err == nil && v != 0 {
			res.Bias = fmt.Sprintf("%+.1g EV", v)
		}
	}

	exposurePrograms := map[int]string{
		1: "Manuale", 2: "Auto", 3: "Priorità A", 4: "Priorità T",
		5: "Creativo", 6: "Sport", 7: "Ritratto", 8: "Paesaggio",
	}
	if v := exifInt(x, exif.ExposureProgram); v > 0 {
		if s, ok := exposurePrograms[v]; ok {
			res.Program = s
		}
	}

	meteringModes := map[int]string{
		1: "Media", 2: "Centro-pesata", 3: "Spot",
		4: "Multi-spot", 5: "Valutativa", 6: "Parziale",
	}
	if v := exifInt(x, exif.MeteringMode); v > 0 {
		if s, ok := meteringModes[v]; ok {
			res.Metering = s
		}
	}

	if tag, err := x.Get(exif.MaxApertureValue); err == nil {
		if apex, err := tag.Float(0); err == nil && apex > 0 {
			fn := math.Pow(2, apex/2)
			res.MaxAp = fmt.Sprintf("max f/%.1g", fn)
		}
	}

	return res
}

// PhotoData restituisce l'immagine come data-URL base64, caricata su richiesta.
// Per i formati RAW estrae la miniatura JPEG incorporata nell'EXIF.
// Restituisce stringa vuota se il path non è valido, la lettura fallisce
// o il RAW non contiene miniatura. Esposto al frontend.
func (a *App) PhotoData(path string) string {
	abs, ok := a.resolveCulling(path)
	if !ok {
		return ""
	}
	ext := strings.ToLower(filepath.Ext(abs))

	if viewableExtensions[ext] {
		data, err := os.ReadFile(abs)
		if err != nil {
			return ""
		}
		mime := "image/jpeg"
		switch ext {
		case ".png":
			mime = "image/png"
		case ".webp":
			mime = "image/webp"
		}
		return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
	}

	// Formato RAW: estrai la miniatura JPEG dall'EXIF.
	f, err := os.Open(abs)
	if err != nil {
		return ""
	}
	defer f.Close()
	x, err := exif.Decode(f)
	if err != nil {
		return ""
	}
	thumb, err := x.JpegThumbnail()
	if err != nil || len(thumb) == 0 {
		return ""
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(thumb)
}

// ListCullingPhotos restituisce le foto visualizzabili della cartella di output
// con le marcature correnti. Esposto al frontend.
func (a *App) ListCullingPhotos() CullingListResult {
	photos, err := a.listCullablePhotos()
	res := CullingListResult{Root: a.cullingRoot(), Photos: photos}
	if err != nil {
		res.Err = err.Error()
	}
	return res
}

// MarkPhoto salva la marcatura di una foto (path assoluto). Un mark vuoto la
// rimuove. Esposto al frontend.
func (a *App) MarkPhoto(path, mark string) {
	if path == "" {
		return
	}
	marks := a.loadCullingMarks()
	if mark == "" {
		delete(marks, path)
	} else if validMark(mark) {
		marks[path] = mark
	} else {
		return
	}
	a.saveCullingMarks(marks)
}

// ApplyCulling esegue le decisioni di revisione. Esposto al frontend.
func (a *App) ApplyCulling(dryRun bool) CullingApplyResult {
	return a.applyCulling(dryRun)
}

// ResetCullingMarks azzera tutte le marcature. Esposto al frontend.
func (a *App) ResetCullingMarks() {
	a.saveCullingMarks(map[string]string{})
}
