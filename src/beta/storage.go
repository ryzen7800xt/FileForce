package main

import (
    "io"
    "io/ioutil"
    "os"
    "path/filepath"
)

func SaveFile(name string, r io.Reader) error {
    fpath := filepath.Join(dataDir, filepath.Base(name))
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

func ListFiles() ([]string, error) {
    entries, err := ioutil.ReadDir(dataDir)
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

func FilePath(name string) string {
    return filepath.Join(dataDir, filepath.Base(name))
}

func DeleteFile(name string) error {
    return os.Remove(FilePath(name))
}
