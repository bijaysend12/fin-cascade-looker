package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

type SQLiteClient struct {
	DB *sql.DB
}

func NewSQLiteClient(path string) (*SQLiteClient, error) {
	db, err := sql.Open("sqlite", path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &SQLiteClient{DB: db}, nil
}

func (c *SQLiteClient) Close() {
	c.DB.Close()
}
