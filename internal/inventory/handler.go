package inventory

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateInventory handles POST /inventory requests
func (m *Module) CreateInventory(w http.ResponseWriter, r *http.Request) {
	var inv Inventory
	if err := json.NewDecoder(r.Body).Decode(&inv); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	inv.UpdatedAt = time.Now()
	if err := m.repo.SaveInventory(r.Context(), &inv); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create outbox event
	event := InventoryEvent{
		ProductID: inv.ProductID,
		Quantity:  inv.Quantity,
		Status:    inv.Status,
		UpdatedAt: inv.UpdatedAt,
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	outboxEvent := OutboxEvent{
		EventType: "inventory.created",
		Payload:   eventData,
		CreatedAt: time.Now(),
		Status:    OutboxStatusPending,
	}

	if err := m.repo.SaveOutboxEvent(r.Context(), outboxEvent); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(inv)
}

// UpdateInventory handles PUT /inventory/{id} requests
func (m *Module) UpdateInventory(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.Split(r.URL.Path, "/")
	if len(path) != 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	
	id, err := primitive.ObjectIDFromHex(path[2])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var update Inventory
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	update.ID = id
	update.UpdatedAt = time.Now()

	if err := m.repo.SaveInventory(r.Context(), &update); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create outbox event
	event := InventoryEvent{
		ProductID: update.ProductID,
		Quantity:  update.Quantity,
		Status:    update.Status,
		UpdatedAt: update.UpdatedAt,
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	outboxEvent := OutboxEvent{
		EventType: "inventory.updated",
		Payload:   eventData,
		CreatedAt: time.Now(),
		Status:    OutboxStatusPending,
	}

	if err := m.repo.SaveOutboxEvent(r.Context(), outboxEvent); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(update)
}

// GetInventory handles GET /inventory/{id} requests
func (m *Module) GetInventory(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.Split(r.URL.Path, "/")
	if len(path) != 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	
	id, err := primitive.ObjectIDFromHex(path[2])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	inv, err := m.repo.GetInventory(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(inv)
}
