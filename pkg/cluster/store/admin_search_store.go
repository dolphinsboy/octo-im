package store

import (
	"fmt"
	"sort"
	"strings"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
)

type AdminSearchStore interface {
	SearchUsers(req wkdb.UserSearchReq) ([]wkdb.User, error)
	CountUsers() (int, error)
	SearchDevices(req wkdb.DeviceSearchReq) ([]wkdb.Device, error)
	CountDevices() (int, error)
	SearchChannels(req wkdb.ChannelSearchReq) ([]wkdb.ChannelInfo, error)
	SearchConversations(req wkdb.ConversationSearchReq) ([]wkdb.Conversation, error)
	CountConversations() (int, error)
	SearchChannelClusterConfigs(req wkdb.ChannelClusterConfigSearchReq, filter ...func(cfg wkdb.ChannelClusterConfig) bool) ([]wkdb.ChannelClusterConfig, error)
}

type SlotAdminSearchStore struct {
	db        wkdbv3.DB
	slotCount uint32
	routeSlot func(key string) uint32
}

func NewSlotAdminSearchStore(db wkdbv3.DB, slotCount uint32, routeSlot func(key string) uint32) *SlotAdminSearchStore {
	if db == nil || slotCount == 0 || routeSlot == nil {
		return nil
	}
	return &SlotAdminSearchStore{
		db:        db,
		slotCount: slotCount,
		routeSlot: routeSlot,
	}
}

func (s *SlotAdminSearchStore) slotIDForKey(key string) uint32 {
	return s.routeSlot(key)
}

func (s *SlotAdminSearchStore) slotScopeByKey(key string) wkdbv3.SlotScope {
	return s.db.Slots().Scope(s.slotIDForKey(key))
}

func (s *SlotAdminSearchStore) slotScope(slotID uint32) wkdbv3.SlotScope {
	return s.db.Slots().Scope(slotID)
}

func (s *SlotAdminSearchStore) eachSlot(fn func(slotID uint32, scope wkdbv3.SlotScope) error) error {
	if s == nil || s.db == nil || s.routeSlot == nil || s.slotCount == 0 {
		return fmt.Errorf("slot admin search store is not configured")
	}
	for slotID := uint32(0); slotID < s.slotCount; slotID++ {
		if err := fn(slotID, s.slotScope(slotID)); err != nil {
			return err
		}
	}
	return nil
}

func (s *SlotAdminSearchStore) SearchUsers(req wkdb.UserSearchReq) ([]wkdb.User, error) {
	if strings.TrimSpace(req.Uid) != "" {
		return s.slotScopeByKey(req.Uid).Users().Search(req)
	}
	localReq := req
	localReq.Limit = 0
	users := make([]wkdb.User, 0)
	if err := s.eachSlot(func(_ uint32, scope wkdbv3.SlotScope) error {
		items, err := scope.Users().Search(localReq)
		if err != nil {
			return err
		}
		users = append(users, items...)
		return nil
	}); err != nil {
		return nil, err
	}
	sort.SliceStable(users, func(i, j int) bool {
		left := userSortTime(users[i])
		right := userSortTime(users[j])
		if left == right {
			return strings.Compare(users[i].Uid, users[j].Uid) < 0
		}
		return left > right
	})
	return limitUsers(users, req.Limit, req.Pre), nil
}

func (s *SlotAdminSearchStore) CountUsers() (int, error) {
	users, err := s.SearchUsers(wkdb.UserSearchReq{Limit: 0})
	if err != nil {
		return 0, err
	}
	return len(users), nil
}

func (s *SlotAdminSearchStore) SearchDevices(req wkdb.DeviceSearchReq) ([]wkdb.Device, error) {
	if strings.TrimSpace(req.Uid) != "" {
		return s.slotScopeByKey(req.Uid).Devices().Search(req)
	}
	localReq := req
	localReq.Limit = 0
	devices := make([]wkdb.Device, 0)
	if err := s.eachSlot(func(_ uint32, scope wkdbv3.SlotScope) error {
		items, err := scope.Devices().Search(localReq)
		if err != nil {
			return err
		}
		devices = append(devices, items...)
		return nil
	}); err != nil {
		return nil, err
	}
	sortDevices(devices)
	return limitDevices(devices, req.Limit, req.Pre), nil
}

func (s *SlotAdminSearchStore) CountDevices() (int, error) {
	devices, err := s.SearchDevices(wkdb.DeviceSearchReq{Limit: 0})
	if err != nil {
		return 0, err
	}
	return len(devices), nil
}

func (s *SlotAdminSearchStore) SearchChannels(req wkdb.ChannelSearchReq) ([]wkdb.ChannelInfo, error) {
	if strings.TrimSpace(req.ChannelId) != "" {
		return s.slotScopeByKey(req.ChannelId).Channels().Search(req)
	}
	localReq := req
	localReq.Limit = 0
	channels := make([]wkdb.ChannelInfo, 0)
	if err := s.eachSlot(func(_ uint32, scope wkdbv3.SlotScope) error {
		items, err := scope.Channels().Search(localReq)
		if err != nil {
			return err
		}
		channels = append(channels, items...)
		return nil
	}); err != nil {
		return nil, err
	}
	sort.SliceStable(channels, func(i, j int) bool {
		left := channelSortTime(channels[i])
		right := channelSortTime(channels[j])
		if left == right {
			return strings.Compare(wkdb.ChannelToKey(channels[i].ChannelId, channels[i].ChannelType), wkdb.ChannelToKey(channels[j].ChannelId, channels[j].ChannelType)) < 0
		}
		return left > right
	})
	return limitChannelInfos(channels, req.Limit, req.Pre), nil
}

func (s *SlotAdminSearchStore) SearchConversations(req wkdb.ConversationSearchReq) ([]wkdb.Conversation, error) {
	if strings.TrimSpace(req.Uid) != "" {
		return s.slotScopeByKey(req.Uid).Conversations().Search(req)
	}
	localReq := req
	localReq.Limit = 0
	localReq.CurrentPage = 0
	conversations := make([]wkdb.Conversation, 0)
	if err := s.eachSlot(func(_ uint32, scope wkdbv3.SlotScope) error {
		items, err := scope.Conversations().Search(localReq)
		if err != nil {
			return err
		}
		conversations = append(conversations, items...)
		return nil
	}); err != nil {
		return nil, err
	}
	sortConversations(conversations)
	return paginateConversations(conversations, req.CurrentPage, req.Limit), nil
}

func (s *SlotAdminSearchStore) CountConversations() (int, error) {
	conversations, err := s.SearchConversations(wkdb.ConversationSearchReq{Limit: 0})
	if err != nil {
		return 0, err
	}
	return len(conversations), nil
}

func (s *SlotAdminSearchStore) SearchChannelClusterConfigs(req wkdb.ChannelClusterConfigSearchReq, filter ...func(cfg wkdb.ChannelClusterConfig) bool) ([]wkdb.ChannelClusterConfig, error) {
	applyFilters := func(cfgs []wkdb.ChannelClusterConfig) []wkdb.ChannelClusterConfig {
		if len(filter) == 0 {
			return cfgs
		}
		filtered := make([]wkdb.ChannelClusterConfig, 0, len(cfgs))
		for _, cfg := range cfgs {
			matched := true
			for _, fn := range filter {
				if fn == nil {
					continue
				}
				if !fn(cfg) {
					matched = false
					break
				}
			}
			if matched {
				filtered = append(filtered, cfg)
			}
		}
		return filtered
	}
	if strings.TrimSpace(req.ChannelId) != "" {
		cfgs, err := s.slotScopeByKey(req.ChannelId).ChannelClusterConfigs().Search(req)
		if err != nil {
			return nil, err
		}
		return applyFilters(cfgs), nil
	}
	localReq := req
	localReq.Limit = 0
	cfgs := make([]wkdb.ChannelClusterConfig, 0)
	if err := s.eachSlot(func(_ uint32, scope wkdbv3.SlotScope) error {
		items, err := scope.ChannelClusterConfigs().Search(localReq)
		if err != nil {
			return err
		}
		cfgs = append(cfgs, items...)
		return nil
	}); err != nil {
		return nil, err
	}
	cfgs = applyFilters(cfgs)
	sort.SliceStable(cfgs, func(i, j int) bool {
		left := channelClusterConfigSortTime(cfgs[i])
		right := channelClusterConfigSortTime(cfgs[j])
		if left == right {
			return strings.Compare(wkdb.ChannelToKey(cfgs[i].ChannelId, cfgs[i].ChannelType), wkdb.ChannelToKey(cfgs[j].ChannelId, cfgs[j].ChannelType)) < 0
		}
		return left > right
	})
	return limitChannelClusterConfigs(cfgs, req.Limit, req.Pre), nil
}
