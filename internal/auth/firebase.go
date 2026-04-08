package auth

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

type FirebaseVerifier struct {
	ProjectID     string
	AllowedEmails map[string]bool
	keys          map[string]*rsa.PublicKey
	keysExpiry    time.Time
	mu            sync.RWMutex
}

type Claims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Issuer        string `json:"iss"`
	Audience      string `json:"aud"`
	Subject       string `json:"sub"`
	IssuedAt      int64  `json:"iat"`
	ExpiresAt     int64  `json:"exp"`
}

func NewFirebaseVerifier(projectID string, allowedEmails []string) *FirebaseVerifier {
	emails := make(map[string]bool)
	for _, e := range allowedEmails {
		e = strings.TrimSpace(e)
		if e != "" {
			emails[e] = true
		}
	}
	return &FirebaseVerifier{
		ProjectID:     projectID,
		AllowedEmails: emails,
	}
}

func (v *FirebaseVerifier) VerifyToken(tokenStr string) (*Claims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	headerJSON, err := b64Decode(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}
	var header struct {
		Kid string `json:"kid"`
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}
	if header.Alg != "RS256" {
		return nil, fmt.Errorf("unexpected algorithm: %s", header.Alg)
	}

	claimsJSON, err := b64Decode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}
	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}

	now := time.Now().Unix()
	if claims.ExpiresAt < now {
		return nil, fmt.Errorf("token expired")
	}
	if claims.IssuedAt > now+300 {
		return nil, fmt.Errorf("token issued in the future")
	}
	if claims.Issuer != "https://securetoken.google.com/"+v.ProjectID {
		return nil, fmt.Errorf("invalid issuer")
	}
	if claims.Audience != v.ProjectID {
		return nil, fmt.Errorf("invalid audience")
	}
	if claims.Subject == "" {
		return nil, fmt.Errorf("empty subject")
	}

	key, err := v.getPublicKey(header.Kid)
	if err != nil {
		return nil, fmt.Errorf("get public key: %w", err)
	}

	sigBytes, err := b64Decode(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}
	hash := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, hash[:], sigBytes); err != nil {
		return nil, fmt.Errorf("invalid signature")
	}

	if len(v.AllowedEmails) > 0 && !v.AllowedEmails[claims.Email] {
		return nil, fmt.Errorf("email %s not allowed", claims.Email)
	}

	return &claims, nil
}

func (v *FirebaseVerifier) getPublicKey(kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	if v.keys != nil && time.Now().Before(v.keysExpiry) {
		if key, ok := v.keys[kid]; ok {
			v.mu.RUnlock()
			return key, nil
		}
	}
	v.mu.RUnlock()

	v.mu.Lock()
	defer v.mu.Unlock()

	if v.keys != nil && time.Now().Before(v.keysExpiry) {
		if key, ok := v.keys[kid]; ok {
			return key, nil
		}
	}

	resp, err := http.Get("https://www.googleapis.com/service_accounts/v1/jwk/securetoken@system.gserviceaccount.com")
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	var jwkSet struct {
		Keys []struct {
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwkSet); err != nil {
		return nil, fmt.Errorf("parse jwks: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, k := range jwkSet.Keys {
		nBytes, err := b64Decode(k.N)
		if err != nil {
			continue
		}
		eBytes, err := b64Decode(k.E)
		if err != nil {
			continue
		}
		n := new(big.Int).SetBytes(nBytes)
		e := 0
		for _, b := range eBytes {
			e = e<<8 + int(b)
		}
		keys[k.Kid] = &rsa.PublicKey{N: n, E: e}
	}

	v.keys = keys
	v.keysExpiry = time.Now().Add(1 * time.Hour)

	key, ok := keys[kid]
	if !ok {
		return nil, fmt.Errorf("key %s not found", kid)
	}
	return key, nil
}

func b64Decode(s string) ([]byte, error) {
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.StdEncoding.DecodeString(s)
}
