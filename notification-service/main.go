package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	// "os"
)

type NotificationRequest struct {
	OrderID       int    `json:"orderId"`
	PaymentStatus string `json:"paymentStatus"`
	Amount        int    `json:"amount"`
	UserEmail     string `json:"userEmail"`
}

type NotificationResponse struct {
	OrderID int    `json:"orderId"`
	Status  string `json:"status"`
}

func sendEmail(to, subject, body string) error {
	host := "mailhog"
	port := "1025"
	from := "from@example.com"

	if host == "" || port == "" {
		host = "localhost"
		port = "1025"
	}

	msg := []byte("To: "+to+"\r\n"+
		"Subject: "+subject+"\r\n"+
		"\r\n"+
		body+"\r\n")
	addr := fmt.Sprintf("%s:%s", host, port)
	return smtp.SendMail(addr, nil, from, []string{to}, msg)
}

func main() {
	http.HandleFunc("/api/notifications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		var req NotificationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "Invalid request"}`, http.StatusBadRequest)
			return
		}

		resp := NotificationResponse{OrderID: req.OrderID, Status: "sent"}

		if req.PaymentStatus == "success" {
			amountInDollars := float64(req.Amount)
			subject := fmt.Sprintf("Payment Confirmation for Order #%d", req.OrderID)
			body := fmt.Sprintf("Dear User,\n\nYour payment of $%.2f for Order #%d has been successfully processed.\n\nThank you!", amountInDollars, req.OrderID)
			if err := sendEmail(req.UserEmail, subject, body); err != nil {
				log.Printf("Failed to send email for order %d: %v", req.OrderID, err)
				resp.Status = "email_failed"
			}
		}

		json.NewEncoder(w).Encode(resp)
	})

	log.Println("Notification Service running on :3004")
	log.Fatal(http.ListenAndServe(":3004", nil))
}