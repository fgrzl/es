package es

type mockDomainEvent struct {
	*DomainEventBase
}

func (m *mockDomainEvent) GetDiscriminator() string {
	return "es://mock_domain_event"
}

func (m *mockDomainEvent) GetSpaces() []string {
	return []string{"test-area"}
}

// Optionally, implement other DomainEvent methods as needed for tests.
