# Mobile Money Payment Client (Go)

This is a simple command-line application written in Go (Golang) that simulates a payment process with a the CAMPAY API. It handles transaction initiation, checks for immediate API errors, and uses Polling to continuously track the status of the transaction until it is successful, failed, or timed out.

## Prerequisites

To run this application, you need:

- Go (version 1.18 or higher) installed on your system.
- A terminal/command line interface.
- The necessary Go dependencies.

## Setup

### 1. Initialize Project

First, initialize your Go module and download the required dependency for handling environment variables:

Bash

go mod init your_project_name
go get github.com/joho/godotenv

### 2. Configure API Key

The application requires an API token to communicate with the payment gateway. This must be stored in a file named .env in the same directory as your main.go file.

Create a file named .env and add your API key:
Plaintext

API_KEY="your_api_token_here"

## How to Run the Client

- Save the Code: Ensure your Go code is saved as main.go.

- Run the application:

Bash

go run main.go

The application will prompt you to enter the phone number, amount, and transaction description.

## Key Concepts

### Polling

- When you initiate a mobile money transaction, the result is not instantaneous—the user has to enter their PIN on their phone.

- The client starts a transaction and receives a referenceID.

- Instead of stopping, the client enters a polling loop where it repeatedly asks the payment server, "What is the status of this referenceID?"

- It waits 5 seconds between each check (pollInterval).

- It checks a maximum of 12 times (maxRetries), giving the transaction 60 seconds to complete.

### Transaction Outcomes

The program handles three outcomes:

####  Immediate API Error

Occurs before polling starts.

The program stops immediately and displays the error (e.g., invalid phone number).

#### SUCCESSFUL

Polling stops immediately.

Final transaction details are printed.

#### FAILED

The program stops immediately.

Prints the failure reason.

#### TIMEOUT

Happens if no success or failure response is received within 60 seconds.

Prints a timeout message and advises checking the transaction manually using referenceID.
