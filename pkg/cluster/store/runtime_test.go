package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHybridRuntimeUsesDedicatedPrimaryKeyAllocator(t *testing.T) {
	runtime, err := NewHybridRuntime(HybridRuntimeOptions{
		DataDir:      t.TempDir(),
		NodeID:       1,
		SlotCount:    8,
		ShardNum:     1,
		MemTableSize: 1024 * 1024,
	})
	require.NoError(t, err)

	require.IsType(t, &SnowflakePrimaryKeyAllocator{}, runtime.PrimaryKeyAllocator)

	conversationStore, ok := runtime.ConversationStore.(*SlotConversationStore)
	require.True(t, ok)
	require.NotNil(t, conversationStore.nextPrimaryKey)

	id1 := runtime.PrimaryKeyAllocator.NextPrimaryKey()
	id2 := conversationStore.nextPrimaryKey()
	require.NotZero(t, id1)
	require.NotZero(t, id2)
	require.NotEqual(t, id1, id2)
	require.IsType(t, &V3MessageStore{}, runtime.ChannelLogStore)
	require.IsType(t, &V3MessageStore{}, runtime.MessageQueryStore)
	require.IsType(t, &V3MessageStore{}, runtime.MessageIndexStore)
	require.IsType(t, &V3MessageStore{}, runtime.MessageSearchStore)
}
