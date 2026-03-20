package store

import (
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/stretchr/testify/require"
)

type stubConversationStore struct {
	conversationsByUID map[string][]wkdb.Conversation
	lastUID            string
	lastChannelID      string
	lastChannelType    uint8
	lastType           wkdb.ConversationType
	lastUpdatedAt      uint64
	lastExcludeTypes   []uint8
	lastLimit          int
	lastChannels       []wkdb.Channel
	lastSeq            uint64
	lastOp             string
	added              []wkdb.Conversation
	addedBatch         []wkdb.Conversation
	addedByUID         []wkdb.Conversation
}

var _ ConversationStore = (*stubConversationStore)(nil)

func (s *stubConversationStore) AddOrUpdateConversations(conversations []wkdb.Conversation) error {
	s.lastOp = "add_or_update"
	s.added = append([]wkdb.Conversation(nil), conversations...)
	return nil
}

func (s *stubConversationStore) AddOrUpdateConversationsBatchIfNotExist(conversations []wkdb.Conversation) error {
	s.lastOp = "add_or_update_if_not_exist"
	s.addedBatch = append([]wkdb.Conversation(nil), conversations...)
	return nil
}

func (s *stubConversationStore) AddOrUpdateConversationsWithUser(uid string, conversations []wkdb.Conversation) error {
	s.lastOp = "add_or_update_with_user"
	s.lastUID = uid
	s.addedByUID = append([]wkdb.Conversation(nil), conversations...)
	return nil
}

func (s *stubConversationStore) DeleteConversation(uid string, channelID string, channelType uint8) error {
	s.lastOp = "delete"
	s.lastUID = uid
	s.lastChannelID = channelID
	s.lastChannelType = channelType
	return nil
}

func (s *stubConversationStore) DeleteConversations(uid string, channels []wkdb.Channel) error {
	s.lastOp = "delete_batch"
	s.lastUID = uid
	s.lastChannels = append([]wkdb.Channel(nil), channels...)
	return nil
}

func (s *stubConversationStore) UpdateConversationDeletedAtMsgSeq(uid string, channelID string, channelType uint8, seq uint64) error {
	s.lastOp = "update_deleted_at"
	s.lastUID = uid
	s.lastChannelID = channelID
	s.lastChannelType = channelType
	s.lastSeq = seq
	return nil
}

func (s *stubConversationStore) UpdateConversationIfSeqGreater(uid string, channelID string, channelType uint8, seq uint64) error {
	s.lastOp = "update_if_seq_greater"
	s.lastUID = uid
	s.lastChannelID = channelID
	s.lastChannelType = channelType
	s.lastSeq = seq
	return nil
}

func (s *stubConversationStore) GetConversations(uid string) ([]wkdb.Conversation, error) {
	s.lastOp = "get_conversations"
	s.lastUID = uid
	return append([]wkdb.Conversation(nil), s.conversationsByUID[uid]...), nil
}

func (s *stubConversationStore) GetConversationsByType(uid string, tp wkdb.ConversationType) ([]wkdb.Conversation, error) {
	s.lastOp = "get_conversations_by_type"
	s.lastUID = uid
	s.lastType = tp
	conversations := s.conversationsByUID[uid]
	filtered := make([]wkdb.Conversation, 0, len(conversations))
	for _, conversation := range conversations {
		if conversation.Type == tp {
			filtered = append(filtered, conversation)
		}
	}
	return filtered, nil
}

func (s *stubConversationStore) GetConversation(uid string, channelID string, channelType uint8) (wkdb.Conversation, error) {
	s.lastOp = "get_conversation"
	s.lastUID = uid
	s.lastChannelID = channelID
	s.lastChannelType = channelType
	for _, conversation := range s.conversationsByUID[uid] {
		if conversation.ChannelId == channelID && conversation.ChannelType == channelType {
			return conversation, nil
		}
	}
	return wkdb.Conversation{}, wkdb.ErrNotFound
}

func (s *stubConversationStore) GetLastConversations(uid string, tp wkdb.ConversationType, updatedAt uint64, excludeChannelTypes []uint8, limit int) ([]wkdb.Conversation, error) {
	s.lastOp = "get_last_conversations"
	s.lastUID = uid
	s.lastType = tp
	s.lastUpdatedAt = updatedAt
	s.lastExcludeTypes = append([]uint8(nil), excludeChannelTypes...)
	s.lastLimit = limit
	return append([]wkdb.Conversation(nil), s.conversationsByUID[uid]...), nil
}

func (s *stubConversationStore) GetChannelConversationLocalUsers(channelID string, channelType uint8) ([]string, error) {
	s.lastOp = "get_local_users"
	s.lastChannelID = channelID
	s.lastChannelType = channelType
	uidSet := make(map[string]struct{})
	for uid, conversations := range s.conversationsByUID {
		for _, conversation := range conversations {
			if conversation.ChannelId == channelID && conversation.ChannelType == channelType {
				uidSet[uid] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(uidSet))
	for uid := range uidSet {
		out = append(out, uid)
	}
	return out, nil
}

func (s *stubConversationStore) ExistConversation(uid string, channelID string, channelType uint8) (bool, error) {
	s.lastOp = "exist"
	s.lastUID = uid
	s.lastChannelID = channelID
	s.lastChannelType = channelType
	for _, conversation := range s.conversationsByUID[uid] {
		if conversation.ChannelId == channelID && conversation.ChannelType == channelType {
			return true, nil
		}
	}
	return false, nil
}

func TestStoreDerivesConversationStoreFromHybridDB(t *testing.T) {
	hybrid, _ := newHybridMetaLocalTestDB(t)
	now := time.Unix(1710000000, 0)

	require.NoError(t, hybrid.slotDB.Slots().Scope(0).Conversations().Put("user-1", []wkdb.Conversation{
		{
			Id:          1,
			Uid:         "user-1",
			Type:        wkdb.ConversationTypeChat,
			ChannelId:   "channel-1",
			ChannelType: 2,
			CreatedAt:   &now,
			UpdatedAt:   &now,
		},
	}))

	s := New(NewOptions(WithCompatDBRuntime(hybrid)))
	require.NotNil(t, s.conversationStore)

	conversations, err := s.GetConversations("user-1")
	require.NoError(t, err)
	require.Len(t, conversations, 1)
	require.Equal(t, "channel-1", conversations[0].ChannelId)

	conversation, err := s.GetConversation("user-1", "channel-1", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(1), conversation.Id)

	uids, err := s.conversationStore.GetChannelConversationLocalUsers("channel-1", 2)
	require.NoError(t, err)
	require.Equal(t, []string{"user-1"}, uids)
}

func TestStoreConversationReadsUseConversationStore(t *testing.T) {
	now := time.Unix(1710000000, 0)
	stub := &stubConversationStore{
		conversationsByUID: map[string][]wkdb.Conversation{
			"user-1": {
				{
					Id:          1,
					Uid:         "user-1",
					Type:        wkdb.ConversationTypeChat,
					ChannelId:   "channel-1",
					ChannelType: 2,
					CreatedAt:   &now,
					UpdatedAt:   &now,
				},
			},
		},
	}
	s := &Store{conversationStore: stub}

	conversations, err := s.GetConversations("user-1")
	require.NoError(t, err)
	require.Len(t, conversations, 1)
	require.Equal(t, "get_conversations", stub.lastOp)
	require.Equal(t, "user-1", stub.lastUID)

	conversation, err := s.GetConversation("user-1", "channel-1", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(1), conversation.Id)
	require.Equal(t, "get_conversation", stub.lastOp)

	_, err = s.GetConversationsByType("user-1", wkdb.ConversationTypeChat)
	require.NoError(t, err)
	require.Equal(t, "get_conversations_by_type", stub.lastOp)
	require.Equal(t, wkdb.ConversationTypeChat, stub.lastType)

	_, err = s.GetLastConversations("user-1", wkdb.ConversationTypeChat, 123, []uint8{3}, 10)
	require.NoError(t, err)
	require.Equal(t, "get_last_conversations", stub.lastOp)
	require.Equal(t, uint64(123), stub.lastUpdatedAt)
	require.Equal(t, []uint8{3}, stub.lastExcludeTypes)
	require.Equal(t, 10, stub.lastLimit)

	uids, err := s.GetChannelConversationLocalUsers("channel-1", 2)
	require.NoError(t, err)
	require.Equal(t, []string{"user-1"}, uids)
	require.Equal(t, "get_local_users", stub.lastOp)
}

func TestStoreApplyConversationHandlersUseConversationStore(t *testing.T) {
	now := time.Unix(1710000000, 0)
	conversation := wkdb.Conversation{
		Id:          1,
		Uid:         "user-1",
		Type:        wkdb.ConversationTypeChat,
		ChannelId:   "channel-1",
		ChannelType: 2,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}

	stub := &stubConversationStore{}
	s := &Store{
		conversationStore: stub,
		opts: &Options{
			IsCmdChannel: func(string) bool { return false },
		},
	}

	userConversationData, err := EncodeCMDAddOrUpdateUserConversations("user-1", []wkdb.Conversation{conversation})
	require.NoError(t, err)
	require.NoError(t, s.handleAddOrUpdateUserConversations(NewCMD(CMDAddOrUpdateUserConversations, userConversationData)))
	require.Equal(t, "add_or_update_with_user", stub.lastOp)
	require.Equal(t, "user-1", stub.lastUID)

	require.NoError(t, s.handleDeleteConversation(NewCMD(CMDDeleteConversation, EncodeCMDDeleteConversation("user-1", "channel-1", 2))))
	require.Equal(t, "delete", stub.lastOp)
	require.Equal(t, "channel-1", stub.lastChannelID)

	require.NoError(t, s.handleDeleteConversations(NewCMD(CMDDeleteConversations, EncodeCMDDeleteConversations("user-1", []wkdb.Channel{{ChannelId: "channel-1", ChannelType: 2}}))))
	require.Equal(t, "delete_batch", stub.lastOp)
	require.Len(t, stub.lastChannels, 1)

	conversationData, err := EncodeCMDAddOrUpdateConversations(wkdb.ConversationSet{conversation})
	require.NoError(t, err)
	require.NoError(t, s.handleAddOrUpdateConversations(NewCMD(CMDAddOrUpdateConversations, conversationData)))
	require.Equal(t, "add_or_update", stub.lastOp)
	require.Len(t, stub.added, 1)

	require.NoError(t, s.handleAddOrUpdateConversationsBatchIfNotExist(NewCMD(CMDAddOrUpdateConversationsBatchIfNotExist, conversationData)))
	require.Equal(t, "add_or_update_if_not_exist", stub.lastOp)
	require.Len(t, stub.addedBatch, 1)

	require.NoError(t, s.handleUpdateConversationDeletedAtMsgSeq(NewCMD(CMDUpdateConversationDeletedAtMsgSeq, EncodeCMDUpdateConversationDeletedAtMsgSeq("user-1", "channel-1", 2, 99))))
	require.Equal(t, "update_deleted_at", stub.lastOp)
	require.Equal(t, uint64(99), stub.lastSeq)

	require.NoError(t, s.handleUpdateConversationIfSeqGreater(NewCMD(CMDUpdateConversationIfSeqGreater, EncodeCMDUpdateConversationIfSeqGreater("user-1", "channel-1", 2, 101))))
	require.Equal(t, "update_if_seq_greater", stub.lastOp)
	require.Equal(t, uint64(101), stub.lastSeq)

	require.NoError(t, s.handleBatchUpdateConversation(NewCMD(CMDBatchUpdateConversation, EncodeCMDBatchUpdateConversation([]*wkdb.BatchUpdateConversationModel{{
		ChannelId:   "channel-2",
		ChannelType: 2,
		Uids: map[string]uint64{
			"user-1": 88,
		},
	}}))))
	require.Equal(t, "add_or_update_with_user", stub.lastOp)
	require.Equal(t, "user-1", stub.lastUID)
	require.Len(t, stub.addedByUID, 1)
	require.Equal(t, uint64(88), stub.addedByUID[0].ReadToMsgSeq)
}
