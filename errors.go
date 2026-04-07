package es

import "errors"

var (
	// ErrAlreadyExists is returned when attempting to create an aggregate that already exists.
	ErrAlreadyExists error = errors.New("aggregate already exists")
	// ErrNotFound is returned when attempting to load an aggregate that doesn't exist.
	ErrNotFound error = errors.New("aggregate not found")
	// ErrConcurrency is returned when a concurrency conflict is detected during save operations.
	ErrConcurrency error = errors.New("concurrency error")
	// ErrInvalidEventSpace is a compatibility-preserved sentinel for invalid event
	// compatibility checks, including invalid event-to-aggregate mappings in alternate
	// workflows. The default aggregate implementation panics instead of returning it.
	ErrInvalidEventSpace error = errors.New("invalid event type")
	// ErrEventHandlerNotFound is available to alternate aggregate workflows that need
	// explicit handler lookup failures.
	ErrEventHandlerNotFound error = errors.New("event handler not found")
	// ErrInvalidEntity is returned when entity validation fails.
	ErrInvalidEntity error = errors.New("invalid entity")
)

type wrappedSentinelError struct {
	message  string
	sentinel error
}

func (e wrappedSentinelError) Error() string {
	return e.message
}

func (e wrappedSentinelError) Unwrap() error {
	return e.sentinel
}

func wrapSentinelError(message string, sentinel error) error {
	return wrappedSentinelError{message: message, sentinel: sentinel}
}
