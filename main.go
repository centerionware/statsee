package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

    "./src/cpu"
    "./src/ram"
    "./src/disk"
    "./src/network"
    "./src/speedtest"
    "./src/db"

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

	http.HandleFunc("/", serveIndex(subFS))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(subFS))))

	http.HandleFunc("/api/network-totals", src.HandleNetworkTotals)
	http.HandleFunc("/ws", src.WSHandler)

	go src.StartCollector()

	log.Println("StatSee running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func serveIndex(fsys fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(fsys, "index.html")
		if err != nil {
			http.Error(w, "index.html not found", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(data)
	}
}