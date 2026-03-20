package store

import (
	"context"
	"sync"
	"testing"

	"github.com/WuKongIM/WuKongIM/pkg/raft/types"
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/stretchr/testify/require"
)

type testSlot struct {
	slotByKey                   map[string]uint32
	getSlotIdCalls              []string
	proposedSlots               []uint32
	proposedUntilAppliedSlots   []uint32
	proposedUntilAppliedTimeout []uint32
	mu                          sync.Mutex
}

func (t *testSlot) SlotLeaderId(slotId uint32) uint64 {
	return 0
}

func (t *testSlot) GetSlotId(v string) uint32 {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.getSlotIdCalls = append(t.getSlotIdCalls, v)
	return t.slotByKey[v]
}

func (t *testSlot) Propose(slotId uint32, data []byte) (*types.ProposeResp, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.proposedSlots = append(t.proposedSlots, slotId)
	return &types.ProposeResp{}, nil
}

func (t *testSlot) ProposeUntilApplied(slotId uint32, data []byte) (*types.ProposeResp, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.proposedUntilAppliedSlots = append(t.proposedUntilAppliedSlots, slotId)
	return &types.ProposeResp{}, nil
}

func (t *testSlot) ProposeUntilAppliedTimeout(ctx context.Context, slotId uint32, data []byte) (*types.ProposeResp, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.proposedUntilAppliedTimeout = append(t.proposedUntilAppliedTimeout, slotId)
	return &types.ProposeResp{}, nil
}

func newTestStoreWithSlot(slotMap map[string]uint32) (*Store, *testSlot) {
	slot := &testSlot{slotByKey: slotMap}
	return New(NewOptions(WithSlot(slot))), slot
}

func TestAddOrUpdateConversationsRoutesByUIDSlot(t *testing.T) {
	s, slot := newTestStoreWithSlot(map[string]uint32{
		"user-1": 11,
		"user-2": 22,
	})

	err := s.AddOrUpdateConversations([]wkdb.Conversation{
		{Id: 1, Uid: "user-1", ChannelId: "ch-1", ChannelType: 2},
		{Id: 2, Uid: "user-2", ChannelId: "ch-1", ChannelType: 2},
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"user-1", "user-2"}, slot.getSlotIdCalls)
	require.ElementsMatch(t, []uint32{11, 22}, slot.proposedUntilAppliedTimeout)
}

func TestAddOrUpdateUserConversationsRoutesByUIDSlot(t *testing.T) {
	s, slot := newTestStoreWithSlot(map[string]uint32{
		"user-1": 11,
	})

	err := s.AddOrUpdateUserConversations("user-1", []wkdb.Conversation{
		{Id: 1, Uid: "user-1", ChannelId: "ch-1", ChannelType: 2},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"user-1"}, slot.getSlotIdCalls)
	require.Equal(t, []uint32{11}, slot.proposedUntilAppliedSlots)
}

func TestUpdateConversationDeletedAtMsgSeqRoutesByUIDSlot(t *testing.T) {
	s, slot := newTestStoreWithSlot(map[string]uint32{
		"user-1": 11,
	})

	err := s.UpdateConversationDeletedAtMsgSeq("user-1", "group-100", 2, 100)
	require.NoError(t, err)
	require.Equal(t, []string{"user-1"}, slot.getSlotIdCalls)
	require.Equal(t, []uint32{11}, slot.proposedUntilAppliedSlots)
}

func TestDeleteConversationRoutesByUIDSlot(t *testing.T) {
	s, slot := newTestStoreWithSlot(map[string]uint32{
		"user-1": 11,
	})

	err := s.DeleteConversation("user-1", "group-100", 2)
	require.NoError(t, err)
	require.Equal(t, []string{"user-1"}, slot.getSlotIdCalls)
	require.Equal(t, []uint32{11}, slot.proposedUntilAppliedSlots)
}

func TestDeleteConversationsRoutesByUIDSlot(t *testing.T) {
	s, slot := newTestStoreWithSlot(map[string]uint32{
		"user-1": 11,
	})

	err := s.DeleteConversations("user-1", []wkdb.Channel{
		{ChannelId: "group-100", ChannelType: 2},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"user-1"}, slot.getSlotIdCalls)
	require.Equal(t, []uint32{11}, slot.proposedUntilAppliedSlots)
}

func TestUpdateConversationIfSeqGreaterAsyncRoutesByUIDSlot(t *testing.T) {
	s, slot := newTestStoreWithSlot(map[string]uint32{
		"user-1":    11,
		"group-100": 22,
	})

	err := s.UpdateConversationIfSeqGreaterAsync("user-1", "group-100", 2, 100)
	require.NoError(t, err)
	require.Equal(t, []string{"user-1"}, slot.getSlotIdCalls)
	require.Equal(t, []uint32{11}, slot.proposedSlots)
}
