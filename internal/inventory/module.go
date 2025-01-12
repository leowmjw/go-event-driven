package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nats-io/nats.go"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RepositoryInterface interface {
	SaveInventory(ctx context.Context, inv *Inventory) error
	GetInventory(ctx context.Context, id primitive.ObjectID) (*Inventory, error)
	SaveOutboxEvent(ctx context.Context, event OutboxEvent) error
	GetPendingOutboxEvents(ctx context.Context) ([]OutboxEvent, error)
	UpdateOutboxEvent(ctx context.Context, event OutboxEvent) error
	UpsertProjection(ctx context.Context, proj *InventoryProjection) error
}

type Module struct {
	repo      RepositoryInterface
	natsConn  *nats.Conn
	publisher Publisher
}

type Publisher interface {
	Publish(subject string, data []byte) error
}

type HTTPHandler struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

type MsgHandler struct {
	Subject string
	Handler nats.MsgHandler
}

func NewModule(natsConn *nats.Conn) *Module {
	return &Module{
		natsConn: natsConn,
	}
}

func (m *Module) Name() string {
	return "inventory"
}

func (m *Module) Init(config map[string]any) error {
	// Initialize MongoDB repository
	if db, ok := config["db"].(*mongo.Database); ok {
		m.repo = NewRepository(db)
		return nil
	}
	return fmt.Errorf("invalid db configuration")
}

func (m *Module) HTTPHandlers(pub Publisher) []HTTPHandler {
	m.publisher = pub
	return []HTTPHandler{
		{
			Method:  http.MethodPost,
			Path:    "/inventory",
			Handler: m.CreateInventory,
		},
		{
			Method:  http.MethodPut,
			Path:    "/inventory/{id}",
			Handler: m.UpdateInventory,
		},
		{
			Method:  http.MethodGet,
			Path:    "/inventory/{id}",
			Handler: m.GetInventory,
		},
	}
}

func (m *Module) MsgHandlers(pub Publisher) []MsgHandler {
	m.publisher = pub
	return []MsgHandler{
		{
			Subject: "inventory.outbox",
			Handler: m.ProcessOutboxEvents,
		},
		{
			Subject: "inventory.projection.update",
			Handler: m.HandleInventoryProjection,
		},
	}
}

// ServeHTTP implements http.Handler interface for routing
func (m *Module) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		if r.URL.Path == "/inventory" {
			m.CreateInventory(w, r)
			return
		}
	case http.MethodPut:
		if len(r.URL.Path) > 10 && r.URL.Path[:10] == "/inventory/" {
			m.UpdateInventory(w, r)
			return
		}
	case http.MethodGet:
		if len(r.URL.Path) > 10 && r.URL.Path[:10] == "/inventory/" {
			m.GetInventory(w, r)
			return
		}
	}
	http.NotFound(w, r)
}

// ProcessOutboxEvents processes pending events from the outbox
func (m *Module) ProcessOutboxEvents(msg *nats.Msg) {
	ctx := context.Background()
	events, err := m.repo.GetPendingOutboxEvents(ctx)
	if err != nil {
		// Log error
		return
	}

	for _, event := range events {
		// Publish to appropriate NATS subject for projection
		if err := m.publisher.Publish("inventory.projection.update", event.Payload); err != nil {
			continue
		}

		// Mark as processed
		event.Status = OutboxStatusProcessed
		m.repo.UpdateOutboxEvent(ctx, event)
	}
}

// HandleInventoryProjection updates the inventory projection
func (m *Module) HandleInventoryProjection(msg *nats.Msg) {
	var event InventoryEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return
	}

	ctx := context.Background()
	if err := m.repo.UpsertProjection(ctx, event.ToProjection()); err != nil {
		// Log error
		return
	}
}
