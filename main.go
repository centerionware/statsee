package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"github.com/centerionware/statsee/src"
)

//go:embed static/*
var staticFS embed.FS

func main() {
	src.InitDB()
	defer src.CloseDB()

	log.Println("Using interface:", src.Iface)
	src.DebugNet()

	// Get embedded static sub-FS
	subFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}

	// SPA file server: serve files, fallback to index.html for unknown paths
	fileServer := http.FileServer(http.FS(subFS))
	http.Handle("/", spaHandler(subFS, fileServer))

	// API & WebSocket routes
	http.HandleFunc("/api/network-totals", src.HandleNetworkTotals)
	http.HandleFunc("/ws", src.WSHandler)

	// Start background collector
	go src.StartCollector()

	log.Println("StatSee running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// spaHandler serves static files from fsys, falls back to index.html for SPA routing
func spaHandler(fsys fs.FS, fsHandler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try to open the requested file
		f, err := fsys.Open(r.URL.Path)
		if err == nil {
			f.Close()
			fsHandler.ServeHTTP(w, r)
			return
		}

		// Fallback to index.html
		data, err := fs.ReadFile(fsys, "index.html")
		if err != nil {
			http.Error(w, "index.html not found", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(data)
	}
}