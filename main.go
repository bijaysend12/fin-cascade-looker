package main

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/coachengo/fin-cascade-looker/internal/config"
	"github.com/coachengo/fin-cascade-looker/internal/db"
	"github.com/coachengo/fin-cascade-looker/internal/handlers"
)

func main() {
	cfg := config.Load()

	neo4j, err := db.NewNeo4jClient(cfg.Neo4jURI, cfg.Neo4jUser, cfg.Neo4jPassword)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Neo4j connection failed: %s\n", err)
		os.Exit(1)
	}
	defer neo4j.Close()
	fmt.Fprintf(os.Stderr, "Connected to Neo4j at %s\n", cfg.Neo4jURI)

	var sqlite *db.SQLiteClient
	sqlite, err = db.NewSQLiteClient(cfg.SQLitePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SQLite open failed (news data won't be available): %s\n", err)
	} else {
		defer sqlite.Close()
		fmt.Fprintf(os.Stderr, "Connected to SQLite at %s\n", cfg.SQLitePath)
	}

	h := handlers.New(neo4j, sqlite)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/stats", h.GetStats)
	mux.HandleFunc("GET /api/companies", h.ListCompanies)
	mux.HandleFunc("GET /api/companies/{ticker}", h.GetCompany)
	mux.HandleFunc("GET /api/companies/{ticker}/graph", h.GetCompanyGraph)
	mux.HandleFunc("GET /api/sectors", h.ListSectors)
	mux.HandleFunc("GET /api/news", h.ListNews)
	mux.HandleFunc("GET /api/news/stats", h.GetNewsStats)
	mux.HandleFunc("GET /api/scans", h.ListScans)

	distDir := filepath.Join(cfg.ProjectDir, "frontend", "dist")
	if _, err := os.Stat(distDir); err == nil {
		frontendFS := http.FileServer(http.Dir(distDir))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			path := filepath.Join(distDir, r.URL.Path)
			if _, err := fs.Stat(os.DirFS(distDir), r.URL.Path[1:]); err != nil {
				http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
				return
			}
			_ = path
			frontendFS.ServeHTTP(w, r)
		})
		fmt.Fprintf(os.Stderr, "Serving frontend from %s\n", distDir)
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "fin-cascade-looker API running. Frontend not built yet — run: cd frontend && npm run build")
		})
	}

	corsHandler := cors(mux)

	addr := ":" + cfg.Port
	fmt.Fprintf(os.Stderr, "Server listening on http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, corsHandler); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %s\n", err)
		os.Exit(1)
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
