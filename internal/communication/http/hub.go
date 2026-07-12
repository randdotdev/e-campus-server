package http

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/communication"
)

const maxConnectionsPerUser = 5

// Hub is the websocket connection manager. It implements
// communication.Broadcaster — the realtime side of the Notifier.
type Hub struct {
	mu         sync.RWMutex
	clients    map[uuid.UUID]map[*Client]bool
	register   chan *Client
	unregister chan *Client
}

var _ communication.Broadcaster = (*Hub)(nil)

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run pumps registrations until the context ends.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.register:
			h.addClient(client)
		case client := <-h.unregister:
			h.removeClient(client)
		case <-ctx.Done():
			return
		}
	}
}

func (h *Hub) addClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[client.userID] == nil {
		h.clients[client.userID] = make(map[*Client]bool)
	}
	if len(h.clients[client.userID]) >= maxConnectionsPerUser {
		for oldest := range h.clients[client.userID] {
			delete(h.clients[client.userID], oldest)
			close(oldest.send)
			break
		}
	}
	h.clients[client.userID][client] = true
}

func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[client.userID]; ok {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			close(client.send)
			if len(clients) == 0 {
				delete(h.clients, client.userID)
			}
		}
	}
}

func (h *Hub) Register(client *Client)   { h.register <- client }
func (h *Hub) Unregister(client *Client) { h.unregister <- client }

func (h *Hub) Broadcast(userID uuid.UUID, notification *communication.Notification) {
	data, err := json.Marshal(toNotificationResponse(notification))
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients[userID] {
		h.trySend(client, data)
	}
}

func (h *Hub) trySend(client *Client, data []byte) {
	defer func() {
		if recover() != nil {
			go func() { h.unregister <- client }()
		}
	}()
	select {
	case client.send <- data:
	default:
		go func() { h.unregister <- client }()
	}
}

func (h *Hub) ConnectionCount(userID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}
