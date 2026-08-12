package main

import (
    "io"
    "io/ioutil"
    "os"
    "path/filepath"
)

// SaveFile saves into a per-user directory when username is provided.
func SaveFile(name string, r io.Reader, username string) error {
    dir := dataDir
    if username != "" {
        dir = filepath.Join(dataDir, "users", username)
    }
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }
    fpath := filepath.Join(dir, filepath.Base(name))
    tmp := fpath + ".tmp"
    out, err := os.Create(tmp)
    if err != nil {
        return err
    }
    defer out.Close()
    if _, err := io.Copy(out, r); err != nil {
        os.Remove(tmp)
        return err
    }
    return os.Rename(tmp, fpath)
}

// ListFiles lists files for a user when username is provided. If username is empty, lists all files across users and top-level data dir.
func ListFiles(username string) ([]string, error) {
    if username != "" {
        dir := filepath.Join(dataDir, "users", username)
        entries, err := ioutil.ReadDir(dir)
        if err != nil {
            return nil, err
        }
        res := make([]string, 0, len(entries))
        for _, e := range entries {
            if !e.IsDir() {
                res = append(res, e.Name())
            }
        }
        return res, nil
    }
    // list top-level and per-user files
    res := []string{}
    entries, err := ioutil.ReadDir(dataDir)
    if err != nil {
        return nil, err
    }
    for _, e := range entries {
        if e.IsDir() && e.Name() == "users" {
            users, _ := ioutil.ReadDir(filepath.Join(dataDir, "users"))
            for _, u := range users {
                if u.IsDir() {
                    files, _ := ioutil.ReadDir(filepath.Join(dataDir, "users", u.Name()))
                    for _, f := range files {
                        if !f.IsDir() {
                            res = append(res, filepath.Join(u.Name(), f.Name()))
                        }
                    }
                }
            }
        } else if !e.IsDir() {
            res = append(res, e.Name())
        }
    }
    return res, nil
}

// FilePath returns the filesystem path for a given username and filename.
func FilePath(username, name string) string {
    if username != "" {
        return filepath.Join(dataDir, "users", username, filepath.Base(name))
    }
    return filepath.Join(dataDir, filepath.Base(name))
}

// DeleteFile deletes a file for a given username (empty for top-level).
func DeleteFile(username, name string) error {
    return os.Remove(FilePath(username, name))
}
