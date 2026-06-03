package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// HistoryEntry descrive una singola esecuzione di Organize.
type HistoryEntry struct {
	RunAt    time.Time `json:"runAt"`
	InputDir string    `json:"inputDir"`
	Moved    int       `json:"moved"`
	Raw      int       `json:"raw"`
	Others   int       `json:"others"`
	Skipped  int       `json:"skipped"`
	Dupes    int       `json:"dupes"`
}

const maxHistoryEntries = 100

func (a *App) historyFile() string {
	return filepath.Join(os.Getenv("HOME"), ".myphoto", "history.json")
}

func (a *App) loadHistory() []HistoryEntry {
	data, err := os.ReadFile(a.historyFile())
	if err != nil {
		return nil
	}
	var entries []HistoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil
	}
	return entries
}

func (a *App) appendHistory(entry HistoryEntry) {
	entries := a.loadHistory()
	entries = append(entries, entry)
	if len(entries) > maxHistoryEntries {
		entries = entries[len(entries)-maxHistoryEntries:]
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(a.historyFile(), data, 0o644)
}

// GetHistory restituisce le ultime esecuzioni, dalla più recente alla più vecchia.
// Esposto al frontend.
func (a *App) GetHistory() []HistoryEntry {
	entries := a.loadHistory()
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries
}
