package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/coachengo/fin-cascade-looker/internal/db"
)

type contextKey string

const UserContextKey contextKey = "user"

type Handler struct {
	Neo4j  *db.Neo4jClient
	SQLite *db.SQLiteClient
	PG     *db.PGClient
}

func New(n *db.Neo4jClient, s *db.SQLiteClient, pg *db.PGClient) *Handler {
	return &Handler{Neo4j: n, SQLite: s, PG: pg}
}

func isAdmin(r *http.Request) bool {
	u, ok := r.Context().Value(UserContextKey).(*db.User)
	return ok && u != nil && u.IsAdmin
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func queryInt(r *http.Request, key string, fallback int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	if key == "limit" && n > 100 {
		return 100
	}
	if n < 0 {
		return fallback
	}
	return n
}

func queryStr(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}
