package messaging

import (
	"context"
	"log"
	"time"
)

type HandlerFunc func(ctx context.Context, job *RecipeJob) error

type WorkerPool struct {
	queue      RedisQueue
	handler    HandlerFunc
	numWorkers int
	timeout    time.Duration
}

func NewWorkerPool(queue RedisQueue, handler HandlerFunc, numWorkers int, timeout time.Duration) *WorkerPool {
	return &WorkerPool{queue: queue, handler: handler, numWorkers: numWorkers, timeout: timeout}
}

func (w *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < w.numWorkers; i++ {
		go w.runWorker(ctx, i)
	}
}

func (w *WorkerPool) runWorker(ctx context.Context, id int) {
	log.Printf("Worker %d started", id)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d exiting", id)
			return
		default:
		}

		job, err := w.queue.Dequeue(ctx, w.timeout)
		if err != nil {
			if ctx.Err() != nil {
				return // context cancelled, exit
			}
			log.Printf("Worker %d error: %v", id, err)
			continue
		}

		err = w.handler(ctx, job)
		if err != nil {
			log.Printf("Worker %d error: %v", id, err)
			err = w.queue.Nack(ctx, job, false)
			if err != nil {
				return
			}
		}

		_ = w.queue.Ack(ctx, job)
	}
}
