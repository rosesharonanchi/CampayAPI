package main

import (
	"bufio"
	"bytes"
	"encoding/json"
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
	err := godotenv.Load() //load the .env file

	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	//Retrieve the api key from the .env file
	apiKey = os.Getenv("API_KEY")

	if apiKey == "" {
		log.Fatalln("API_KEY  not found make sure it is set in your .env file")
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
	Status            string  `json:"status"` // PENDING, SUCCESSFUL, or FAILED
	Amount            float64 `json:"amount"`
	Currency          string  `json:"currency"`
	Operator          string  `json:"operator"`
	Code              string  `json:"code"`
	OperatorReference string  `json:"operator_reference"`
	Description       string  `json:"description"`
	Reason            string  `json:"reason"` // Will contain the failure reason if status is FAILED
	PhoneNumber       string  `json:"phone_number"`
	Endpoint          string  `json:"endpoint"`
}

func getUserInput(prompt string) (message string) {
	//Getting input from the user
	reader := bufio.NewReader(os.Stdin) // Create a new reader
	fmt.Println(prompt)
	input, err := reader.ReadString('\n')

	if err != nil {
		log.Fatalf("Error reading input", err)
	}
	message = strings.TrimSpace(input)
	return message
}

func main() {

	// calling the getUserInput function
	mobileNumber := getUserInput("Enter your phone number (prefix with country code)")
	amount := getUserInput("Enter the amount to be debited")
	i, _ := strconv.ParseFloat(amount, 32)
	desc := getUserInput("Enter a description:")

	payments := Payment{
		Amount:       i,
		MobileNumber: mobileNumber,
		Currency:     "XAF",
		Description:  desc,
	}

	postBody, _ := json.Marshal(payments) // for http transmission,
	//Step 2: convert to a buffer
	responseBody := bytes.NewBuffer(postBody)

	// step 3: Post the response
	req, err := http.NewRequest("POST", "https://demo.campay.net/api/collect/", responseBody)

	if err != nil {
		log.Fatalf("The error is %v", err)
	}
	req.Header.Set("Authorization", "Token "+apiKey)
	req.Header.Set("Content-Type", "application/json") //setting the authorization header

	client := &http.Client{}  // Creating a new http client instance
	resp, _ := client.Do(req) //Execute the http request

	// step 4  Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	sb := string(body)
	log.Println(sb)

	//Show Payment status

	//Get the reference Id

	var postResponse CollectResponse

	var referenceID string

	err = json.Unmarshal(body, &postResponse)

	if postResponse.ErrorCode != "" { //This part is very important, ensures if any error with user inputs transaction fails and doest enter the polling loop
		log.Fatalf("REQUEST FAILED at initiation: %s(Code: %s)", postResponse.Message, postResponse.ErrorCode)
	}

	if err != nil {
		log.Fatalf("Error unmashling JSON %v\n", err)
		return
	}

	referenceID = postResponse.Reference

	statusURL := fmt.Sprintf("https://demo.campay.net/api/transaction/%s/", referenceID)

	//Polling
	//Note: The transaction "status" changes over "time"
	//We need to ensure that pending transactions after (a said amount of time in s) if not confirmed it is set to failed otherwise succesful.
	maxRetries := 12
	pollInterval := 5 * time.Second // time from pending to successful or failure must be atleast 5 sec

	for j := 0; j < maxRetries; j++ {
		if j > 0 {
			time.Sleep(pollInterval)
		}

		// Creating the actual request
		req, err = http.NewRequest("GET", statusURL, nil)

		if err != nil {
			log.Printf("Error creating status request %v", err)
		}
		req.Header.Set("Authorization", "Token "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		statusResp, _ := client.Do(req)

		// Reading the response
		defer statusResp.Body.Close()
		statusBody, _ := io.ReadAll(statusResp.Body)

		var statusResponse TransactionStatusResponse

		err = json.Unmarshal(statusBody, &statusResponse)
		if err != nil {
			log.Printf("Warning: Failed to unmarsh status response, retrying. %v", err)
		}

		currentStatus := statusResponse.Status
		//log.Printf("Transaction status: %s", currentStatus)
		if currentStatus == "SUCCESSFUL" {
			log.Printf("\n\n TRANSACTION COMPLETED: SUCCESSFUL \n Details: %v", statusResponse)
			return
		} else if currentStatus == "FAILED" {
			log.Fatalf("\n\n TRANSACTION FAILED \n Details: %v", statusResponse.Reason)
		}

	}
	log.Print("	TRANSACTION FAILED, Timed Out")

}
