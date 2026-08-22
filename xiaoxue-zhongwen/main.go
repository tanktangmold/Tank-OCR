package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"xiaoxue-zhongwen/handler"
	"xiaoxue-zhongwen/learn"
)

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = "0.0.0.0:8080"
	}
	if path := os.Getenv("CLASSROOM_PATH"); path != "" {
		learn.SetClassroomPath(path)
	} else {
		wd, _ := os.Getwd()
		learn.SetClassroomPath(filepath.Join(wd, "data", "classroom.json"))
	}

	staticDir := getStaticDir()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/catalog", handler.HandleCatalog)
	mux.HandleFunc("/api/interest", handler.HandleInterest)
	mux.HandleFunc("/api/books", handler.HandleBooks)
	mux.HandleFunc("/api/story/{id}", handler.HandleStory)
	mux.HandleFunc("/api/course", handler.HandleCourse)
	mux.HandleFunc("/api/course/day/{day}", handler.HandleCourseDay)
	mux.HandleFunc("/api/children", handler.HandleChildren)
	mux.HandleFunc("/api/children/{id}/progress", handler.HandleChildProgress)
	mux.HandleFunc("/api/game/{day}", handler.HandleGame)
	mux.HandleFunc("/api/server-info", handler.HandleServerInfo)
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
