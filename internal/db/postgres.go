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

func (c *PGClient) UpsertUser(firebaseUID, email, name, avatarURL string) (*User, error) {
	var u User
	err := c.DB.QueryRow(`
		INSERT INTO users (firebase_uid, email, name, avatar_url, last_login)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (firebase_uid) DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			avatar_url = EXCLUDED.avatar_url,
			last_login = NOW()
		RETURNING id, firebase_uid, email, COALESCE(name,''), COALESCE(avatar_url,''), is_admin, created_at, last_login
	`, firebaseUID, email, name, avatarURL).Scan(
		&u.ID, &u.FirebaseUID, &u.Email, &u.Name, &u.AvatarURL, &u.IsAdmin, &u.CreatedAt, &u.LastLogin,
	)
	return &u, err
}
