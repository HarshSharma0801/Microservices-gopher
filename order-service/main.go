package main

import (
    "bytes"
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "time"

    _ "github.com/mattn/go-sqlite3"
)

type Order struct {
    ID            int    `json:"id"`
    UserID        int    `json:"user_id"`
    Amount        int    `json:"amount"`
    Description   string `json:"description,omitempty"`
    PaymentStatus string `json:"payment_status,omitempty"`
}

type PaymentResponse struct {
    OrderID       int    `json:"orderId"`
    Amount        int    `json:"amount"`
    Status        string `json:"status"`
    TransactionID string `json:"transactionId,omitempty"`
    ErrorMessage  string `json:"errorMessage,omitempty"`
}

type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    dbPath := os.Getenv("DB_PATH")
    if dbPath == "" {
        dbPath = "./data/orders.db"
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

    http.HandleFunc("/api/orders", orderHandler(db))

    log.Println("Order Service running on :3002")
    log.Fatal(http.ListenAndServe(":3002", nil))
}

func initializeDatabase(db *sql.DB) error {
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS orders (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL,
            amount INTEGER NOT NULL,
            description TEXT,
            payment_status TEXT DEFAULT 'pending',
            FOREIGN KEY (user_id) REFERENCES users(id)
        )
    `)
    return err
}

func orderHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")

        if r.Method != http.MethodPost {
            http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
            return
        }

        var order Order
        if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
            http.Error(w, `{"error": "Invalid request"}`, http.StatusBadRequest)
            return
        }

        if order.UserID <= 0 || order.Amount <= 0 {
            http.Error(w, `{"error": "Invalid user_id or amount"}`, http.StatusBadRequest)
            return
        }

        user, err := getUser(order.UserID)
        if err != nil {
            log.Printf("User validation failed: %v", err)
            http.Error(w, fmt.Sprintf(`{"error": "User validation failed: %v"}`, err), http.StatusBadRequest)
            return
        }

        result, err := db.Exec(
            "INSERT INTO orders (user_id, amount, description) VALUES (?, ?, ?)",
            order.UserID, order.Amount, order.Description,
        )
        if err != nil {
            log.Printf("Insert error: %v", err)
            http.Error(w, `{"error": "Database error"}`, http.StatusInternalServerError)
            return
        }

        id, err := result.LastInsertId()
        if err != nil {
            log.Printf("LastInsertId error: %v", err)
            http.Error(w, `{"error": "Failed to get order ID"}`, http.StatusInternalServerError)
            return
        }
        order.ID = int(id)

        paymentResp, err := processPayment(order)
        if err != nil {
            log.Printf("Payment failed for order %d: %v", order.ID, err)
            updatePaymentStatus(db, order.ID, "failed")
            json.NewEncoder(w).Encode(map[string]interface{}{
                "order":        order,
                "paymentError": err.Error(),
            })
            return
        }

        updatePaymentStatus(db, order.ID, paymentResp.Status)

        go func(order Order, status string, email string) {
            err := sendNotification(order.ID, status, order.Amount, email)
            if err != nil {
                log.Printf("Notification failed for order %d: %v", order.ID, err)
            }
        }(order, paymentResp.Status, user.Email)

        json.NewEncoder(w).Encode(map[string]interface{}{
            "order":   order,
            "payment": paymentResp,
        })
    }
}

func getUser(userID int) (User, error) {
    client := &http.Client{Timeout: 5 * time.Second}
    url := fmt.Sprintf("http://user-service:3001/api/users/%d", userID)
    log.Printf("Requesting user at: %s", url) // Added for debugging
    resp, err := client.Get(url)
    if err != nil {
        log.Printf("Failed to connect to user service: %v", err)
        return User{}, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        log.Printf("User service returned non-OK status: %d", resp.StatusCode)
        return User{}, fmt.Errorf("user not found or error: %d", resp.StatusCode)
    }

    var user User
    if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
        log.Printf("Failed to decode user response: %v", err)
        return User{}, err
    }
    log.Printf("Successfully retrieved user: %+v", user)
    return user, nil
}

func processPayment(order Order) (PaymentResponse, error) {
    payload, err := json.Marshal(struct {
        OrderID     int    `json:"orderId"`
        Amount      int    `json:"amount"`
        Description string `json:"description"`
    }{
        OrderID:     order.ID,
        Amount:      order.Amount,
        Description: order.Description,
    })
    if err != nil {
        return PaymentResponse{}, err
    }

    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Post("http://payment-service:3003/api/payments", "application/json", bytes.NewBuffer(payload))
    if err != nil {
        return PaymentResponse{}, err
    }
    defer resp.Body.Close()

    var paymentResp PaymentResponse
    if err := json.NewDecoder(resp.Body).Decode(&paymentResp); err != nil {
        return PaymentResponse{}, err
    }

    if resp.StatusCode != http.StatusOK {
        return paymentResp, fmt.Errorf("payment failed: %s", paymentResp.ErrorMessage)
    }
    return paymentResp, nil
}

func sendNotification(orderID int, paymentStatus string, amount int, userEmail string) error {
    payload, err := json.Marshal(struct {
        OrderID       int    `json:"orderId"`
        PaymentStatus string `json:"paymentStatus"`
        Amount        int    `json:"amount"`
        UserEmail     string `json:"userEmail"`
    }{
        OrderID:       orderID,
        PaymentStatus: paymentStatus,
        Amount:        amount,
        UserEmail:     userEmail,
    })
    if err != nil {
        return err
    }

    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Post("http://notification-service:3004/api/notifications", "application/json", bytes.NewBuffer(payload)) 
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("notification failed with status: %d", resp.StatusCode)
    }
    return nil
}

func updatePaymentStatus(db *sql.DB, orderID int, status string) {
    _, err := db.Exec("UPDATE orders SET payment_status = ? WHERE id = ?", status, orderID)
    if err != nil {
        log.Printf("Failed to update payment status for order %d: %v", orderID, err)
    }
}