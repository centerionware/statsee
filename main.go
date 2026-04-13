package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/centerionware/statsee/src"
)

//go:embed static/*
var staticFS embed.FS

func main() {
	src.InitDB()
	defer src.CloseDB()

	log.Println("Using interface:", src.Iface)
	src.DebugNet()

	subFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}

	fileServer := http.FileServer(http.FS(subFS))
	http.Handle("/", spaHandler(subFS, fileServer))

	// API
	http.HandleFunc("/api/network-totals", src.HandleNetworkTotals) // compatibility
	http.HandleFunc("/api/network-live", src.HandleNetworkLive)
	http.HandleFunc("/api/network-history", src.HandleNetworkHistory)

	http.HandleFunc("/ws", src.WSHandler)

	go src.StartCollector()

	log.Println("StatSee running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func spaHandler(fsys fs.FS, fsHandler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")

		f, err := fsys.Open(path)
		if err == nil {
			f.Close()
			fsHandler.ServeHTTP(w, r)
			return
		}

		data, err := fs.ReadFile(fsys, "index.html")
		if err != nil {
			http.Error(w, "index.html not found", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(data)
	}
}