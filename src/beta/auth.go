package main

import (
    "context"
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "html/template"
    "net/http"
    "sync"
    "time"
)

var (
    sessions = map[string]string{} // token -> username
    sessMu   sync.Mutex
    // NOTE: replace this in production with a proper user store + persistent DB
    users = map[string]string{}
)

func init() {
    // seed a default user for convenience; password is "password"
    users["admin"] = HashPassword("password")
}

// HashPassword produces a salted SHA-256 hash in the form salt$hash (hex-encoded)
func HashPassword(password string) string {
    salt := make([]byte, 16)
    rand.Read(salt)
    h := sha256.New()
    h.Write(salt)
    h.Write([]byte(password))
    sum := h.Sum(nil)
    return hex.EncodeToString(salt) + "$" + hex.EncodeToString(sum)
}

// ComparePassword verifies a plaintext password against a stored salted hash
func ComparePassword(stored, password string) bool {
    // expected format salt$hash
    parts := make([]string, 0, 2)
    for i := 0; i < len(stored); i++ {
        if stored[i] == '$' {
            parts = append(parts, stored[:i], stored[i+1:])
            break
        }
    }
    if len(parts) != 2 {
        return false
    }
    salt, err := hex.DecodeString(parts[0])
    if err != nil {
        return false
    }
    expected, err := hex.DecodeString(parts[1])
    if err != nil {
        return false
    }
    h := sha256.New()
    h.Write(salt)
    h.Write([]byte(password))
    sum := h.Sum(nil)
    return hmacEqual(sum, expected)
}

// hmacEqual compares two byte slices in constant time
func hmacEqual(a, b []byte) bool {
    if len(a) != len(b) {
        return false
    }
    var res byte
    for i := 0; i < len(a); i++ {
        res |= a[i] ^ b[i]
    }
    return res == 0
}

type key int

const userKey key = 0

func LoginPage(w http.ResponseWriter, r *http.Request) {
    t, _ := template.ParseFiles("templates/login.html")
    t.Execute(w, nil)
}

func APIlogin(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    if err := r.ParseForm(); err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }
    u := r.FormValue("username")
    p := r.FormValue("password")
    stored, ok := users[u]
    if !ok || !ComparePassword(stored, p) {
        http.Error(w, "invalid credentials", http.StatusUnauthorized)
        return
    }
    tok := newToken()
    sessMu.Lock()
    sessions[tok] = u
    sessMu.Unlock()

    http.SetCookie(w, &http.Cookie{
        Name:     "session",
        Value:    tok,
        Path:     "/",
        HttpOnly: true,
        Expires:  time.Now().Add(24 * time.Hour),
    })
    // respond with JSON for API clients
    json.NewEncoder(w).Encode(map[string]string{"status": "ok", "user": u})
}

func Logout(w http.ResponseWriter, r *http.Request) {
    c, err := r.Cookie("session")
    if err == nil {
        sessMu.Lock()
        delete(sessions, c.Value)
        sessMu.Unlock()
        http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
    }
    http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func newToken() string {
    b := make([]byte, 32)
    rand.Read(b)
    return hex.EncodeToString(b)
}

// fromRequest returns username and true when session is valid
func fromRequest(r *http.Request) (string, bool) {
    c, err := r.Cookie("session")
    if err != nil {
        return "", false
    }
    sessMu.Lock()
    defer sessMu.Unlock()
    u, ok := sessions[c.Value]
    return u, ok
}

// RequireAuth wraps an http.Handler and redirects to /login if not authenticated.
func RequireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if u, ok := fromRequest(r); ok {
            ctx := context.WithValue(r.Context(), userKey, u)
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }
        // If request looks like API (Accept: application/json or X-Requested-With) return 401
        if r.Header.Get("Accept") == "application/json" || r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        http.Redirect(w, r, "/login", http.StatusSeeOther)
    })
}

// GetUser returns the authenticated username from context, if present.
func GetUser(r *http.Request) string {
    if v := r.Context().Value(userKey); v != nil {
        if s, ok := v.(string); ok {
            return s
        }
    }
    return ""
}
