package es

type mockDomainEvent struct {
	*DomainEventBase
}

func (m *mockDomainEvent) GetSpaces() []string {
	return []string{"test-area"}
}

// Optionally, implement other DomainEvent methods as needed for tests.
