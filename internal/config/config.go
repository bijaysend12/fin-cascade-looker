package config

import (
	"os"
	"path/filepath"
	"github.com/joho/godotenv"
)

type Config struct {
	Neo4jURI          string
	Neo4jUser         string
	Neo4jPassword     string
	SQLitePath        string
	PostgresDSN       string
	Port              string
	ProjectDir        string
	ServerKey  string
	JWTSecret  string
	CORSOrigin string
}

func Load() *Config {
	projectDir := os.Getenv("FIN_CASCADE_LOOKER_DIR")
	if projectDir == "" {
		exe, err := os.Executable()
		if err == nil {
			projectDir = filepath.Dir(filepath.Dir(exe))
		}
	}

	_ = godotenv.Load(filepath.Join(projectDir, ".env"))

	return &Config{
		Neo4jURI:      getEnv("NEO4J_URI", "bolt://localhost:7687"),
		Neo4jUser:     getEnv("NEO4J_USER", "neo4j"),
		Neo4jPassword: getEnv("NEO4J_PASSWORD", "fincascade123"),
		SQLitePath:    getEnv("SQLITE_PATH", "/home/kanshi/project/fin-cascade/data/news.db"),
		PostgresDSN:   getEnv("POSTGRES_DSN", "postgres://fincascade:fincascade123@localhost:5432/fincascade?sslmode=disable"),
		Port:          getEnv("PORT", "8080"),
		ProjectDir:    projectDir,
		ServerKey:  getEnv("SERVER_KEY", ""),
		JWTSecret:  getEnv("JWT_SECRET", ""),
		CORSOrigin: getEnv("CORS_ORIGIN", "http://localhost:5173"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
