package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type HistoryEntry struct {
	ID        string  `json:"id"`
	Filename  string  `json:"filename"`
	Type      string  `json:"type"`
	Size      int     `json:"size"`
	Duration  float64 `json:"duration"`
	Pages     int     `json:"pages"`
	Timestamp string  `json:"timestamp"`
}

var (
	historyMutex sync.Mutex
	historyFile  = "history.json"
)

func getHistoryPath() string {
	exePath, _ := os.Executable()
	dir := filepath.Dir(exePath)

	// Check if we are running via go run
	if _, err := os.Stat(filepath.Join(dir, "config.json")); err != nil {
		wd, _ := os.Getwd()
		return filepath.Join(wd, historyFile)
	}
	return filepath.Join(dir, historyFile)
}

func loadHistory() []HistoryEntry {
	path := getHistoryPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []HistoryEntry{}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Failed to read history file: %v", err)
		return []HistoryEntry{}
	}

	var history []HistoryEntry
	if err := json.Unmarshal(data, &history); err != nil {
		log.Printf("Failed to unmarshal history: %v", err)
		return []HistoryEntry{}
	}
	return history
}

func saveHistory(history []HistoryEntry) {
	path := getHistoryPath()
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal history: %v", err)
		return
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		log.Printf("Failed to write history file: %v", err)
	}
}

func AddHistoryEntry(filename string, fileType string, size int, duration float64, pages int) {
	historyMutex.Lock()
	defer historyMutex.Unlock()

	history := loadHistory()

	// Create new entry
	entry := HistoryEntry{
		ID:        time.Now().Format("20060102150405.000"), // Simple unique timestamp ID
		Filename:  filename,
		Type:      fileType,
		Size:      size,
		Duration:  duration,
		Pages:     pages,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}

	// Prepend
	history = append([]HistoryEntry{entry}, history...)

	if len(history) > HistoryLimit {
		history = history[:HistoryLimit]
	}

	saveHistory(history)
}

func HandleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		historyMutex.Lock()
		history := loadHistory()
		historyMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(history)
		return
	}

	if r.Method == http.MethodDelete {
		historyMutex.Lock()
		saveHistory([]HistoryEntry{})
		historyMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "history cleared"})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
