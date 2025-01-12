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

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoURI = "mongodb://localhost:27017"
)

type App struct {
	mux          *http.ServeMux
	repository   *customers.MongoRepository
	setupHandler *customers.SetupHandler
	service      *customers.Service
	forwarder    customers.EventForwarder
}

func NewApp() (*App, error) {
	// Initialize MongoDB client
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Initialize repository
	repository := customers.NewMongoRepository(client.Database("CustomersDB"))

	// Initialize setup handler
	setupHandler, err := customers.NewSetupHandler(mongoURI)
	if err != nil {
		return nil, fmt.Errorf("failed to create setup handler: %v", err)
	}

	// Initialize forwarder
	forwarder := customers.NewEventForwarder(mongoURI)
	forwarder.Start()

	// Initialize service
	service := customers.NewService(repository, forwarder)

	// Initialize router
	mux := http.NewServeMux()

	app := &App{
		mux:          mux,
		repository:   repository,
		setupHandler: setupHandler,
		service:      service,
		forwarder:    forwarder,
	}

	// Setup routes
	app.setupRoutes()

	return app, nil
}

func (a *App) setupRoutes() {
	// Customer routes
	a.mux.HandleFunc("POST /customers", a.createCustomer)
	a.mux.HandleFunc("PUT /customers/{id}", a.updateCustomer)
	a.mux.HandleFunc("GET /customers/{id}", a.getCustomer)
	a.mux.HandleFunc("DELETE /customers/{id}", a.deleteCustomer)

	// Setup routes
	a.mux.HandleFunc("POST /setup/testdata", a.setupTestData)
	a.mux.HandleFunc("POST /setup/reset", a.resetData)
}

func (a *App) Cleanup() {
	a.setupHandler.Close()
	a.forwarder.Stop()
}

func (a *App) Run(addr string) error {
	server := &http.Server{
		Addr:         addr,
		Handler:      a.mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	// Shutdown server gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %v", err)
	}

	return nil
}

func (a *App) setupTestData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := a.setupHandler.SetupTestData(ctx); err != nil {
		http.Error(w, fmt.Sprintf("Failed to setup test data: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Test data setup completed successfully")
}

func (a *App) resetData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := a.setupHandler.SetupTestData(ctx); err != nil {
		http.Error(w, fmt.Sprintf("Failed to reset data: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Data reset completed successfully")
}

func (a *App) createCustomer(w http.ResponseWriter, r *http.Request) {
	var customer customers.Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if err := a.service.CreateCustomer(r.Context(), &customer); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create customer: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(customer)
}

func (a *App) updateCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid customer ID", http.StatusBadRequest)
		return
	}

	var customer customers.Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	customer.ID = id
	if err := a.service.UpdateCustomer(r.Context(), &customer); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update customer: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(customer)
}

func (a *App) getCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid customer ID", http.StatusBadRequest)
		return
	}

	customer, err := a.service.FindCustomerByID(r.Context(), id, false)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Customer not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get customer: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(customer)
}

func (a *App) deleteCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid customer ID", http.StatusBadRequest)
		return
	}

	if err := a.service.SoftDeleteCustomer(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete customer: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}
	defer app.Cleanup()

	log.Println("Starting server on :8080...")
	if err := app.Run(":8080"); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
