package main

import (
    "context"
    "crypto/rand"
    "database/sql"
    "encoding/json"
    "html/template"
    "net/http"
    "time"

    "golang.org/x/crypto/bcrypt"
)

var (
    db *sql.DB
)

// InitAuth opens the SQLite DB at path and ensures schema and default user exist.
func InitAuth(dbPath string) error {
    var err error
    db, err = OpenDB(dbPath)
    if err != nil {
        return err
    }
    if err := InitDB(db); err != nil {
        return err
    }
    // ensure default admin user exists
    if _, err := GetUserHash(db, "admin"); err == sql.ErrNoRows {
        _ = CreateUser(db, "admin", HashPassword("password"))
    }
    return nil
}

// HashPassword uses bcrypt to hash a plaintext password.
func HashPassword(password string) string {
    b, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(b)
}

// ComparePassword verifies a bcrypt hashed password.
func ComparePassword(stored, password string) bool {
    if stored == "" {
        return false
    }
    err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(password))
    return err == nil
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
    stored, err := GetUserHash(db, u)
    if err != nil {
        http.Error(w, "invalid credentials", http.StatusUnauthorized)
        return
    }
    if !ComparePassword(stored, p) {
        http.Error(w, "invalid credentials", http.StatusUnauthorized)
        return
    }
    tok := newToken()
    // persist session in Redis
    _ = RedisCreateSession(tok, u, time.Now().Add(24*time.Hour))

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

// APIregister handles user registration (development/demo use).
func APIregister(w http.ResponseWriter, r *http.Request) {
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
    if u == "" || p == "" {
        http.Error(w, "missing fields", http.StatusBadRequest)
        return
    }
    exists, err := UserExists(db, u)
    if err != nil {
        http.Error(w, "server error", http.StatusInternalServerError)
        return
    }
    if exists {
        http.Error(w, "user exists", http.StatusConflict)
        return
    }
    if err := CreateUser(db, u, HashPassword(p)); err != nil {
        http.Error(w, "create error", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{"status": "created", "user": u})
}

// APIchangePassword allows an authenticated user to change their password.
func APIchangePassword(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    user := GetUser(r)
    if user == "" {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    if err := r.ParseForm(); err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }
    old := r.FormValue("old_password")
    nw := r.FormValue("new_password")
    if old == "" || nw == "" {
        http.Error(w, "missing fields", http.StatusBadRequest)
        return
    }
    stored, err := GetUserHash(db, user)
    if err != nil {
        http.Error(w, "server error", http.StatusInternalServerError)
        return
    }
    if !ComparePassword(stored, old) {
        http.Error(w, "invalid current password", http.StatusUnauthorized)
        return
    }
    if err := UpdateUserPassword(db, user, HashPassword(nw)); err != nil {
        http.Error(w, "update error", http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func Logout(w http.ResponseWriter, r *http.Request) {
    c, err := r.Cookie("session")
    if err == nil {
        _ = DeleteSession(db, c.Value)
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
    u, ok, err := GetSessionUser(db, c.Value)
    if err != nil {
        return "", false
    }
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
