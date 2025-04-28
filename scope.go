package es

type Scope int

const (
	ScopeGlobal Scope = iota
	ScopeTenant
)
