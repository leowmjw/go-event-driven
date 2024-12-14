package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"app/internal/customers"

	"github.com/bitfield/script"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const mongoURI = "mongodb://localhost:27017/?replicaSet=rs0&directConnection=true"

type CustomerRequest struct {
	Action string `json:"action"` // "create" or "update"
	Name   string `json:"name"`
	Email  string `json:"email"`
	CustomerID string `json:"customerID"`
}

type CustomerResponse struct {
	Message  string              `json:"message,omitempty"`
	Customer *customers.Customer `json:"customer"`
}

func mainToo() {

	// Create a new server mux
	server := http.NewServeMux()

	// Register routes
	server.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello World")
		return
	})

	// Create server with timeouts
	srv := &http.Server{
		Addr:         ":8080", // This prevents Mac Firewall noisy ..
		Handler:      server,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		fmt.Printf("Server starting on port %s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v\n", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	fmt.Println("\nShutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v\n", err)
	}

	fmt.Println("Server gracefully stopped")

}

func main() {
	fmt.Println("Welcome to EventDriven e-Commerce! Toasties!! Overmind!!")

	// Initialize setup handler
	setupHandler, err := customers.NewSetupHandler(mongoURI)
	if err != nil {
		log.Fatalf("Failed to create setup handler: %v", err)
	}
	defer setupHandler.Close(context.Background())

	// Initialize repository for customer operations
	repository := customers.NewMongoRepository(mongoURI)

	useBento := os.Getenv("ENABLE_BENTO")
	if useBento != "" {
		fmt.Println("ACTIVE: Bento Forwarder")
		// Use Bento for the forwarder instead ...
		// Start to call bento command line; and have a termination as a defer
		script.Exec("echo Hello, world!").Stdout()

		defer func() {
			fmt.Println("KILL: Bento Forwarder")
			script.Exec("kill `pgrep bento`").Stdout()
			fmt.Println("FINISH: Bento Forwarder")
		}()
	} else {
		fmt.Println("DEFAULT: channel Forwarder")
		// DEFAULT: Initialize forwarder for normal operations
		forwarder := customers.NewEventForwarder(mongoURI)
		forwarder.Start()
		defer forwarder.Stop()
	}

	// Create a new server mux
	server := http.NewServeMux()

	// Register routes
	server.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello World")
		return
	})
	server.HandleFunc("POST /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello POSTy")
		return
	})
	server.HandleFunc("GET /error", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	})

	// Add customers endpoints
	server.HandleFunc("GET /customers", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get all customers
		customers, err := setupHandler.GetTestCustomers(ctx)
		if err != nil {
			log.Printf("Failed to get customers: %v", err)
			http.Error(w, "Failed to get customers", http.StatusInternalServerError)
			return
		}

		// Return customers as JSON
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(customers); err != nil {
			log.Printf("Failed to encode response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	})

	// Add customer create/update endpoint
	server.HandleFunc("POST /customers/action", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Parse request
		var req CustomerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Failed to decode request: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate action
		if req.Action != "create" && req.Action != "update" && req.Action != "delete" {
			http.Error(w, "Invalid action, must be 'create', 'update', or 'delete'", http.StatusBadRequest)
			return
		}

		var customer *customers.Customer
		var message string
		var err error

		if req.Action == "create" {
			// Create new customer
			customer = &customers.Customer{
				Name:      req.Name,
				Email:     req.Email,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			err = repository.Create(customer)
			message = "Customer created successfully"
		} else if req.Action == "update" {
			// Get a random customer to update
			customer, err = repository.GetRandomCustomer(ctx)
			if err != nil {
				log.Printf("Failed to get random customer: %v", err)
				http.Error(w, "Failed to get random customer", http.StatusInternalServerError)
				return
			}

			// Update the customer with new data
			customer.Name = req.Name
			customer.Email = req.Email
			customer.UpdatedAt = time.Now()

			err = repository.Update(customer)
			message = fmt.Sprintf("Customer %s updated successfully", customer.ID.Hex())
		} else if req.Action == "delete" {
			// Get customer by ID
			customerID, err := primitive.ObjectIDFromHex(req.CustomerID)
			if err != nil {
				log.Printf("Invalid customer ID: %v", err)
				http.Error(w, "Invalid customer ID", http.StatusBadRequest)
				return
			}
			
			// Find customer first
			customer, err = repository.FindByID(customerID, true)
			if err != nil {
				log.Printf("Failed to find customer: %v", err)
				http.Error(w, "Failed to find customer", http.StatusInternalServerError)
				return
			}
			if customer == nil {
				http.Error(w, "Customer not found", http.StatusNotFound)
				return
			}

			// Soft delete the customer
			err = repository.SoftDelete(customerID)
			message = fmt.Sprintf("Customer %s deleted successfully", customerID.Hex())
		}

		if err != nil {
			log.Printf("Failed to %s customer: %v", req.Action, err)
			http.Error(w, fmt.Sprintf("Failed to %s customer", req.Action), http.StatusInternalServerError)
			return
		}

		// Return response with message and customer data
		response := CustomerResponse{
			Message:  message,
			Customer: customer,
		}

		// Return the response as JSON
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	})

	// Add outbox endpoint
	server.HandleFunc("GET /customers/outbox", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get all outbox entries
		entries, err := setupHandler.GetOutboxEntries(ctx)
		if err != nil {
			log.Printf("Failed to get outbox entries: %v", err)
			http.Error(w, "Failed to get outbox entries", http.StatusInternalServerError)
			return
		}

		// Return entries as JSON
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(entries); err != nil {
			log.Printf("Failed to encode response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	})

	// Add projection endpoint
	server.HandleFunc("GET /customers/projection", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get all customer projections
		projections, err := setupHandler.GetCustomerProjections(ctx)
		if err != nil {
			log.Printf("Failed to get customer projections: %v", err)
			http.Error(w, "Failed to get customer projections", http.StatusInternalServerError)
			return
		}

		// Return projections as JSON
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(projections); err != nil {
			log.Printf("Failed to encode response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	})

	// Move setup to /customers/setup
	server.HandleFunc("POST /customers/setup", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if err := setupHandler.ResetDatabases(ctx); err != nil {
			log.Printf("Failed to reset databases: %v", err)
			http.Error(w, "Failed to reset databases", http.StatusInternalServerError)
			return
		}

		// Get test customers for response
		testCustomers, err := setupHandler.GetTestCustomers(ctx)
		if err != nil {
			log.Printf("Failed to get test customers: %v", err)
			http.Error(w, "Failed to get test customers", http.StatusInternalServerError)
			return
		}

		// Return test customers as JSON
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(testCustomers); err != nil {
			log.Printf("Failed to encode response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	})

	// Create server with timeouts
	srv := &http.Server{
		Addr:         "localhost:8080", // This prevents Mac Firewall noisy ..
		Handler:      server,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		fmt.Printf("Server starting on port %s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v\n", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	fmt.Println("\nShutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v\n", err)
	}

	fmt.Println("Server gracefully stopped")
}
