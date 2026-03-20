package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type stubStorageLifecycle struct {
	openCount  int
	closeCount int
	openErr    error
}

func (s *stubStorageLifecycle) Open() error {
	s.openCount++
	return s.openErr
}

func (s *stubStorageLifecycle) Close() error {
	s.closeCount++
	return nil
}

func TestStoreStartStopUsesConfiguredLifecycle(t *testing.T) {
	lifecycle := &stubStorageLifecycle{}
	s := New(NewOptions(WithStorageLifecycle(lifecycle)))

	require.NoError(t, s.Start())
	require.Equal(t, 1, lifecycle.openCount)

	require.NoError(t, s.Start())
	require.Equal(t, 1, lifecycle.openCount)

	s.Stop()
	require.Equal(t, 1, lifecycle.closeCount)

	require.NoError(t, s.Start())
	require.Equal(t, 2, lifecycle.openCount)

	s.Stop()
	require.Equal(t, 2, lifecycle.closeCount)
}

func TestStoreStartStopUsesHybridRuntimeLifecycle(t *testing.T) {
	lifecycle := &stubStorageLifecycle{}
	s := New(NewOptions(WithHybridRuntime(&HybridRuntime{
		Lifecycle: lifecycle,
	})))

	require.NoError(t, s.Start())
	require.Equal(t, 1, lifecycle.openCount)

	s.Stop()
	require.Equal(t, 1, lifecycle.closeCount)
}
