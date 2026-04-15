package db

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

type PGClient struct {
	DB *sql.DB
}

type User struct {
	ID          int
	FirebaseUID string
	Email       string
	Name        string
	AvatarURL   string
	IsAdmin     bool
	CreatedAt   time.Time
	LastLogin   time.Time
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

func (c *PGClient) GetUserByUID(uid string) (*User, error) {
	var u User
	err := c.DB.QueryRow(`
		SELECT id, firebase_uid, email, COALESCE(name,''), COALESCE(avatar_url,''), is_admin, created_at, last_login
		FROM users WHERE firebase_uid = $1
	`, uid).Scan(
		&u.ID, &u.FirebaseUID, &u.Email, &u.Name, &u.AvatarURL, &u.IsAdmin, &u.CreatedAt, &u.LastLogin,
	)
	return &u, err
}

func (c *PGClient) RegisterUser(userID, name, email, authType string) (*User, error) {
	var u User
	err := c.DB.QueryRow(`
		INSERT INTO users (firebase_uid, email, name, last_login)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (firebase_uid) DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			last_login = NOW()
		RETURNING id, firebase_uid, email, COALESCE(name,''), COALESCE(avatar_url,''), is_admin, created_at, last_login
	`, userID, email, name).Scan(
		&u.ID, &u.FirebaseUID, &u.Email, &u.Name, &u.AvatarURL, &u.IsAdmin, &u.CreatedAt, &u.LastLogin,
	)
	return &u, err
}
