package es

import "errors"

var (
	ErrAlreadyExists        error = errors.New("aggregate already exists")
	ErrNotFound             error = errors.New("aggregate not found")
	ErrConcurrency          error = errors.New("concurrency error")
	ErrInvalidEventType     error = errors.New("invalid event type")
	ErrEventHandlerNotFound error = errors.New("event handler not found")
)
