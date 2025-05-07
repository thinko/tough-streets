package processor

import (
	"context"
	"sync"

	"tough-streets/internal/transport"
)

// PacketProcessor defines the interface for processing packets
type PacketProcessor interface {
	Process(ctx context.Context, packet transport.CapturedPacket) error
}

// WorkerPool manages a pool of packet processing workers
type WorkerPool struct {
	processor  PacketProcessor
	consumer   transport.PacketConsumer
	workerCount int
	wg         sync.WaitGroup
}

// NewWorkerPool creates a new worker pool for packet processing
func NewWorkerPool(processor PacketProcessor, consumer transport.PacketConsumer, workerCount int) *WorkerPool {
	if workerCount <= 0 {
		workerCount = 1
	}
	return &WorkerPool{
		processor:   processor,
		consumer:    consumer,
		workerCount: workerCount,
	}
}

// Start begins the worker pool
func (wp *WorkerPool) Start(ctx context.Context) error {
	packetChan, err := wp.consumer.Consume(ctx)
	if err != nil {
		return err
	}

	// Start the workers
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(ctx, packetChan)
	}

	return nil
}

// worker is the main processing loop for each worker
func (wp *WorkerPool) worker(ctx context.Context, packetChan transport.PacketChannel) {
	defer wp.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case packet, ok := <-packetChan:
			if !ok {
				return
			}
			if err := wp.processor.Process(ctx, packet); err != nil {
				// TODO: Add error handling and metrics
				continue
			}
		}
	}
}

// Stop gracefully shuts down the worker pool
func (wp *WorkerPool) Stop() {
	wp.wg.Wait()
}
