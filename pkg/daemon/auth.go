package daemon

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AuthMethod represents an authentication method.
type AuthMethod string

const (
	AuthMethodNone   AuthMethod = "none"
	AuthMethodAPIKey AuthMethod = "api_key"
	AuthMethodJWT    AuthMethod = "jwt"
)

// AuthConfig configures authentication.
type AuthConfig struct {
	Method AuthMethod
	APIKey string // For API key auth
	Secret string // For JWT auth
}

// Auth provides authentication middleware.
type Auth struct {
	config AuthConfig
	tokens map[string]*TokenInfo // For session tokens
	mu     sync.RWMutex
}

// TokenInfo stores information about an authentication token.
type TokenInfo struct {
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
	Metadata  map[string]string
}

// NewAuth creates a new Auth instance.
func NewAuth(config AuthConfig) *Auth {
	a := &Auth{
		config: config,
		tokens: make(map[string]*TokenInfo),
	}

	// Generate API key if not provided
	if config.Method == AuthMethodAPIKey && config.APIKey == "" {
		a.config.APIKey = generateAPIKey()
	}

	return a
}

// Middleware returns an HTTP middleware that enforces authentication.
func (a *Auth) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.config.Method == AuthMethodNone {
			next(w, r)
			return
		}

		if !a.authenticate(r) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		next(w, r)
	}
}

// authenticate checks if the request is authenticated.
func (a *Auth) authenticate(r *http.Request) bool {
	switch a.config.Method {
	case AuthMethodNone:
		return true
	case AuthMethodAPIKey:
		return a.authenticateAPIKey(r)
	case AuthMethodJWT:
		return a.authenticateJWT(r)
	default:
		return false
	}
}

// authenticateAPIKey checks API key authentication.
func (a *Auth) authenticateAPIKey(r *http.Request) bool {
	// Check Authorization header
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		return subtle.ConstantTimeCompare([]byte(token), []byte(a.config.APIKey)) == 1
	}

	// Check X-API-Key header
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		return subtle.ConstantTimeCompare([]byte(apiKey), []byte(a.config.APIKey)) == 1
	}

	// Check query parameter (less secure, but convenient for testing)
	apiKey = r.URL.Query().Get("api_key")
	if apiKey != "" {
		return subtle.ConstantTimeCompare([]byte(apiKey), []byte(a.config.APIKey)) == 1
	}

	return false
}

// authenticateJWT checks JWT authentication.
func (a *Auth) authenticateJWT(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return false
	}

	token := strings.TrimPrefix(auth, "Bearer ")

	// Check if it's a session token
	a.mu.RLock()
	info, ok := a.tokens[token]
	a.mu.RUnlock()

	if ok {
		if time.Now().After(info.ExpiresAt) {
			a.mu.Lock()
			delete(a.tokens, token)
			a.mu.Unlock()
			return false
		}
		return true
	}

	// For full JWT validation, you'd verify the signature here
	// This is a simplified implementation
	return false
}

// GenerateToken creates a new session token.
func (a *Auth) GenerateToken(ttl time.Duration, metadata map[string]string) *TokenInfo {
	token := generateAPIKey()
	now := time.Now()

	info := &TokenInfo{
		Token:     token,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
		Metadata:  metadata,
	}

	a.mu.Lock()
	a.tokens[token] = info
	a.mu.Unlock()

	return info
}

// RevokeToken removes a session token.
func (a *Auth) RevokeToken(token string) {
	a.mu.Lock()
	delete(a.tokens, token)
	a.mu.Unlock()
}

// GetAPIKey returns the current API key.
func (a *Auth) GetAPIKey() string {
	return a.config.APIKey
}

// CleanupExpiredTokens removes expired tokens.
func (a *Auth) CleanupExpiredTokens() {
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()

	for token, info := range a.tokens {
		if now.After(info.ExpiresAt) {
			delete(a.tokens, token)
		}
	}
}

// generateAPIKey generates a random API key.
func generateAPIKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based key (less secure)
		return hex.EncodeToString([]byte(time.Now().String()))[:64]
	}
	return hex.EncodeToString(bytes)
}
