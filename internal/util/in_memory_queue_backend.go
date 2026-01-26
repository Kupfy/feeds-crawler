package util

import (
	"errors"
	"sync"
)

// InMemoryQueueBackend implements queue.Storage interface
type InMemoryQueueBackend struct {
	mu       sync.Mutex
	requests [][]byte
}

func (b *InMemoryQueueBackend) Init() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.requests = make([][]byte, 0)
	return nil
}

func (b *InMemoryQueueBackend) AddRequest(req []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Make a copy to avoid slice mutation issues
	reqCopy := make([]byte, len(req))
	copy(reqCopy, req)

	b.requests = append(b.requests, reqCopy)
	return nil
}

func (b *InMemoryQueueBackend) GetRequest() ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.requests) == 0 {
		// Return error when empty - this tells the queue to stop
		return nil, errors.New("queue is empty")
	}

	// Pop the first request (FIFO)
	req := b.requests[0]
	b.requests = b.requests[1:]

	return req, nil
}

func (b *InMemoryQueueBackend) QueueSize() (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.requests), nil
}

func (b *InMemoryQueueBackend) Close() error {
	return nil
}
