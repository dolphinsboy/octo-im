package store

import (
	"sort"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
)

type ConversationStore interface {
	AddOrUpdateConversations(conversations []wkdb.Conversation) error
	AddOrUpdateConversationsBatchIfNotExist(conversations []wkdb.Conversation) error
	AddOrUpdateConversationsWithUser(uid string, conversations []wkdb.Conversation) error
	DeleteConversation(uid string, channelID string, channelType uint8) error
	DeleteConversations(uid string, channels []wkdb.Channel) error
	UpdateConversationDeletedAtMsgSeq(uid string, channelID string, channelType uint8, seq uint64) error
	UpdateConversationIfSeqGreater(uid string, channelID string, channelType uint8, seq uint64) error
	GetConversations(uid string) ([]wkdb.Conversation, error)
	GetConversationsByType(uid string, tp wkdb.ConversationType) ([]wkdb.Conversation, error)
	GetConversation(uid string, channelID string, channelType uint8) (wkdb.Conversation, error)
	GetLastConversations(uid string, tp wkdb.ConversationType, updatedAt uint64, excludeChannelTypes []uint8, limit int) ([]wkdb.Conversation, error)
	GetChannelConversationLocalUsers(channelID string, channelType uint8) ([]string, error)
	ExistConversation(uid string, channelID string, channelType uint8) (bool, error)
}

type SlotConversationStore struct {
	db             wkdbv3.DB
	slotCount      uint32
	routeSlot      func(key string) uint32
	nextPrimaryKey func() uint64
}

var _ ConversationStore = (*SlotConversationStore)(nil)

func NewSlotConversationStore(db wkdbv3.DB, slotCount uint32, routeSlot func(key string) uint32, nextPrimaryKey func() uint64) *SlotConversationStore {
	if db == nil || routeSlot == nil || slotCount == 0 {
		return nil
	}
	return &SlotConversationStore{
		db:             db,
		slotCount:      slotCount,
		routeSlot:      routeSlot,
		nextPrimaryKey: nextPrimaryKey,
	}
}

func (s *SlotConversationStore) slotScopeByUID(uid string) wkdbv3.SlotScope {
	return s.db.Slots().Scope(s.routeSlot(uid))
}

func (s *SlotConversationStore) eachSlot(fn func(slotID uint32, scope wkdbv3.SlotScope) error) error {
	for slotID := uint32(0); slotID < s.slotCount; slotID++ {
		if err := fn(slotID, s.db.Slots().Scope(slotID)); err != nil {
			return err
		}
	}
	return nil
}

func (s *SlotConversationStore) AddOrUpdateConversations(conversations []wkdb.Conversation) error {
	if len(conversations) == 0 {
		return nil
	}
	grouped := make(map[string][]wkdb.Conversation)
	for _, conversation := range conversations {
		grouped[conversation.Uid] = append(grouped[conversation.Uid], s.normalizeConversation(conversation))
	}
	for uid, items := range grouped {
		if err := s.slotScopeByUID(uid).Conversations().Put(uid, items); err != nil {
			return err
		}
	}
	return nil
}

func (s *SlotConversationStore) AddOrUpdateConversationsBatchIfNotExist(conversations []wkdb.Conversation) error {
	if len(conversations) == 0 {
		return nil
	}
	grouped := make(map[string][]wkdb.Conversation)
	for _, conversation := range conversations {
		exists, err := s.ExistConversation(conversation.Uid, conversation.ChannelId, conversation.ChannelType)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		grouped[conversation.Uid] = append(grouped[conversation.Uid], s.normalizeConversation(conversation))
	}
	for uid, items := range grouped {
		if err := s.slotScopeByUID(uid).Conversations().Put(uid, items); err != nil {
			return err
		}
	}
	return nil
}

func (s *SlotConversationStore) AddOrUpdateConversationsWithUser(uid string, conversations []wkdb.Conversation) error {
	normalized := make([]wkdb.Conversation, 0, len(conversations))
	for _, conversation := range conversations {
		if conversation.Uid == "" {
			conversation.Uid = uid
		}
		normalized = append(normalized, s.normalizeConversation(conversation))
	}
	return s.slotScopeByUID(uid).Conversations().Put(uid, normalized)
}

func (s *SlotConversationStore) DeleteConversation(uid string, channelID string, channelType uint8) error {
	return s.slotScopeByUID(uid).Conversations().Delete(uid, channelID, channelType)
}

func (s *SlotConversationStore) DeleteConversations(uid string, channels []wkdb.Channel) error {
	return s.slotScopeByUID(uid).Conversations().DeleteBatch(uid, channels)
}

func (s *SlotConversationStore) UpdateConversationDeletedAtMsgSeq(uid string, channelID string, channelType uint8, seq uint64) error {
	return s.slotScopeByUID(uid).Conversations().UpdateDeletedAt(uid, channelID, channelType, seq)
}

func (s *SlotConversationStore) UpdateConversationIfSeqGreater(uid string, channelID string, channelType uint8, seq uint64) error {
	return s.slotScopeByUID(uid).Conversations().UpdateIfSeqGreater(uid, channelID, channelType, seq)
}

func (s *SlotConversationStore) GetConversations(uid string) ([]wkdb.Conversation, error) {
	conversations, err := s.slotScopeByUID(uid).Conversations().ListByUID(uid)
	if err != nil {
		return nil, err
	}
	out := make([]wkdb.Conversation, 0, len(conversations))
	for _, conversation := range conversations {
		out = append(out, cloneConversation(conversation))
	}
	return out, nil
}

func (s *SlotConversationStore) GetConversationsByType(uid string, tp wkdb.ConversationType) ([]wkdb.Conversation, error) {
	conversations, err := s.GetConversations(uid)
	if err != nil {
		return nil, err
	}
	filtered := make([]wkdb.Conversation, 0, len(conversations))
	for _, conversation := range conversations {
		if conversation.Type != tp {
			continue
		}
		filtered = append(filtered, cloneConversation(conversation))
	}
	return filtered, nil
}

func (s *SlotConversationStore) GetConversation(uid string, channelID string, channelType uint8) (wkdb.Conversation, error) {
	conversation, err := s.slotScopeByUID(uid).Conversations().Get(uid, channelID, channelType)
	if err != nil {
		return wkdb.Conversation{}, err
	}
	return cloneConversation(conversation), nil
}

func (s *SlotConversationStore) GetLastConversations(uid string, tp wkdb.ConversationType, updatedAt uint64, excludeChannelTypes []uint8, limit int) ([]wkdb.Conversation, error) {
	conversations, err := s.GetConversations(uid)
	if err != nil {
		return nil, err
	}
	excluded := make(map[uint8]struct{}, len(excludeChannelTypes))
	for _, channelType := range excludeChannelTypes {
		excluded[channelType] = struct{}{}
	}
	filtered := make([]wkdb.Conversation, 0, len(conversations))
	for _, conversation := range conversations {
		if conversation.Type != tp {
			continue
		}
		if _, ok := excluded[conversation.ChannelType]; ok {
			continue
		}
		if updatedAt > 0 && uint64(conversationSortTime(conversation)) >= updatedAt {
			continue
		}
		filtered = append(filtered, cloneConversation(conversation))
	}
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

func (s *SlotConversationStore) GetChannelConversationLocalUsers(channelID string, channelType uint8) ([]string, error) {
	uidSet := make(map[string]struct{})
	if err := s.eachSlot(func(_ uint32, scope wkdbv3.SlotScope) error {
		conversations, err := scope.Conversations().Search(wkdb.ConversationSearchReq{Limit: 0})
		if err != nil {
			return err
		}
		for _, conversation := range conversations {
			if conversation.ChannelId != channelID || conversation.ChannelType != channelType {
				continue
			}
			uidSet[conversation.Uid] = struct{}{}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	uids := make([]string, 0, len(uidSet))
	for uid := range uidSet {
		uids = append(uids, uid)
	}
	sort.Strings(uids)
	return uids, nil
}

func (s *SlotConversationStore) ExistConversation(uid string, channelID string, channelType uint8) (bool, error) {
	_, err := s.slotScopeByUID(uid).Conversations().Get(uid, channelID, channelType)
	if err == nil {
		return true, nil
	}
	if err == wkdb.ErrNotFound {
		return false, nil
	}
	return false, err
}

func (s *SlotConversationStore) normalizeConversation(conversation wkdb.Conversation) wkdb.Conversation {
	if conversation.Id == 0 && s.nextPrimaryKey != nil {
		conversation.Id = s.nextPrimaryKey()
	}
	return conversation
}
