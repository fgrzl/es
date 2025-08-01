package es

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldHaveCorrectScopeValues(t *testing.T) {
	// Arrange & Act & Assert
	assert.Equal(t, Scope(0), ScopeGlobal)
	assert.Equal(t, Scope(1), ScopeTenant)
}

func TestShouldDifferentiateBetweenScopes(t *testing.T) {
	// Arrange & Act & Assert
	assert.NotEqual(t, ScopeGlobal, ScopeTenant)
}
