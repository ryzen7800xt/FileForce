package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const dataDir = "data" 

var allowedExts = map[string]bool{
	".txt":  true,
	".pdf":  true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".mp3":  true,
	".mp4":  true,
	".zip":  true,
	".tscn": true,
	
}

func main() {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	// initialize auth DB
	dbPath := filepath.Join(dataDir, "auth.db")
	if err := InitAuth(dbPath); err != nil {
		log.Fatalf("init auth db: %v", err)
	}

	// Auth routes
	http.HandleFunc("/login", LoginPage)
	http.HandleFunc("/api/login", APIlogin)
	http.HandleFunc("/logout", Logout)
	http.HandleFunc("/api/register", APIregister)
	http.Handle("/api/change-password", RequireAuth(http.HandlerFunc(APIchangePassword)))

	// Public routes
	http.HandleFunc("/files", listHandler)
	http.HandleFunc("/files/", fileHandler)

	// Protected routes
	http.Handle("/upload", RequireAuth(http.HandlerFunc(uploadHandler)))

	addr := ":8080"
	fmt.Printf("server listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer f.Close()

	name := filepath.Base(hdr.Filename)
	user := GetUser(r)
	// user should be present because upload is protected by RequireAuth
	if user == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	// check allowed extensions
	ext := strings.ToLower(filepath.Ext(name))
	if !allowedExts[ext] {
		http.Error(w, "file type not allowed", http.StatusBadRequest)
		return
	}
	if err := SaveFile(name, f, user); err != nil {
		http.Error(w, "save error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"name": name})
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := GetUser(r)
	// support ?all=1 to list all files (admin use)
	all := r.URL.Query().Get("all") == "1"
	var files []string
	var err error
	if user != "" && !all {
		files, err = ListFiles(user)
	} else {
		files, err = ListFiles("")
	}
	if err != nil {
		http.Error(w, "list error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(files)
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/files/")
	if name == "" {
		http.Error(w, "missing filename", http.StatusBadRequest)
		return
	}
	// support URL forms: username/filename or filename (own)
	parts := strings.SplitN(name, "/", 2)
	var user, fname string
	if len(parts) == 2 {
		user = parts[0]
		fname = parts[1]
	} else {
		user = GetUser(r)
		fname = parts[0]
	}
	safe := filepath.Base(fname)
	path := FilePath(user, safe)

	switch r.Method {
	case http.MethodGet:
		// download
		f, err := os.Open(path)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", safe))
		http.ServeFile(w, r, path)
	case http.MethodDelete:
		// require auth for delete
		if _, ok := fromRequest(r); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if err := DeleteFile(safe); err != nil {
			http.Error(w, "delete error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
