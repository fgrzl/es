package es

import "errors"

var (
	// ErrAlreadyExists is returned when attempting to create an aggregate that already exists.
	ErrAlreadyExists error = errors.New("aggregate already exists")
	// ErrNotFound is returned when attempting to load an aggregate that doesn't exist.
	ErrNotFound error = errors.New("aggregate not found")
	// ErrConcurrency is returned when a concurrency conflict is detected during save operations.
	ErrConcurrency error = errors.New("concurrency error")
	// ErrInvalidEventSpace is returned when an event has an invalid discriminator or type.
	ErrInvalidEventSpace error = errors.New("invalid event type")
	// ErrEventHandlerNotFound is returned when no handler is registered for an event type.
	ErrEventHandlerNotFound error = errors.New("event handler not found")
)
