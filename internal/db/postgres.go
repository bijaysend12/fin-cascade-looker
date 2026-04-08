package db

import (
	"database/sql"

	_ "github.com/lib/pq"
)

type PGClient struct {
	DB *sql.DB
}

func NewPGClient(dsn string) (*PGClient, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	db.SetMaxOpenConns(5)
	return &PGClient{DB: db}, nil
}

func (c *PGClient) Close() {
	c.DB.Close()
}
