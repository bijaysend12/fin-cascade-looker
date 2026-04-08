package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/coachengo/fin-cascade-looker/internal/config"
	"github.com/coachengo/fin-cascade-looker/internal/db"
	"github.com/coachengo/fin-cascade-looker/internal/handlers"
)

func main() {
	cfg := config.Load()

	if cfg.ServerKey == "" {
		fmt.Fprintf(os.Stderr, "WARNING: SERVER_KEY not set — API is unprotected\n")
	}

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

	var pg *db.PGClient
	pg, err = db.NewPGClient(cfg.PostgresDSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PostgreSQL connection failed (analysis data won't be available): %s\n", err)
	} else {
		defer pg.Close()
		fmt.Fprintf(os.Stderr, "Connected to PostgreSQL\n")
	}

	h := handlers.New(neo4j, sqlite, pg)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/stats", h.GetStats)
	mux.HandleFunc("GET /api/companies", h.ListCompanies)
	mux.HandleFunc("GET /api/companies/{ticker}", h.GetCompany)
	mux.HandleFunc("GET /api/companies/{ticker}/graph", h.GetCompanyGraph)
	mux.HandleFunc("GET /api/sectors", h.ListSectors)
	mux.HandleFunc("GET /api/news", h.ListNews)
	mux.HandleFunc("GET /api/news/stats", h.GetNewsStats)
	mux.HandleFunc("GET /api/scans", h.ListScans)
	mux.HandleFunc("GET /api/analysis/scans", h.ListAnalysisScans)
	mux.HandleFunc("GET /api/analysis/scans/{id}", h.GetAnalysisScan)
	mux.HandleFunc("GET /api/analysis/signals", h.ListSignals)
	mux.HandleFunc("GET /api/analysis/stats", h.GetAnalysisStats)

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

	handler := corsMiddleware(cfg.CORSOrigin, authMiddleware(cfg.ServerKey, mux))

	addr := "127.0.0.1:" + cfg.Port
	fmt.Fprintf(os.Stderr, "Server listening on http://%s\n", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %s\n", err)
		os.Exit(1)
	}
}

func authMiddleware(serverKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		if serverKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		token := r.Header.Get("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = token[7:]
		} else {
			token = r.Header.Get("X-Server-Key")
		}

		if token != serverKey {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Server-Key")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
