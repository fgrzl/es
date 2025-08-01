package es

// Scope defines the visibility scope for entities and events.
type Scope int

const (
	// ScopeGlobal indicates global visibility across all tenants.
	ScopeGlobal Scope = iota
	// ScopeTenant indicates visibility restricted to a specific tenant.
	ScopeTenant
)
