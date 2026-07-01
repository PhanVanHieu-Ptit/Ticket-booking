package sse

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Broker manages connected clients and coordinates SSE event broadcasting.
type Broker struct {
	mu             sync.RWMutex
	clients        map[chan string]bool
	BroadcastChan  chan string
	newClients     chan chan string
	defunctClients chan chan string
}

// GlobalBroker is the shared SSE broker instance.
var GlobalBroker *Broker

// NewBroker initializes and returns a new Broker.
func NewBroker() *Broker {
	return &Broker{
		clients:        make(map[chan string]bool),
		BroadcastChan:  make(chan string, 100),
		newClients:     make(chan chan string),
		defunctClients: make(chan chan string),
	}
}

// Start runs the broker connection handling loop in a separate goroutine.
func (b *Broker) Start() {
	go func() {
		for {
			select {
			case s := <-b.newClients:
				b.mu.Lock()
				b.clients[s] = true
				b.mu.Unlock()
			case s := <-b.defunctClients:
				b.mu.Lock()
				delete(b.clients, s)
				close(s)
				b.mu.Unlock()
			case msg := <-b.BroadcastChan:
				b.mu.RLock()
				for clientChan := range b.clients {
					select {
					case clientChan <- msg:
					default:
						// Non-blocking write to prevent slow clients from blocking the broker loop
					}
				}
				b.mu.RUnlock()
			}
		}
	}()
}

// AddClient registers a new client channel with the broker.
func (b *Broker) AddClient() chan string {
	c := make(chan string, 10)
	b.newClients <- c
	return c
}

// RemoveClient unregisters and cleans up a client channel.
func (b *Broker) RemoveClient(c chan string) {
	b.defunctClients <- c
}

// BroadcastInventoryUpdate formats and broadcasts a category's remaining ticket count to all subscribers.
func BroadcastInventoryUpdate(category string, available int64) {
	status := "Available"
	if available <= 0 {
		status = "Sold Out"
	}

	payload := map[string]interface{}{
		"category":  category,
		"available": available,
		"status":    status,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	msg := fmt.Sprintf("event: inventory_update\ndata: %s\n\n", data)
	select {
	case GlobalBroker.BroadcastChan <- msg:
	default:
	}
}

// BroadcastEventSoldOut broadcasts that all tickets are sold out.
func BroadcastEventSoldOut(eventName string) {
	payload := map[string]interface{}{
		"message": fmt.Sprintf("All tickets for %s are sold out!", eventName),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	msg := fmt.Sprintf("event: event_sold_out\ndata: %s\n\n", data)
	select {
	case GlobalBroker.BroadcastChan <- msg:
	default:
	}
}
