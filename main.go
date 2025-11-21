package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var apiKey string

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	apiKey = os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatalln("API_KEY not found. Make sure it is set in your .env file")
	}
}

type Payment struct {
	Amount       float64 `json:"amount"`
	MobileNumber string  `json:"from"`
	Currency     string  `json:"currency"`
	Description  string  `json:"description"`
}

type CollectResponse struct {
	Reference string `json:"reference"`
	USSDCode  string `json:"ussd_code"`
	Operator  string `json:"operator"`
	Message   string `json:"message"`
	ErrorCode string `json:"error_code"`
}

type TransactionStatusResponse struct {
	Reference         string  `json:"reference"`
	ExternalReference string  `json:"external_reference"`
	Status            string  `json:"status"`
	Amount            float64 `json:"amount"`
	Currency          string  `json:"currency"`
	Operator          string  `json:"operator"`
	Code              string  `json:"code"`
	OperatorReference string  `json:"operator_reference"`
	Description       string  `json:"description"`
	Reason            string  `json:"reason"`
	PhoneNumber       string  `json:"phone_number"`
	Endpoint          string  `json:"endpoint"`
}

func getUserInput(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Error reading input: %v", err)
	}
	return strings.TrimSpace(input)
}

func parseAmount(input string) (float64, error) {
	value, err := strconv.ParseFloat(input, 64)
	if err != nil || value <= 0 {
		return 0, errors.New("invalid amount, must be a positive number")
	}
	return value, nil
}

func createPayment(payment Payment) (*CollectResponse, error) {
	bodyBytes, err := json.Marshal(payment)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment: %v", err)
	}

	req, err := http.NewRequest("POST", "https://demo.campay.net/api/collect/", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Token "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %v", err)
	}
	defer resp.Body.Close()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var collectResp CollectResponse
	if err := json.Unmarshal(responseData, &collectResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if collectResp.ErrorCode != "" {
		return nil, fmt.Errorf("API returned error: %s (%s)", collectResp.Message, collectResp.ErrorCode)
	}

	return &collectResp, nil
}

func pollTransaction(reference string) {
	statusURL := fmt.Sprintf("https://demo.campay.net/api/transaction/%s/", reference)

	client := &http.Client{}
	maxRetries := 12
	interval := 5 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(interval)
		}

		req, err := http.NewRequest("GET", statusURL, nil)
		if err != nil {
			log.Printf("Error creating status request: %v", err)
			continue
		}

		req.Header.Set("Authorization", "Token "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error sending status request: %v", err)
			continue
		}
		defer resp.Body.Close()

		statusData, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading status response: %v", err)
			continue
		}

		var statusResp TransactionStatusResponse
		if err := json.Unmarshal(statusData, &statusResp); err != nil {
			log.Printf("Error parsing status response: %v", err)
			continue
		}

		switch statusResp.Status {
		case "SUCCESSFUL":
			log.Printf("\n\n TRANSACTION SUCCESSFUL\nDetails: %+v\n", statusResp)
			return
		case "FAILED":
			log.Fatalf("\n\n TRANSACTION FAILED\nReason: %s\n", statusResp.Reason)
		}
	}

	log.Println("\nTRANSACTION FAILED: TIMED OUT")
}

func main() {
	mobile := getUserInput("Enter your phone number (with country code):")
	amountStr := getUserInput("Enter the amount to be debited:")
	desc := getUserInput("Enter a description:")

	amount, err := parseAmount(amountStr)
	if err != nil {
		log.Fatalf("Invalid input: %v", err)
	}

	payment := Payment{
		Amount:       amount,
		MobileNumber: mobile,
		Currency:     "XAF",
		Description:  desc,
	}

	collectResp, err := createPayment(payment)
	if err != nil {
		log.Fatalf("Payment initiation failed: %v", err)
	}

	log.Println("Payment request sent. Waiting for user confirmation...")
	pollTransaction(collectResp.Reference)
}
