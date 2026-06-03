package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// DestFolder è una cartella-data all'interno di una categoria (es. "2024_12_25").
type DestFolder struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

// DestCategory raggruppa le cartelle-data per tipo di file (raw, jpg, senza_data…).
type DestCategory struct {
	Name    string       `json:"name"`
	Count   int          `json:"count"`
	Folders []DestFolder `json:"folders"`
}

// DestTreeResult è il risultato di PreviewTree.
type DestTreeResult struct {
	OutputDir  string         `json:"outputDir"`
	Categories []DestCategory `json:"categories"`
	Total      int            `json:"total"`
	Scanned    int            `json:"scanned"`
	Truncated  bool           `json:"truncated"`
	Err        string         `json:"err,omitempty"`
}

const maxTreeScan = 500

// collectAllPhotosForPreview raccoglie tutti i file immagine nella cartella,
// senza escludere le sottocartelle gestite. Serve solo per la visualizzazione
// della struttura di destinazione, non per l'organizzazione effettiva.
func collectAllPhotosForPreview(inputDir string) ([]string, error) {
	var photos []string
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		if strings.HasPrefix(name, "._") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(name))
		if rawExtensions[ext] || otherExtensions[ext] {
			photos = append(photos, path)
		}
		return nil
	})
	return photos, err
}

// PreviewTree calcola la struttura di cartelle che verrebbe creata organizzando
// le foto nella cartella di input con le preferenze correnti.
// Analizza al massimo maxTreeScan file in parallelo. Esposto al frontend.
func (a *App) PreviewTree(p Prefs) DestTreeResult {
	outputDir := p.OutputDir
	if outputDir == "" {
		outputDir = p.InputDir
	}
	res := DestTreeResult{OutputDir: outputDir}
	if p.InputDir == "" {
		return res
	}

	photos, err := collectAllPhotosForPreview(p.InputDir)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	res.Scanned = len(photos)

	if len(photos) > maxTreeScan {
		photos = photos[:maxTreeScan]
		res.Truncated = true
	}

	folderFmt := p.FolderFmt
	if folderFmt == "" {
		folderFmt = "2006_01_02"
	}

	type photoInfo struct {
		cat    string
		folder string
	}
	infos := make([]photoInfo, len(photos))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, photo := range photos {
		wg.Add(1)
		go func(i int, photo string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			ext := strings.ToLower(filepath.Ext(photo))
			isRaw := rawExtensions[ext]
			cat := categoryFor(ext, isRaw)

			dt, hasExif := getExifDatetime(photo)
			if !hasExif && p.ModTime {
				if info, e := os.Stat(photo); e == nil {
					dt = info.ModTime()
					hasExif = true
				}
			}

			folder := "senza_data"
			if hasExif {
				folder = dt.Format(folderFmt)
			}
			infos[i] = photoInfo{cat: cat, folder: folder}
		}(i, photo)
	}
	wg.Wait()

	type key struct{ cat, folder string }
	counts := map[key]int{}
	var catOrder []string
	catSeen := map[string]bool{}

	for _, info := range infos {
		k := key{info.cat, info.folder}
		counts[k]++
		res.Total++
		if !catSeen[info.cat] {
			catSeen[info.cat] = true
			catOrder = append(catOrder, info.cat)
		}
	}

	sort.Slice(catOrder, func(i, j int) bool {
		if catOrder[i] == "raw" {
			return true
		}
		if catOrder[j] == "raw" {
			return false
		}
		return catOrder[i] < catOrder[j]
	})

	for _, cat := range catOrder {
		dc := DestCategory{Name: cat}
		dateMap := map[string]int{}
		for k, c := range counts {
			if k.cat == cat {
				dc.Count += c
				dateMap[k.folder] += c
			}
		}
		dates := make([]string, 0, len(dateMap))
		for d := range dateMap {
			dates = append(dates, d)
		}
		sort.Slice(dates, func(i, j int) bool {
			if dates[i] == "senza_data" {
				return false
			}
			if dates[j] == "senza_data" {
				return true
			}
			return dates[i] < dates[j]
		})
		for _, d := range dates {
			dc.Folders = append(dc.Folders, DestFolder{Path: d, Count: dateMap[d]})
		}
		res.Categories = append(res.Categories, dc)
	}

	return res
}
