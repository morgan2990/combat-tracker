package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"combatapp/api"
	"combatapp/room"
	"combatapp/store"
	"combatapp/ws"
	"github.com/joho/godotenv"
)

const roomSweepInterval = 30 * time.Second

//go:embed all:frontend/dist
var frontendDist embed.FS

func main() {
	// Load .env for local development if present; silently no-op otherwise
	// (production sets real environment variables directly).
	_ = godotenv.Load()

	if err := store.Init(); err != nil {
		log.Fatalf("mongodb: %v", err)
	}
	store.InitMinio()
	store.InitTypesense()

	go func() {
		ticker := time.NewTicker(roomSweepInterval)
		defer ticker.Stop()
		for range ticker.C {
			room.Global.SweepDirty(&store.GlobalRooms)
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/signup", api.SignUp)
	mux.HandleFunc("POST /api/login", api.Login)
	mux.HandleFunc("POST /api/logout", api.Logout)
	mux.HandleFunc("GET /api/me", api.Me)
	mux.HandleFunc("POST /api/rooms", api.CreateRoom)
	mux.HandleFunc("GET /api/rooms/{room_id}", api.GetRoom)
	mux.HandleFunc("POST /api/pcs", api.CreatePC)
	mux.HandleFunc("PUT /api/pcs/{id}", api.UpdatePC)
	mux.HandleFunc("GET /api/pcs/{id}", api.GetPC)
	mux.HandleFunc("POST /api/pcs/{id}/companions", api.CreateCompanion)
	mux.HandleFunc("POST /api/monsters", api.CreateCustomMonster)
	// custom-monsters is a distinct top-level path from monsters/{name} so the
	// two route sets can never structurally overlap in Go's ServeMux (e.g.
	// monsters/{name}/pdf vs monsters/custom/{id} would otherwise conflict).
	mux.HandleFunc("GET /api/custom-monsters", api.ListMyCustomMonsters)
	mux.HandleFunc("GET /api/custom-monsters/{id}", api.GetCustomMonster)
	mux.HandleFunc("PUT /api/custom-monsters/{id}", api.UpdateCustomMonster)
	mux.HandleFunc("DELETE /api/custom-monsters/{id}", api.DeleteCustomMonster)
	mux.HandleFunc("GET /api/custom-monsters/{id}/pdf", api.StreamCustomMonsterPDF)
	mux.HandleFunc("GET /api/monsters/{name}", api.GetMonster)
	mux.HandleFunc("GET /api/search/monsters", api.SearchMonsters)
	mux.HandleFunc("GET /api/monsters/{name}/pdf", api.StreamMonsterPDF)
	mux.HandleFunc("POST /api/encounters", api.CreateEncounter)
	mux.HandleFunc("GET /api/encounters", api.ListMyEncounters)
	mux.HandleFunc("GET /api/encounters/{id}", api.GetEncounter)
	mux.HandleFunc("PUT /api/encounters/{id}", api.UpdateEncounter)
	mux.HandleFunc("DELETE /api/encounters/{id}", api.DeleteEncounter)
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
