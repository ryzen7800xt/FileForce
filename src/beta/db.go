package main

import (
    "database/sql"
    "time"

    _ "github.com/mattn/go-sqlite3"
)

func OpenDB(path string) (*sql.DB, error) {
    db, err := sql.Open("sqlite3", path)
    if err != nil {
        return nil, err
    }
    // sqlite: keep connections low
    db.SetMaxOpenConns(1)
    return db, nil
}

func InitDB(db *sql.DB) error {
    stmts := []string{
        `CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY,
            username TEXT UNIQUE,
            password_hash TEXT,
            created_at DATETIME
        );`,
        `CREATE TABLE IF NOT EXISTS sessions (
            token TEXT PRIMARY KEY,
            username TEXT,
            expires_at DATETIME
        );`,
    }
    for _, s := range stmts {
        if _, err := db.Exec(s); err != nil {
            return err
        }
    }
    return nil
}

func CreateUser(db *sql.DB, username, passwordHash string) error {
    _, err := db.Exec("INSERT INTO users(username, password_hash, created_at) VALUES(?,?,?)", username, passwordHash, time.Now().UTC())
    return err
}

func GetUserHash(db *sql.DB, username string) (string, error) {
    var h string
    err := db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&h)
    return h, err
}

func UpdateUserPassword(db *sql.DB, username, passwordHash string) error {
    _, err := db.Exec("UPDATE users SET password_hash = ? WHERE username = ?", passwordHash, username)
    return err
}

func UserExists(db *sql.DB, username string) (bool, error) {
    var count int
    err := db.QueryRow("SELECT COUNT(1) FROM users WHERE username = ?", username).Scan(&count)
    if err != nil {
        return false, err
    }
    return count > 0, nil
}

func CreateSession(db *sql.DB, token, username string, expires time.Time) error {
    _, err := db.Exec("INSERT OR REPLACE INTO sessions(token, username, expires_at) VALUES(?,?,?)", token, username, expires.UTC())
    return err
}

func GetSessionUser(db *sql.DB, token string) (string, bool, error) {
    var username string
    var expiresStr string
    err := db.QueryRow("SELECT username, expires_at FROM sessions WHERE token = ?", token).Scan(&username, &expiresStr)
    if err != nil {
        if err == sql.ErrNoRows {
            return "", false, nil
        }
        return "", false, err
    }
    // parse time
    expires, err := time.Parse(time.RFC3339Nano, expiresStr)
    if err != nil {
        // fallback: if DB stored as sqlite datetime we can try parse
        t, perr := time.Parse("2006-01-02 15:04:05", expiresStr)
        if perr == nil {
            expires = t
        } else {
            return "", false, perr
        }
    }
    if time.Now().After(expires) {
        // expired: remove session
        _ = DeleteSession(db, token)
        return "", false, nil
    }
    return username, true, nil
}

func DeleteSession(db *sql.DB, token string) error {
    _, err := db.Exec("DELETE FROM sessions WHERE token = ?", token)
    return err
}
