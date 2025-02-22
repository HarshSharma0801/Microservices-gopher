package main

import (
    "database/sql"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "strconv"

    _ "github.com/mattn/go-sqlite3"
)

type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    dbPath := os.Getenv("DB_PATH")
    if dbPath == "" {
        dbPath = "./data/users.db"
    }

    if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
        log.Fatalf("Failed to create directory: %v", err)
    }

    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        log.Fatalf("Failed to open database: %v", err)
    }
    defer db.Close()

    if err := initializeDatabase(db); err != nil {
        log.Fatalf("Failed to initialize database: %v", err)
    }

    if err := db.Ping(); err != nil {
        log.Fatalf("Database connection failed: %v", err)
    }

    http.HandleFunc("/api/users", userHandler(db))
    http.HandleFunc("/api/users/", getUserHandler(db))

    log.Println("User Service running on :3001")
    log.Fatal(http.ListenAndServe(":3001", nil))
}

func initializeDatabase(db *sql.DB) error {
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            email TEXT NOT NULL
        )
    `)
    return err
}

func userHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        if r.Method != http.MethodPost {
            http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
            return
        }

        var user User
        if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
            http.Error(w, `{"error": "Invalid request"}`, http.StatusBadRequest)
            return
        }

        result, err := db.Exec("INSERT INTO users (name, email) VALUES (?, ?)", user.Name, user.Email)
        if err != nil {
            log.Printf("Insert error: %v", err) 
            http.Error(w, `{"error": "Database error"}`, http.StatusInternalServerError)
            return
        }

        id, err := result.LastInsertId()
        if err != nil {
            log.Printf("LastInsertId error: %v", err)
            http.Error(w, `{"error": "Failed to get user ID"}`, http.StatusInternalServerError)
            return
        }

        user.ID = int(id)
        if err := json.NewEncoder(w).Encode(user); err != nil {
            log.Printf("Response encoding error: %v", err)
        }
    }
}

func getUserHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        if r.Method != http.MethodGet {
            http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
            return
        }

        idStr := r.URL.Path[len("/api/users/"):]
        id, err := strconv.Atoi(idStr)
        if err != nil {
            http.Error(w, `{"error": "Invalid user ID"}`, http.StatusBadRequest)
            return
        }

        var user User
        err = db.QueryRow("SELECT id, name, email FROM users WHERE id = ?", id).Scan(&user.ID, &user.Name, &user.Email)
        if err == sql.ErrNoRows {
            http.Error(w, `{"error": "User not found"}`, http.StatusNotFound)
            return
        } else if err != nil {
            log.Printf("Query error: %v", err)
            http.Error(w, `{"error": "Database error"}`, http.StatusInternalServerError)
            return
        }

        if err := json.NewEncoder(w).Encode(user); err != nil {
            log.Printf("Response encoding error: %v", err)
        }
    }
}