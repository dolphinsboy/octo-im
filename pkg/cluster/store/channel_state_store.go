package store

import (
	"strings"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
)

type ChannelStateStore interface {
	AddSubscribers(channelId string, channelType uint8, members []wkdb.Member) error
	RemoveSubscribers(channelId string, channelType uint8, uids []string) error
	ExistSubscriber(channelId string, channelType uint8, uid string) (bool, error)
	RemoveAllSubscriber(channelId string, channelType uint8) error
	GetSubscribers(channelId string, channelType uint8) ([]wkdb.Member, error)
	AddChannel(channelInfo wkdb.ChannelInfo) (uint64, error)
	GetChannel(channelId string, channelType uint8) (wkdb.ChannelInfo, error)
	UpdateChannel(channelInfo wkdb.ChannelInfo) error
	ExistChannel(channelId string, channelType uint8) (bool, error)
	DeleteChannel(channelId string, channelType uint8) error
	AddDenylist(channelId string, channelType uint8, members []wkdb.Member) error
	GetDenylist(channelId string, channelType uint8) ([]wkdb.Member, error)
	RemoveDenylist(channelId string, channelType uint8, uids []string) error
	RemoveAllDenylist(channelId string, channelType uint8) error
	ExistDenylist(channelId string, channelType uint8, uid string) (bool, error)
	AddAllowlist(channelId string, channelType uint8, members []wkdb.Member) error
	GetAllowlist(channelId string, channelType uint8) ([]wkdb.Member, error)
	RemoveAllowlist(channelId string, channelType uint8, uids []string) error
	RemoveAllAllowlist(channelId string, channelType uint8) error
	ExistAllowlist(channelId string, channelType uint8, uid string) (bool, error)
	HasAllowlist(channelId string, channelType uint8) (bool, error)
}

type SlotChannelStateStore struct {
	db        wkdbv3.DB
	routeSlot func(key string) uint32
}

func NewSlotChannelStateStore(db wkdbv3.DB, routeSlot func(key string) uint32) *SlotChannelStateStore {
	if db == nil || routeSlot == nil {
		return nil
	}
	return &SlotChannelStateStore{
		db:        db,
		routeSlot: routeSlot,
	}
}

func (s *SlotChannelStateStore) slotScopeByChannel(channelId string) wkdbv3.SlotScope {
	return s.db.Slots().Scope(s.routeSlot(channelId))
}

func (s *SlotChannelStateStore) AddSubscribers(channelId string, channelType uint8, members []wkdb.Member) error {
	return s.slotScopeByChannel(channelId).Subscribers().Put(channelId, channelType, members)
}

func (s *SlotChannelStateStore) RemoveSubscribers(channelId string, channelType uint8, uids []string) error {
	return s.slotScopeByChannel(channelId).Subscribers().Delete(channelId, channelType, uids)
}

func (s *SlotChannelStateStore) ExistSubscriber(channelId string, channelType uint8, uid string) (bool, error) {
	return s.memberExists(s.GetSubscribers, channelId, channelType, uid)
}

func (s *SlotChannelStateStore) RemoveAllSubscriber(channelId string, channelType uint8) error {
	return s.slotScopeByChannel(channelId).Subscribers().DeleteAll(channelId, channelType)
}

func (s *SlotChannelStateStore) GetSubscribers(channelId string, channelType uint8) ([]wkdb.Member, error) {
	return s.slotScopeByChannel(channelId).Subscribers().List(channelId, channelType)
}

func (s *SlotChannelStateStore) AddChannel(channelInfo wkdb.ChannelInfo) (uint64, error) {
	if err := s.slotScopeByChannel(channelInfo.ChannelId).Channels().Put(channelInfo); err != nil {
		return 0, err
	}
	return channelInfo.Id, nil
}

func (s *SlotChannelStateStore) GetChannel(channelId string, channelType uint8) (wkdb.ChannelInfo, error) {
	return s.slotScopeByChannel(channelId).Channels().Get(channelId, channelType)
}

func (s *SlotChannelStateStore) UpdateChannel(channelInfo wkdb.ChannelInfo) error {
	return s.slotScopeByChannel(channelInfo.ChannelId).Channels().Put(channelInfo)
}

func (s *SlotChannelStateStore) ExistChannel(channelId string, channelType uint8) (bool, error) {
	return s.slotScopeByChannel(channelId).Channels().Exists(channelId, channelType)
}

func (s *SlotChannelStateStore) DeleteChannel(channelId string, channelType uint8) error {
	return s.slotScopeByChannel(channelId).Channels().Delete(channelId, channelType)
}

func (s *SlotChannelStateStore) AddDenylist(channelId string, channelType uint8, members []wkdb.Member) error {
	return s.slotScopeByChannel(channelId).Denylists().Put(channelId, channelType, members)
}

func (s *SlotChannelStateStore) GetDenylist(channelId string, channelType uint8) ([]wkdb.Member, error) {
	return s.slotScopeByChannel(channelId).Denylists().List(channelId, channelType)
}

func (s *SlotChannelStateStore) RemoveDenylist(channelId string, channelType uint8, uids []string) error {
	return s.slotScopeByChannel(channelId).Denylists().Delete(channelId, channelType, uids)
}

func (s *SlotChannelStateStore) RemoveAllDenylist(channelId string, channelType uint8) error {
	return s.slotScopeByChannel(channelId).Denylists().DeleteAll(channelId, channelType)
}

func (s *SlotChannelStateStore) ExistDenylist(channelId string, channelType uint8, uid string) (bool, error) {
	return s.memberExists(s.GetDenylist, channelId, channelType, uid)
}

func (s *SlotChannelStateStore) AddAllowlist(channelId string, channelType uint8, members []wkdb.Member) error {
	return s.slotScopeByChannel(channelId).Allowlists().Put(channelId, channelType, members)
}

func (s *SlotChannelStateStore) GetAllowlist(channelId string, channelType uint8) ([]wkdb.Member, error) {
	return s.slotScopeByChannel(channelId).Allowlists().List(channelId, channelType)
}

func (s *SlotChannelStateStore) RemoveAllowlist(channelId string, channelType uint8, uids []string) error {
	return s.slotScopeByChannel(channelId).Allowlists().Delete(channelId, channelType, uids)
}

func (s *SlotChannelStateStore) RemoveAllAllowlist(channelId string, channelType uint8) error {
	return s.slotScopeByChannel(channelId).Allowlists().DeleteAll(channelId, channelType)
}

func (s *SlotChannelStateStore) ExistAllowlist(channelId string, channelType uint8, uid string) (bool, error) {
	return s.memberExists(s.GetAllowlist, channelId, channelType, uid)
}

func (s *SlotChannelStateStore) HasAllowlist(channelId string, channelType uint8) (bool, error) {
	members, err := s.GetAllowlist(channelId, channelType)
	if err != nil {
		return false, err
	}
	return len(members) > 0, nil
}

func (s *SlotChannelStateStore) memberExists(listFn func(channelId string, channelType uint8) ([]wkdb.Member, error), channelId string, channelType uint8, uid string) (bool, error) {
	if strings.TrimSpace(uid) == "" {
		return false, nil
	}
	members, err := listFn(channelId, channelType)
	if err != nil {
		return false, err
	}
	for _, member := range members {
		if member.Uid == uid {
			return true, nil
		}
	}
	return false, nil
}
