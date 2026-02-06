// Copyright (c) Microsoft. All rights reserved.

package mcp

import (
	"encoding/json"
	"sync"
)

type notificationHub struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
}

func newNotificationHub() *notificationHub {
	return &notificationHub{
		clients: make(map[chan []byte]struct{}),
	}
}

func (h *notificationHub) Subscribe() chan []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	client := make(chan []byte, 100)
	h.clients[client] = struct{}{}
	return client
}

func (h *notificationHub) Unsubscribe(client chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client)
	}
}

func (h *notificationHub) Broadcast(notification Notification) {
	payload, err := json.Marshal(notification)
	if err != nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		select {
		case client <- payload:
		default:
		}
	}
}
