package transport

import "errors"

var (
	// ErrTransportClosed indicates that the transport has been closed
	ErrTransportClosed = errors.New("transport is closed")
	// ErrQueueFull indicates that the transport queue is full
	ErrQueueFull = errors.New("transport queue is full")
)
