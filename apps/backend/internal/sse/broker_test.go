package sse

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBroker_AddClientReceivesBroadcast(t *testing.T) {
	b := NewBroker()
	b.Start()

	client := b.AddClient()
	defer b.RemoveClient(client)

	b.BroadcastChan <- "hello"

	select {
	case msg := <-client:
		if msg != "hello" {
			t.Errorf("expected 'hello', got %q", msg)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for broadcast message")
	}
}

func TestBroker_RemoveClientClosesChannelAndBrokerKeepsRunning(t *testing.T) {
	b := NewBroker()
	b.Start()

	client := b.AddClient()
	b.RemoveClient(client)

	// Channel should be closed by the broker; a receive should return immediately
	// with the zero value and ok == false.
	select {
	case _, ok := <-client:
		if ok {
			t.Error("expected channel to be closed after RemoveClient")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for channel to close")
	}

	// Broker must still be alive and able to serve other clients afterward.
	other := b.AddClient()
	defer b.RemoveClient(other)
	b.BroadcastChan <- "still alive"

	select {
	case msg := <-other:
		if msg != "still alive" {
			t.Errorf("expected 'still alive', got %q", msg)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("broker did not process broadcast after a client was removed")
	}
}

func TestBroker_FanOutToMultipleClients(t *testing.T) {
	b := NewBroker()
	b.Start()

	clientA := b.AddClient()
	clientB := b.AddClient()
	defer b.RemoveClient(clientA)
	defer b.RemoveClient(clientB)

	b.BroadcastChan <- "fanout"

	for name, c := range map[string]chan string{"A": clientA, "B": clientB} {
		select {
		case msg := <-c:
			if msg != "fanout" {
				t.Errorf("client %s: expected 'fanout', got %q", name, msg)
			}
		case <-time.After(1 * time.Second):
			t.Fatalf("client %s: timed out waiting for broadcast", name)
		}
	}
}

func TestBroker_SlowClientDoesNotBlockOtherClients(t *testing.T) {
	b := NewBroker()
	b.Start()

	slow := b.AddClient() // buffered at 10, never drained
	fast := b.AddClient()
	defer b.RemoveClient(slow)
	defer b.RemoveClient(fast)

	// Fill the slow client's buffer beyond capacity plus a couple more
	// broadcasts, none of which should block the broker loop.
	for i := 0; i < 15; i++ {
		b.BroadcastChan <- "msg"
		select {
		case <-fast:
			// drain immediately so fast never fills up
		case <-time.After(1 * time.Second):
			t.Fatalf("broker appears blocked by slow client at iteration %d", i)
		}
	}
}

func TestBroadcastInventoryUpdate_AvailableStatus(t *testing.T) {
	GlobalBroker = NewBroker()
	GlobalBroker.Start()
	client := GlobalBroker.AddClient()
	defer GlobalBroker.RemoveClient(client)

	BroadcastInventoryUpdate("VIP", 5)

	msg := readMessage(t, client)
	if !strings.HasPrefix(msg, "event: inventory_update\n") {
		t.Fatalf("expected inventory_update event, got %q", msg)
	}

	payload := extractDataPayload(t, msg)
	if payload["category"] != "VIP" {
		t.Errorf("expected category VIP, got %v", payload["category"])
	}
	if payload["status"] != "Available" {
		t.Errorf("expected status Available for available=5, got %v", payload["status"])
	}
}

func TestBroadcastInventoryUpdate_SoldOutStatus(t *testing.T) {
	GlobalBroker = NewBroker()
	GlobalBroker.Start()
	client := GlobalBroker.AddClient()
	defer GlobalBroker.RemoveClient(client)

	BroadcastInventoryUpdate("Standard", 0)

	msg := readMessage(t, client)
	payload := extractDataPayload(t, msg)
	if payload["status"] != "Sold Out" {
		t.Errorf("expected status 'Sold Out' for available=0, got %v", payload["status"])
	}
}

func TestBroadcastEventSoldOut(t *testing.T) {
	GlobalBroker = NewBroker()
	GlobalBroker.Start()
	client := GlobalBroker.AddClient()
	defer GlobalBroker.RemoveClient(client)

	BroadcastEventSoldOut("Concert Finale")

	msg := readMessage(t, client)
	if !strings.HasPrefix(msg, "event: event_sold_out\n") {
		t.Fatalf("expected event_sold_out event, got %q", msg)
	}

	payload := extractDataPayload(t, msg)
	message, _ := payload["message"].(string)
	if !strings.Contains(message, "Concert Finale") {
		t.Errorf("expected message to mention event name, got %q", message)
	}
}

func readMessage(t *testing.T, c chan string) string {
	t.Helper()
	select {
	case msg := <-c:
		return msg
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for broadcast message")
		return ""
	}
}

func extractDataPayload(t *testing.T, sseMessage string) map[string]interface{} {
	t.Helper()
	lines := strings.Split(sseMessage, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &payload); err != nil {
				t.Fatalf("failed to unmarshal SSE data payload: %v", err)
			}
			return payload
		}
	}
	t.Fatalf("no data line found in SSE message: %q", sseMessage)
	return nil
}
