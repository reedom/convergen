package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reedom/convergen/v8/pkg/domain"
)

// Simple test to verify package compiles and basic functionality works.
func TestPackageCompilation(t *testing.T) {
	// Test that we can create basic domain types
	stringType := domain.StringType
	assert.NotNil(t, stringType)
	assert.Equal(t, "string", stringType.Name())
	assert.Equal(t, domain.KindBasic, stringType.Kind())

	// Test that we can create type cache
	cache := NewTypeCache(10)
	assert.NotNil(t, cache)
	assert.Equal(t, 0, cache.Size())
}
