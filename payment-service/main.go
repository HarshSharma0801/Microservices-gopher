package main

import (
	"encoding/json"
	"log"
	"net/http"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/paymentintent"
)

type Config struct {
	StripeSecretKey string
	Port            string
}

type PaymentRequest struct {
	OrderID     int    `json:"orderId"`
	Amount      int    `json:"amount"`
	Description string `json:"description"`
}

type PaymentResponse struct {
	OrderID       int    `json:"orderId"`
	Amount        int    `json:"amount"`
	Status        string `json:"status"`
	TransactionID string `json:"transactionId,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
}

type PaymentProcessor interface {
	ProcessPayment(req PaymentRequest) (PaymentResponse, error)
}

type StripeProcessor struct {
	secretKey string
}

func NewStripeProcessor(secretKey string) *StripeProcessor {
	stripe.Key = secretKey
	return &StripeProcessor{secretKey: secretKey}
}

func (sp *StripeProcessor) ProcessPayment(req PaymentRequest) (PaymentResponse, error) {
	params := &stripe.PaymentIntentParams{
		Amount:      stripe.Int64(int64(req.Amount)),
		Currency:    stripe.String(string(stripe.CurrencyUSD)),
		Description: stripe.String(req.Description),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return PaymentResponse{
			OrderID:      req.OrderID,
			Amount:       req.Amount,
			Status:       "failed",
			ErrorMessage: err.Error(),
		}, err
	}

	return PaymentResponse{
		OrderID:       req.OrderID,
		Amount:        req.Amount,
		Status:        "success",
		TransactionID: pi.ID,
	}, nil
}

func main() {
	config := Config{
		StripeSecretKey: "",
		Port:            ":3003",
	}
	if config.StripeSecretKey == "" {
		log.Fatal("STRIPE_SECRET_KEY is not set")
	}

	processor := NewStripeProcessor(config.StripeSecretKey)
	http.HandleFunc("/api/payments", paymentHandler(processor))

	log.Printf("Payment Service running on %s", config.Port)
	log.Fatal(http.ListenAndServe(config.Port, nil))
}

func paymentHandler(processor PaymentProcessor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		var req PaymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "Invalid request format"}`, http.StatusBadRequest)
			return
		}

		if req.Amount <= 0 {
			http.Error(w, `{"error": "Amount must be greater than 0"}`, http.StatusBadRequest)
			return
		}

		resp, err := processor.ProcessPayment(req)
		if err != nil {
			log.Printf("Payment processing error for order %d: %v", req.OrderID, err)
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
			return
		}
	}
}