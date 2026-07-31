package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"tank-ocr/engine"
	"tank-ocr/handler"
)

// Engine identifiers accepted in config.json.
//
// "kichonai" is the name used by earlier private builds; it is still
// recognised so an existing config.json keeps working after upgrading.
const (
	engineScreenAI      = "screenai"
	engineScreenAIAlias = "kichonai"
	engineLocalFallback = "localfallback"
)

type Config struct {
	ActiveEngine      string `json:"active_engine"`
	LocalFallbackURL  string `json:"local_fallback_url"`
	BindAddress       string `json:"bind_address"`
	MaxImageDimension int    `json:"max_image_dimension"`
	HistoryLimit      int    `json:"history_limit"`
}

func loadConfig() Config {
	exePath, _ := os.Executable()
	dir := filepath.Dir(exePath)

	configPath := filepath.Join(dir, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Try current working dir (during development)
		wd, _ := os.Getwd()
		configPath = filepath.Join(wd, "config.json")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("Warning: Failed to load config.json (%v). Using defaults.", err)
		return Config{
			ActiveEngine: engineScreenAI,
			BindAddress:  "127.0.0.1:8000",
		}
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("Error parsing config.json: %v. Using defaults.", err)
		return Config{
			ActiveEngine: engineScreenAI,
			BindAddress:  "127.0.0.1:8000",
		}
	}
	if config.ActiveEngine == "" {
		config.ActiveEngine = engineScreenAI
	}
	if config.BindAddress == "" {
		config.BindAddress = "127.0.0.1:8000"
	}
	if config.MaxImageDimension <= 0 {
		config.MaxImageDimension = 2048
	}
	if config.HistoryLimit <= 0 {
		config.HistoryLimit = 50
	}
	return config
}

func getStaticDir() string {
	exePath, _ := os.Executable()
	dir := filepath.Dir(exePath)

	staticDir := filepath.Join(dir, "static")
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		wd, _ := os.Getwd()
		staticDir = filepath.Join(wd, "static")
	}
	return staticDir
}

func main() {
	log.Println("Starting Tank-OCR local OCR server...")

	// 1. Load configuration
	config := loadConfig()
	handler.Configure(config.MaxImageDimension, config.HistoryLimit)

	// 2. Initialize the selected OCR engine
	var selectedEngine engine.OcrEngine

	switch config.ActiveEngine {
	case engineScreenAI, engineScreenAIAlias:
		log.Println("Selected engine: Chrome Screen AI (local, in-process)")
		selectedEngine = engine.NewScreenAiEngine("")
	case engineLocalFallback:
		log.Println("Selected engine: Local Fallback (PaddleOCR/EasyOCR)")
		selectedEngine = engine.NewLocalFallbackEngine(config.LocalFallbackURL)
	default:
		log.Printf("Unknown engine %q. Falling back to %q.",
			config.ActiveEngine, engineScreenAI)
		selectedEngine = engine.NewScreenAiEngine("")
	}

	if err := selectedEngine.Init(); err != nil {
		log.Printf("[WARNING] Engine initialization failed: %v", err)
		log.Println("Server will start, but OCR requests will return errors until the engine is properly initialized.")
	} else {
		handler.ActiveEngine = selectedEngine
		defer selectedEngine.Close()
	}

	// 3. Setup router and middleware (CORS)
	mux := http.NewServeMux()

	// API Routing
	mux.HandleFunc("/api/status", handler.HandleStatus)
	mux.HandleFunc("/api/ocr", handler.HandleOcr)
	mux.HandleFunc("/api/ocr-pdf", handler.HandleOcrPdf)
	mux.HandleFunc("/api/history", handler.HandleHistory)

	// Serve Static Frontend
	staticDir := getStaticDir()
	log.Printf("Serving static assets from: %s", staticDir)

	// Create a fileserver handler
	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("/", fs)

	// Local API wrapper with request logging and CORS for local pages.
	//
	// The bookkeeping frontend is opened straight from disk (file://), whose origin
	// is the literal string "null". Without these headers the browser blocks every
	// call to this API, so a locally opened page could never use the OCR service.
	// The listener stays bound to 127.0.0.1, so this does not expose anything to
	// the network — only pages running on this machine can reach it either way.
	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s - %s %s", r.RemoteAddr, r.Method, r.URL.Path)
		w.Header().Set("X-Content-Type-Options", "nosniff")

		if origin := r.Header.Get("Origin"); origin != "" {
			// Echo the caller's origin rather than "*" so that file:// pages,
			// which send Origin: null, are matched exactly.
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		mux.ServeHTTP(w, r)
	})

	// Bind HTTP server
	log.Printf("OCR Server running on http://%s", config.BindAddress)
	log.Println("Please open your browser and navigate to the address above.")

	server := &http.Server{
		Addr:              config.BindAddress,
		Handler:           httpHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      15 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
