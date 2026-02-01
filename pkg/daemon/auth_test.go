package daemon

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewAuth(t *testing.T) {
	tests := []struct {
		name     string
		config   AuthConfig
		checkKey bool
	}{
		{
			name: "no auth",
			config: AuthConfig{
				Method: AuthMethodNone,
			},
			checkKey: false,
		},
		{
			name: "api key auth with provided key",
			config: AuthConfig{
				Method: AuthMethodAPIKey,
				APIKey: "my-secret-key",
			},
			checkKey: true,
		},
		{
			name: "api key auth without key generates one",
			config: AuthConfig{
				Method: AuthMethodAPIKey,
				APIKey: "",
			},
			checkKey: true,
		},
		{
			name: "jwt auth",
			config: AuthConfig{
				Method: AuthMethodJWT,
				Secret: "jwt-secret",
			},
			checkKey: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewAuth(tt.config)

			if auth == nil {
				t.Fatal("NewAuth() returned nil")
			}

			if auth.tokens == nil {
				t.Error("NewAuth() did not initialize tokens map")
			}

			if tt.checkKey && auth.config.APIKey == "" {
				t.Error("NewAuth() did not generate API key when expected")
			}

			if tt.config.APIKey != "" && auth.config.APIKey != tt.config.APIKey {
				t.Errorf("NewAuth() API key = %q, expected %q", auth.config.APIKey, tt.config.APIKey)
			}
		})
	}
}

func TestAuth_Middleware_NoAuth(t *testing.T) {
	auth := NewAuth(AuthConfig{Method: AuthMethodNone})

	called := false
	handler := auth.Middleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if !called {
		t.Error("Middleware() did not call handler with no auth")
	}

	if rr.Code != http.StatusOK {
		t.Errorf("Middleware() status = %d, expected %d", rr.Code, http.StatusOK)
	}
}

func TestAuth_Middleware_APIKey_Valid(t *testing.T) {
	apiKey := "test-api-key"
	auth := NewAuth(AuthConfig{
		Method: AuthMethodAPIKey,
		APIKey: apiKey,
	})

	called := false
	handler := auth.Middleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name        string
		setupReq    func(*http.Request)
		expectAuth  bool
	}{
		{
			name: "Bearer token in Authorization header",
			setupReq: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+apiKey)
			},
			expectAuth: true,
		},
		{
			name: "X-API-Key header",
			setupReq: func(r *http.Request) {
				r.Header.Set("X-API-Key", apiKey)
			},
			expectAuth: true,
		},
		{
			name: "api_key query parameter",
			setupReq: func(r *http.Request) {
				q := r.URL.Query()
				q.Set("api_key", apiKey)
				r.URL.RawQuery = q.Encode()
			},
			expectAuth: true,
		},
		{
			name: "no credentials",
			setupReq: func(r *http.Request) {
				// No auth
			},
			expectAuth: false,
		},
		{
			name: "wrong api key",
			setupReq: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer wrong-key")
			},
			expectAuth: false,
		},
		{
			name: "malformed Authorization header",
			setupReq: func(r *http.Request) {
				r.Header.Set("Authorization", "Basic "+apiKey)
			},
			expectAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called = false
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setupReq(req)
			rr := httptest.NewRecorder()

			handler(rr, req)

			if tt.expectAuth {
				if !called {
					t.Error("Middleware() should have called handler")
				}
				if rr.Code != http.StatusOK {
					t.Errorf("Middleware() status = %d, expected %d", rr.Code, http.StatusOK)
				}
			} else {
				if called {
					t.Error("Middleware() should not have called handler")
				}
				if rr.Code != http.StatusUnauthorized {
					t.Errorf("Middleware() status = %d, expected %d", rr.Code, http.StatusUnauthorized)
				}
			}
		})
	}
}

func TestAuth_GenerateToken(t *testing.T) {
	auth := NewAuth(AuthConfig{Method: AuthMethodJWT})

	ttl := 1 * time.Hour
	metadata := map[string]string{"user": "test"}

	token := auth.GenerateToken(ttl, metadata)

	if token == nil {
		t.Fatal("GenerateToken() returned nil")
	}

	if token.Token == "" {
		t.Error("GenerateToken() returned empty token")
	}

	if token.Metadata["user"] != "test" {
		t.Errorf("GenerateToken() metadata = %v, expected user=test", token.Metadata)
	}

	if token.ExpiresAt.Before(time.Now()) {
		t.Error("GenerateToken() returned already expired token")
	}

	if token.ExpiresAt.After(time.Now().Add(ttl + time.Minute)) {
		t.Error("GenerateToken() expiry too far in future")
	}
}

func TestAuth_RevokeToken(t *testing.T) {
	auth := NewAuth(AuthConfig{Method: AuthMethodJWT})

	token := auth.GenerateToken(1*time.Hour, nil)

	// Verify token exists
	auth.mu.RLock()
	_, exists := auth.tokens[token.Token]
	auth.mu.RUnlock()

	if !exists {
		t.Fatal("Token should exist after generation")
	}

	// Revoke token
	auth.RevokeToken(token.Token)

	// Verify token is removed
	auth.mu.RLock()
	_, exists = auth.tokens[token.Token]
	auth.mu.RUnlock()

	if exists {
		t.Error("Token should not exist after revocation")
	}
}

func TestAuth_GetAPIKey(t *testing.T) {
	apiKey := "my-secret-key"
	auth := NewAuth(AuthConfig{
		Method: AuthMethodAPIKey,
		APIKey: apiKey,
	})

	if auth.GetAPIKey() != apiKey {
		t.Errorf("GetAPIKey() = %q, expected %q", auth.GetAPIKey(), apiKey)
	}
}

func TestAuth_CleanupExpiredTokens(t *testing.T) {
	auth := NewAuth(AuthConfig{Method: AuthMethodJWT})

	// Generate an already-expired token
	auth.mu.Lock()
	expiredToken := "expired-token"
	auth.tokens[expiredToken] = &TokenInfo{
		Token:     expiredToken,
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	// Generate a valid token
	validToken := "valid-token"
	auth.tokens[validToken] = &TokenInfo{
		Token:     validToken,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	auth.mu.Unlock()

	// Run cleanup
	auth.CleanupExpiredTokens()

	// Check expired token is removed
	auth.mu.RLock()
	_, expiredExists := auth.tokens[expiredToken]
	_, validExists := auth.tokens[validToken]
	auth.mu.RUnlock()

	if expiredExists {
		t.Error("CleanupExpiredTokens() did not remove expired token")
	}

	if !validExists {
		t.Error("CleanupExpiredTokens() removed valid token")
	}
}

func TestAuth_JWT_ValidToken(t *testing.T) {
	auth := NewAuth(AuthConfig{Method: AuthMethodJWT})

	// Generate a session token
	tokenInfo := auth.GenerateToken(1*time.Hour, nil)

	called := false
	handler := auth.Middleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenInfo.Token)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if !called {
		t.Error("Middleware() should have called handler with valid JWT token")
	}

	if rr.Code != http.StatusOK {
		t.Errorf("Middleware() status = %d, expected %d", rr.Code, http.StatusOK)
	}
}

func TestAuth_JWT_ExpiredToken(t *testing.T) {
	auth := NewAuth(AuthConfig{Method: AuthMethodJWT})

	// Manually add an expired token
	expiredToken := "expired-jwt"
	auth.mu.Lock()
	auth.tokens[expiredToken] = &TokenInfo{
		Token:     expiredToken,
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	auth.mu.Unlock()

	called := false
	handler := auth.Middleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if called {
		t.Error("Middleware() should not have called handler with expired JWT token")
	}

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Middleware() status = %d, expected %d", rr.Code, http.StatusUnauthorized)
	}

	// Verify expired token was cleaned up
	auth.mu.RLock()
	_, exists := auth.tokens[expiredToken]
	auth.mu.RUnlock()

	if exists {
		t.Error("Expired token should have been removed after failed auth")
	}
}

func TestGenerateAPIKey(t *testing.T) {
	key1 := generateAPIKey()
	key2 := generateAPIKey()

	if key1 == "" {
		t.Error("generateAPIKey() returned empty string")
	}

	if len(key1) != 64 {
		t.Errorf("generateAPIKey() length = %d, expected 64", len(key1))
	}

	if key1 == key2 {
		t.Error("generateAPIKey() returned same key twice")
	}
}
