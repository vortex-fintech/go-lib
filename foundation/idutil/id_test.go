package idutil

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type testTag struct{}

func TestNewID(t *testing.T) {
	id, err := NewID[testTag]()
	assert.NoError(t, err)
	assert.False(t, id.IsZero())
	assert.Len(t, id.String(), 36)
	assert.Equal(t, uuid.Version(7), id.UUID.Version())
}

func TestParseID(t *testing.T) {
	original, _ := NewID[testTag]()
	s := original.String()

	parsed, err := ParseID[testTag](s)
	assert.NoError(t, err)
	assert.Equal(t, original, parsed)
}

func TestParseID_Invalid(t *testing.T) {
	_, err := ParseID[testTag]("invalid")
	assert.Error(t, err)
}

func TestIsZero(t *testing.T) {
	assert.True(t, ID[testTag]{}.IsZero())
	id, _ := NewID[testTag]()
	assert.False(t, id.IsZero())
}

func TestTypeSafety(t *testing.T) {
	type tagA struct{}
	type tagB struct{}

	idA, _ := NewID[tagA]()
	idB, _ := NewID[tagB]()

	assert.NotEqual(t, idA.String(), idB.String())

	var _ ID[tagA] = idA
	var _ ID[tagB] = idB
}
