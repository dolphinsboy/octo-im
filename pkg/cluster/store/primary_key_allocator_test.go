package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type stubPrimaryKeyAllocator struct {
	nextPrimaryKeyCalled bool
	nextPrimaryKey       uint64
}

var _ PrimaryKeyAllocator = (*stubPrimaryKeyAllocator)(nil)

func (s *stubPrimaryKeyAllocator) NextPrimaryKey() uint64 {
	s.nextPrimaryKeyCalled = true
	return s.nextPrimaryKey
}

func TestStorePrimaryKeyAllocatorMethodsUseAllocator(t *testing.T) {
	primaryKeyAllocator := &stubPrimaryKeyAllocator{
		nextPrimaryKey: 99,
	}
	s := &Store{primaryKeyAllocator: primaryKeyAllocator}

	require.Equal(t, uint64(99), s.nextPrimaryKey())
	require.True(t, primaryKeyAllocator.nextPrimaryKeyCalled)
}

func TestSnowflakePrimaryKeyAllocatorGeneratesUniqueIDs(t *testing.T) {
	allocator, err := NewSnowflakePrimaryKeyAllocator(1)
	require.NoError(t, err)

	id1 := allocator.NextPrimaryKey()
	id2 := allocator.NextPrimaryKey()

	require.NotZero(t, id1)
	require.NotZero(t, id2)
	require.NotEqual(t, id1, id2)
}
