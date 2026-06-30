package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"combatapp/api"
	"combatapp/ws"
)

//go:embed all:frontend/dist
var frontendDist embed.FS

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/rooms", api.CreateRoom)
	mux.HandleFunc("/ws", ws.Handler)

	distFS, err := fs.Sub(frontendDist, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", newSPAHandler(http.FS(distFS)))

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// spaHandler serves static files and falls back to index.html for client-side routes.
type spaHandler struct {
	fs   http.FileSystem
	base http.Handler
}

func newSPAHandler(fsys http.FileSystem) http.Handler {
	return &spaHandler{fs: fsys, base: http.FileServer(fsys)}
}

func (s *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Let the API and WS routes through (handled by mux before this)
	if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/ws" {
		http.NotFound(w, r)
		return
	}
	// Try to open the requested file
	f, err := s.fs.Open(r.URL.Path)
	if err == nil {
		f.Close()
		s.base.ServeHTTP(w, r)
		return
	}
	// Not a real file — serve index.html for React Router
	r2 := *r
	r2.URL.Path = "/index.html"
	s.base.ServeHTTP(w, &r2)
}
