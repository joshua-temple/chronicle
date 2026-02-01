package daemon

import (
	"sync"
	"testing"
	"time"
)

func TestNewEmbeddedEventBus(t *testing.T) {
	bus := NewEmbeddedEventBus()

	if bus == nil {
		t.Fatal("NewEmbeddedEventBus() returned nil")
	}

	if bus.typeHandlers == nil {
		t.Error("NewEmbeddedEventBus() did not initialize typeHandlers")
	}

	if bus.handlerIDs == nil {
		t.Error("NewEmbeddedEventBus() did not initialize handlerIDs")
	}
}

func TestEmbeddedEventBus_Publish_Subscribe(t *testing.T) {
	bus := NewEmbeddedEventBus()

	received := make(chan Event, 1)
	unsub := bus.Subscribe(func(e Event) {
		received <- e
	})
	defer unsub()

	event := Event{
		Type:      EventRunStarted,
		Timestamp: time.Now(),
		Data:      map[string]any{"run_id": "test-1"},
	}

	bus.Publish(event)

	select {
	case e := <-received:
		if e.Type != event.Type {
			t.Errorf("Received event type = %q, expected %q", e.Type, event.Type)
		}
		if e.Data["run_id"] != "test-1" {
			t.Error("Received event data mismatch")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive event in time")
	}
}

func TestEmbeddedEventBus_SubscribeType(t *testing.T) {
	bus := NewEmbeddedEventBus()

	startedEvents := make(chan Event, 1)
	completedEvents := make(chan Event, 1)

	unsubStarted := bus.SubscribeType(EventRunStarted, func(e Event) {
		startedEvents <- e
	})
	defer unsubStarted()

	unsubCompleted := bus.SubscribeType(EventRunCompleted, func(e Event) {
		completedEvents <- e
	})
	defer unsubCompleted()

	// Publish started event
	bus.Publish(Event{
		Type:      EventRunStarted,
		Timestamp: time.Now(),
	})

	// Publish completed event
	bus.Publish(Event{
		Type:      EventRunCompleted,
		Timestamp: time.Now(),
	})

	// Should receive started event only in startedEvents channel
	select {
	case <-startedEvents:
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive started event")
	}

	// Should receive completed event only in completedEvents channel
	select {
	case <-completedEvents:
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive completed event")
	}
}

func TestEmbeddedEventBus_Unsubscribe(t *testing.T) {
	bus := NewEmbeddedEventBus()

	received := 0
	var mu sync.Mutex

	unsub := bus.Subscribe(func(e Event) {
		mu.Lock()
		received++
		mu.Unlock()
	})

	// Publish first event
	bus.Publish(Event{Type: EventRunStarted, Timestamp: time.Now()})
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if received != 1 {
		mu.Unlock()
		t.Errorf("Should have received 1 event, got %d", received)
		return
	}
	mu.Unlock()

	// Unsubscribe
	unsub()

	// Publish second event
	bus.Publish(Event{Type: EventRunStarted, Timestamp: time.Now()})
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if received != 1 {
		mu.Unlock()
		t.Errorf("Should still have received only 1 event after unsubscribe, got %d", received)
		return
	}
	mu.Unlock()
}

func TestEmbeddedEventBus_ConcurrentPublish(t *testing.T) {
	bus := NewEmbeddedEventBus()

	received := make(chan Event, 100)
	unsub := bus.Subscribe(func(e Event) {
		received <- e
	})
	defer unsub()

	// Publish many events concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			bus.Publish(Event{
				Type:      EventRunStarted,
				Timestamp: time.Now(),
				Data:      map[string]any{"id": id},
			})
		}(i)
	}

	wg.Wait()

	// Wait for handlers to complete
	time.Sleep(100 * time.Millisecond)

	close(received)

	count := 0
	for range received {
		count++
	}

	if count != 10 {
		t.Errorf("Should have received 10 events, got %d", count)
	}
}

func TestNewBufferedEventBus(t *testing.T) {
	bus := NewBufferedEventBus(10)

	if bus == nil {
		t.Fatal("NewBufferedEventBus() returned nil")
	}

	if bus.maxSize != 10 {
		t.Errorf("NewBufferedEventBus() maxSize = %d, expected 10", bus.maxSize)
	}

	if bus.buffer == nil {
		t.Error("NewBufferedEventBus() did not initialize buffer")
	}
}

func TestBufferedEventBus_Publish(t *testing.T) {
	bus := NewBufferedEventBus(5)

	received := make(chan Event, 1)
	unsub := bus.Subscribe(func(e Event) {
		received <- e
	})
	defer unsub()

	event := Event{
		Type:      EventRunStarted,
		Timestamp: time.Now(),
	}

	bus.Publish(event)

	// Should be received by subscriber
	select {
	case <-received:
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive event")
	}

	// Should be in buffer
	history := bus.GetHistory(time.Time{})
	if len(history) != 1 {
		t.Errorf("Buffer should have 1 event, got %d", len(history))
	}
}

func TestBufferedEventBus_MaxSize(t *testing.T) {
	bus := NewBufferedEventBus(3)

	// Publish more events than maxSize
	for i := 0; i < 5; i++ {
		bus.Publish(Event{
			Type:      EventRunStarted,
			Timestamp: time.Now(),
			Data:      map[string]any{"id": i},
		})
	}

	history := bus.GetHistory(time.Time{})
	if len(history) != 3 {
		t.Errorf("Buffer should have 3 events (maxSize), got %d", len(history))
	}

	// Should have the most recent events (2, 3, 4)
	for i, e := range history {
		expectedID := i + 2
		if e.Data["id"] != expectedID {
			t.Errorf("Event %d ID = %v, expected %d", i, e.Data["id"], expectedID)
		}
	}
}

func TestBufferedEventBus_GetHistory(t *testing.T) {
	bus := NewBufferedEventBus(10)

	now := time.Now()

	// Publish events with different timestamps
	for i := 0; i < 5; i++ {
		bus.Publish(Event{
			Type:      EventRunStarted,
			Timestamp: now.Add(time.Duration(i) * time.Second),
			Data:      map[string]any{"id": i},
		})
	}

	// Get all history
	all := bus.GetHistory(time.Time{})
	if len(all) != 5 {
		t.Errorf("GetHistory() returned %d events, expected 5", len(all))
	}

	// Get history after specific time
	afterSecond2 := bus.GetHistory(now.Add(2 * time.Second))
	if len(afterSecond2) != 2 {
		t.Errorf("GetHistory(after 2s) returned %d events, expected 2", len(afterSecond2))
	}
}

func TestBufferedEventBus_GetHistoryByType(t *testing.T) {
	bus := NewBufferedEventBus(10)

	now := time.Now()

	// Publish events of different types
	bus.Publish(Event{Type: EventRunStarted, Timestamp: now})
	bus.Publish(Event{Type: EventRunCompleted, Timestamp: now.Add(time.Second)})
	bus.Publish(Event{Type: EventRunStarted, Timestamp: now.Add(2 * time.Second)})
	bus.Publish(Event{Type: EventRunFailed, Timestamp: now.Add(3 * time.Second)})

	// Get only started events
	started := bus.GetHistoryByType(EventRunStarted, time.Time{})
	if len(started) != 2 {
		t.Errorf("GetHistoryByType(started) returned %d events, expected 2", len(started))
	}

	// Get completed events
	completed := bus.GetHistoryByType(EventRunCompleted, time.Time{})
	if len(completed) != 1 {
		t.Errorf("GetHistoryByType(completed) returned %d events, expected 1", len(completed))
	}

	// Get started events after specific time
	startedAfter := bus.GetHistoryByType(EventRunStarted, now.Add(time.Second))
	if len(startedAfter) != 1 {
		t.Errorf("GetHistoryByType(started, after 1s) returned %d events, expected 1", len(startedAfter))
	}
}

func TestEmbeddedEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewEmbeddedEventBus()

	received1 := make(chan Event, 1)
	received2 := make(chan Event, 1)

	unsub1 := bus.Subscribe(func(e Event) {
		received1 <- e
	})
	defer unsub1()

	unsub2 := bus.Subscribe(func(e Event) {
		received2 <- e
	})
	defer unsub2()

	bus.Publish(Event{Type: EventRunStarted, Timestamp: time.Now()})

	// Both should receive the event
	select {
	case <-received1:
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Error("Subscriber 1 did not receive event")
	}

	select {
	case <-received2:
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Error("Subscriber 2 did not receive event")
	}
}
