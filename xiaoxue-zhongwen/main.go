package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"xiaoxue-zhongwen/handler"
)

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}

	staticDir := getStaticDir()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/catalog", handler.HandleCatalog)
	mux.HandleFunc("/api/interest", handler.HandleInterest)
	mux.HandleFunc("/api/books", handler.HandleBooks)
	mux.HandleFunc("/api/story/{id}", handler.HandleStory)
	mux.Handle("/", http.FileServer(http.Dir(staticDir)))

	log.Printf("小小中文 listening on http://%s", addr)
	log.Printf("Static files: %s", staticDir)

	server := &http.Server{
		Addr:              addr,
		Handler:           withLocalHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func getStaticDir() string {
	exePath, _ := os.Executable()
	dir := filepath.Join(filepath.Dir(exePath), "static")
	if _, err := os.Stat(dir); err == nil {
		return dir
	}
	wd, _ := os.Getwd()
	return filepath.Join(wd, "static")
}

func withLocalHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
