package store

import (
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/stretchr/testify/require"
)

type stubChannelStateStore struct {
	channel                wkdb.ChannelInfo
	channelExists          bool
	subscribers            []wkdb.Member
	denylist               []wkdb.Member
	allowlist              []wkdb.Member
	hasAllowlist           bool
	lastSubscriberUID      string
	lastDenylistUID        string
	lastAllowlistUID       string
	lastChannelID          string
	lastChannelType        uint8
	lastLookupKind         string
	lastAddedSubscribers   []wkdb.Member
	lastRemovedSubscribers []string
	removeAllSubscriber    bool
	lastAddedChannel       wkdb.ChannelInfo
	lastUpdatedChannel     wkdb.ChannelInfo
	deleteChannelCalled    bool
	lastAddedDenylist      []wkdb.Member
	lastRemovedDenylist    []string
	removeAllDenylist      bool
	lastAddedAllowlist     []wkdb.Member
	lastRemovedAllowlist   []string
	removeAllAllowlist     bool
}

var _ ChannelStateStore = (*stubChannelStateStore)(nil)

func (s *stubChannelStateStore) AddSubscribers(channelId string, channelType uint8, members []wkdb.Member) error {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastLookupKind = "add_subscribers"
	s.lastAddedSubscribers = append([]wkdb.Member(nil), members...)
	s.subscribers = append([]wkdb.Member(nil), members...)
	return nil
}

func (s *stubChannelStateStore) RemoveSubscribers(channelId string, channelType uint8, uids []string) error {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastLookupKind = "remove_subscribers"
	s.lastRemovedSubscribers = append([]string(nil), uids...)
	return nil
}

func (s *stubChannelStateStore) ExistSubscriber(channelId string, channelType uint8, uid string) (bool, error) {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastSubscriberUID = uid
	s.lastLookupKind = "subscriber"
	for _, member := range s.subscribers {
		if member.Uid == uid {
			return true, nil
		}
	}
	return false, nil
}

func (s *stubChannelStateStore) RemoveAllSubscriber(channelId string, channelType uint8) error {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastLookupKind = "remove_all_subscribers"
	s.removeAllSubscriber = true
	s.subscribers = nil
	return nil
}

func (s *stubChannelStateStore) GetSubscribers(channelId string, channelType uint8) ([]wkdb.Member, error) {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastLookupKind = "subscribers"
	return append([]wkdb.Member(nil), s.subscribers...), nil
}

func (s *stubChannelStateStore) AddChannel(channelInfo wkdb.ChannelInfo) (uint64, error) {
	s.lastLookupKind = "add_channel"
	s.lastAddedChannel = channelInfo
	s.channel = channelInfo
	s.channelExists = true
	return channelInfo.Id, nil
}

func (s *stubChannelStateStore) GetChannel(channelId string, channelType uint8) (wkdb.ChannelInfo, error) {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastLookupKind = "channel"
	return s.channel, nil
}

func (s *stubChannelStateStore) UpdateChannel(channelInfo wkdb.ChannelInfo) error {
	s.lastLookupKind = "update_channel"
	s.lastUpdatedChannel = channelInfo
	s.channel = channelInfo
	s.channelExists = true
	return nil
}

func (s *stubChannelStateStore) ExistChannel(channelId string, channelType uint8) (bool, error) {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastLookupKind = "channel_exists"
	return s.channelExists, nil
}

func (s *stubChannelStateStore) DeleteChannel(channelId string, channelType uint8) error {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastLookupKind = "delete_channel"
	s.deleteChannelCalled = true
	s.channel = wkdb.ChannelInfo{}
	s.channelExists = false
	return nil
}

func (s *stubChannelStateStore) AddDenylist(channelId string, channelType uint8, members []wkdb.Member) error {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastLookupKind = "add_denylist"
	s.lastAddedDenylist = append([]wkdb.Member(nil), members...)
	s.denylist = append([]wkdb.Member(nil), members...)
	return nil
}

func (s *stubChannelStateStore) GetDenylist(channelId string, channelType uint8) ([]wkdb.Member, error) {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastLookupKind = "denylist"
	return append([]wkdb.Member(nil), s.denylist...), nil
}

func (s *stubChannelStateStore) RemoveDenylist(channelId string, channelType uint8, uids []string) error {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastLookupKind = "remove_denylist"
	s.lastRemovedDenylist = append([]string(nil), uids...)
	return nil
}

func (s *stubChannelStateStore) RemoveAllDenylist(channelId string, channelType uint8) error {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastLookupKind = "remove_all_denylist"
	s.removeAllDenylist = true
	s.denylist = nil
	return nil
}

func (s *stubChannelStateStore) ExistDenylist(channelId string, channelType uint8, uid string) (bool, error) {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastDenylistUID = uid
	s.lastLookupKind = "denylist_exists"
	for _, member := range s.denylist {
		if member.Uid == uid {
			return true, nil
		}
	}
	return false, nil
}

func (s *stubChannelStateStore) AddAllowlist(channelId string, channelType uint8, members []wkdb.Member) error {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastLookupKind = "add_allowlist"
	s.lastAddedAllowlist = append([]wkdb.Member(nil), members...)
	s.allowlist = append([]wkdb.Member(nil), members...)
	s.hasAllowlist = len(members) > 0
	return nil
}

func (s *stubChannelStateStore) GetAllowlist(channelId string, channelType uint8) ([]wkdb.Member, error) {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastLookupKind = "allowlist"
	return append([]wkdb.Member(nil), s.allowlist...), nil
}

func (s *stubChannelStateStore) RemoveAllowlist(channelId string, channelType uint8, uids []string) error {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastLookupKind = "remove_allowlist"
	s.lastRemovedAllowlist = append([]string(nil), uids...)
	return nil
}

func (s *stubChannelStateStore) RemoveAllAllowlist(channelId string, channelType uint8) error {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastLookupKind = "remove_all_allowlist"
	s.removeAllAllowlist = true
	s.allowlist = nil
	s.hasAllowlist = false
	return nil
}

func (s *stubChannelStateStore) ExistAllowlist(channelId string, channelType uint8, uid string) (bool, error) {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastAllowlistUID = uid
	s.lastLookupKind = "allowlist_exists"
	for _, member := range s.allowlist {
		if member.Uid == uid {
			return true, nil
		}
	}
	return false, nil
}

func (s *stubChannelStateStore) HasAllowlist(channelId string, channelType uint8) (bool, error) {
	s.lastChannelID = channelId
	s.lastChannelType = channelType
	s.lastLookupKind = "has_allowlist"
	return s.hasAllowlist, nil
}

func TestSlotChannelStateStoreReadsFromSlotDB(t *testing.T) {
	slotCount := uint32(2)
	db, routeSlot := newAdminSearchTestDB(t, slotCount)
	store := NewSlotChannelStateStore(db, routeSlot)
	now := time.Unix(1710000000, 0)

	channelID := "group-1"
	channelType := uint8(2)
	_, err := store.AddChannel(wkdb.ChannelInfo{
		Id:          1,
		ChannelId:   channelID,
		ChannelType: channelType,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	})
	require.NoError(t, err)
	require.NoError(t, store.AddSubscribers(channelID, channelType, []wkdb.Member{
		{Uid: "sub-2", CreatedAt: &now, UpdatedAt: &now},
		{Uid: "sub-1", CreatedAt: &now, UpdatedAt: &now},
	}))
	require.NoError(t, store.AddAllowlist(channelID, channelType, []wkdb.Member{
		{Uid: "allow-1", CreatedAt: &now, UpdatedAt: &now},
	}))
	require.NoError(t, store.AddDenylist(channelID, channelType, []wkdb.Member{
		{Uid: "deny-1", CreatedAt: &now, UpdatedAt: &now},
	}))

	channel, err := store.GetChannel(channelID, channelType)
	require.NoError(t, err)
	require.Equal(t, channelID, channel.ChannelId)

	exists, err := store.ExistChannel(channelID, channelType)
	require.NoError(t, err)
	require.True(t, exists)

	subscribers, err := store.GetSubscribers(channelID, channelType)
	require.NoError(t, err)
	require.Len(t, subscribers, 2)
	require.Equal(t, []string{"sub-1", "sub-2"}, []string{subscribers[0].Uid, subscribers[1].Uid})

	exists, err = store.ExistSubscriber(channelID, channelType, "sub-1")
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = store.ExistSubscriber(channelID, channelType, "missing")
	require.NoError(t, err)
	require.False(t, exists)

	allowlist, err := store.GetAllowlist(channelID, channelType)
	require.NoError(t, err)
	require.Len(t, allowlist, 1)
	require.Equal(t, "allow-1", allowlist[0].Uid)

	exists, err = store.ExistAllowlist(channelID, channelType, "allow-1")
	require.NoError(t, err)
	require.True(t, exists)

	hasAllowlist, err := store.HasAllowlist(channelID, channelType)
	require.NoError(t, err)
	require.True(t, hasAllowlist)

	denylist, err := store.GetDenylist(channelID, channelType)
	require.NoError(t, err)
	require.Len(t, denylist, 1)
	require.Equal(t, "deny-1", denylist[0].Uid)

	exists, err = store.ExistDenylist(channelID, channelType, "deny-1")
	require.NoError(t, err)
	require.True(t, exists)

	require.NoError(t, store.RemoveSubscribers(channelID, channelType, []string{"sub-2"}))
	subscribers, err = store.GetSubscribers(channelID, channelType)
	require.NoError(t, err)
	require.Len(t, subscribers, 1)
	require.Equal(t, "sub-1", subscribers[0].Uid)

	require.NoError(t, store.RemoveAllowlist(channelID, channelType, []string{"allow-1"}))
	hasAllowlist, err = store.HasAllowlist(channelID, channelType)
	require.NoError(t, err)
	require.False(t, hasAllowlist)

	require.NoError(t, store.RemoveAllDenylist(channelID, channelType))
	denylist, err = store.GetDenylist(channelID, channelType)
	require.NoError(t, err)
	require.Empty(t, denylist)
}

func TestStoreChannelQueriesUseDedicatedChannelStateStore(t *testing.T) {
	channelState := &stubChannelStateStore{
		channel:       wkdb.ChannelInfo{ChannelId: "channel-1", ChannelType: 2},
		channelExists: true,
		subscribers:   []wkdb.Member{{Uid: "sub-1"}},
		denylist:      []wkdb.Member{{Uid: "deny-1"}},
		allowlist:     []wkdb.Member{{Uid: "allow-1"}},
		hasAllowlist:  true,
	}
	s := &Store{channelStateStore: channelState}

	channel, err := s.GetChannel("channel-1", 2)
	require.NoError(t, err)
	require.Equal(t, "channel-1", channel.ChannelId)
	require.Equal(t, "channel", channelState.lastLookupKind)

	exists, err := s.ExistChannel("channel-1", 2)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "channel_exists", channelState.lastLookupKind)

	subscribers, err := s.GetSubscribers("channel-1", 2)
	require.NoError(t, err)
	require.Equal(t, []wkdb.Member{{Uid: "sub-1"}}, subscribers)
	require.Equal(t, "subscribers", channelState.lastLookupKind)

	exists, err = s.ExistSubscriber("channel-1", 2, "sub-1")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "sub-1", channelState.lastSubscriberUID)
	require.Equal(t, "subscriber", channelState.lastLookupKind)

	denylist, err := s.GetDenylist("channel-1", 2)
	require.NoError(t, err)
	require.Equal(t, []wkdb.Member{{Uid: "deny-1"}}, denylist)
	require.Equal(t, "denylist", channelState.lastLookupKind)

	exists, err = s.ExistDenylist("channel-1", 2, "deny-1")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "deny-1", channelState.lastDenylistUID)
	require.Equal(t, "denylist_exists", channelState.lastLookupKind)

	allowlist, err := s.GetAllowlist("channel-1", 2)
	require.NoError(t, err)
	require.Equal(t, []wkdb.Member{{Uid: "allow-1"}}, allowlist)
	require.Equal(t, "allowlist", channelState.lastLookupKind)

	exists, err = s.ExistAllowlist("channel-1", 2, "allow-1")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "allow-1", channelState.lastAllowlistUID)
	require.Equal(t, "allowlist_exists", channelState.lastLookupKind)

	hasAllowlist, err := s.HasAllowlist("channel-1", 2)
	require.NoError(t, err)
	require.True(t, hasAllowlist)
	require.Equal(t, "has_allowlist", channelState.lastLookupKind)
}

func TestStoreDerivesChannelStateStoreFromHybridDB(t *testing.T) {
	hybrid, _ := newHybridMetaLocalTestDB(t)
	s := New(NewOptions(WithCompatDBRuntime(hybrid)))
	require.NotNil(t, s.channelStateStore)
}

func TestStoreChannelApplyHandlersUseDedicatedChannelStateStore(t *testing.T) {
	now := time.Unix(1710000000, 0)
	channelState := &stubChannelStateStore{}
	s := &Store{channelStateStore: channelState}

	addMembers := []wkdb.Member{{Uid: "sub-1", CreatedAt: &now, UpdatedAt: &now}}
	cmd := NewCMD(CMDAddSubscribers, EncodeMembers("channel-1", 2, addMembers))
	require.NoError(t, s.handleAddSubscribers(cmd))
	require.Equal(t, addMembers, channelState.lastAddedSubscribers)

	cmd = NewCMD(CMDRemoveSubscribers, EncodeChannelUids("channel-1", 2, []string{"sub-1"}))
	require.NoError(t, s.handleRemoveSubscribers(cmd))
	require.Equal(t, []string{"sub-1"}, channelState.lastRemovedSubscribers)

	channelInfo := wkdb.ChannelInfo{ChannelId: "channel-1", ChannelType: 2, Ban: true, CreatedAt: &now, UpdatedAt: &now}
	data, err := EncodeChannelInfo(channelInfo, CmdVersionChannelInfo)
	require.NoError(t, err)
	cmd = NewCMDWithVersion(CMDAddChannelInfo, data, CmdVersionChannelInfo)
	require.NoError(t, s.handleAddChannelInfo(cmd))
	require.Equal(t, channelInfo.ChannelId, channelState.lastAddedChannel.ChannelId)

	updatedChannel := wkdb.ChannelInfo{ChannelId: "channel-1", ChannelType: 2, Disband: true, CreatedAt: &now, UpdatedAt: &now}
	data, err = EncodeChannelInfo(updatedChannel, CmdVersionChannelInfo)
	require.NoError(t, err)
	cmd = NewCMDWithVersion(CMDUpdateChannelInfo, data, CmdVersionChannelInfo)
	require.NoError(t, s.handleUpdateChannel(cmd))
	require.Equal(t, updatedChannel.ChannelId, channelState.lastUpdatedChannel.ChannelId)
	require.True(t, channelState.lastUpdatedChannel.Disband)

	cmd = NewCMD(CMDRemoveAllSubscriber, EncodeChannel("channel-1", 2))
	require.NoError(t, s.handleRemoveAllSubscriber(cmd))
	require.True(t, channelState.removeAllSubscriber)

	cmd = NewCMD(CMDDeleteChannel, EncodeChannel("channel-1", 2))
	require.NoError(t, s.handleDeleteChannel(cmd))
	require.True(t, channelState.deleteChannelCalled)

	denyMembers := []wkdb.Member{{Uid: "deny-1", CreatedAt: &now, UpdatedAt: &now}}
	cmd = NewCMD(CMDAddDenylist, EncodeMembers("channel-1", 2, denyMembers))
	require.NoError(t, s.handleAddDenylist(cmd))
	require.Equal(t, denyMembers, channelState.lastAddedDenylist)

	cmd = NewCMD(CMDRemoveDenylist, EncodeChannelUids("channel-1", 2, []string{"deny-1"}))
	require.NoError(t, s.handleRemoveDenylist(cmd))
	require.Equal(t, []string{"deny-1"}, channelState.lastRemovedDenylist)

	cmd = NewCMD(CMDRemoveAllDenylist, EncodeChannel("channel-1", 2))
	require.NoError(t, s.handleRemoveAllDenylist(cmd))
	require.True(t, channelState.removeAllDenylist)

	allowMembers := []wkdb.Member{{Uid: "allow-1", CreatedAt: &now, UpdatedAt: &now}}
	cmd = NewCMD(CMDAddAllowlist, EncodeMembers("channel-1", 2, allowMembers))
	require.NoError(t, s.handleAddAllowlist(cmd))
	require.Equal(t, allowMembers, channelState.lastAddedAllowlist)

	cmd = NewCMD(CMDRemoveAllowlist, EncodeChannelUids("channel-1", 2, []string{"allow-1"}))
	require.NoError(t, s.handleRemoveAllowlist(cmd))
	require.Equal(t, []string{"allow-1"}, channelState.lastRemovedAllowlist)

	cmd = NewCMD(CMDRemoveAllAllowlist, EncodeChannel("channel-1", 2))
	require.NoError(t, s.handleRemoveAllAllowlist(cmd))
	require.True(t, channelState.removeAllAllowlist)
}
