package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Message struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	logger     *zap.Logger
	redis      *redis.Client
}

func NewHub(logger *zap.Logger, rdb *redis.Client) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
		redis:      rdb,
	}
}

func (h *Hub) Run() {
	if h.redis != nil {
		go h.listenRedis()
	}
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) listenRedis() {
	ctx := context.Background()
	pubsub := h.redis.Subscribe(ctx, "updates")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		var m Message
		if err := json.Unmarshal([]byte(msg.Payload), &m); err != nil {
			h.logger.Error("failed to unmarshal redis message", zap.Error(err))
			continue
		}
		h.broadcast <- m
	}
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}
	client := &Client{hub: h, conn: conn, send: make(chan Message, 256)}
	h.register <- client

	go client.writePump()
	go client.readPump()
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan Message
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteJSON(msg); err != nil {
			break
		}
	}
}
