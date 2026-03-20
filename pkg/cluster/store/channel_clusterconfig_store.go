package store

import (
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
)

type ChannelClusterConfigStore interface {
	SaveChannelClusterConfigs(cfgs []wkdb.ChannelClusterConfig) error
	GetChannelClusterConfig(channelID string, channelType uint8) (wkdb.ChannelClusterConfig, error)
	GetChannelClusterConfigWithSlotId(slotID uint32) ([]wkdb.ChannelClusterConfig, error)
	GetChannelClusterConfigVersion(channelID string, channelType uint8) (uint64, error)
}

type SlotChannelClusterConfigStore struct {
	db        wkdbv3.DB
	slotCount uint32
	routeSlot func(key string) uint32
}

var _ ChannelClusterConfigStore = (*SlotChannelClusterConfigStore)(nil)

func NewSlotChannelClusterConfigStore(db wkdbv3.DB, slotCount uint32, routeSlot func(key string) uint32) *SlotChannelClusterConfigStore {
	if db == nil || slotCount == 0 || routeSlot == nil {
		return nil
	}
	return &SlotChannelClusterConfigStore{
		db:        db,
		slotCount: slotCount,
		routeSlot: routeSlot,
	}
}

func (s *SlotChannelClusterConfigStore) slotScopeByChannel(channelID string) wkdbv3.SlotScope {
	return s.db.Slots().Scope(s.routeSlot(channelID))
}

func (s *SlotChannelClusterConfigStore) SaveChannelClusterConfigs(cfgs []wkdb.ChannelClusterConfig) error {
	for _, cfg := range cfgs {
		if err := s.slotScopeByChannel(cfg.ChannelId).ChannelClusterConfigs().Put(cfg); err != nil {
			return err
		}
	}
	return nil
}

func (s *SlotChannelClusterConfigStore) GetChannelClusterConfig(channelID string, channelType uint8) (wkdb.ChannelClusterConfig, error) {
	return s.slotScopeByChannel(channelID).ChannelClusterConfigs().Get(channelID, channelType)
}

func (s *SlotChannelClusterConfigStore) GetChannelClusterConfigWithSlotId(slotID uint32) ([]wkdb.ChannelClusterConfig, error) {
	return s.db.Slots().Scope(slotID).ChannelClusterConfigs().Search(wkdb.ChannelClusterConfigSearchReq{Limit: 0})
}

func (s *SlotChannelClusterConfigStore) GetChannelClusterConfigVersion(channelID string, channelType uint8) (uint64, error) {
	cfg, err := s.GetChannelClusterConfig(channelID, channelType)
	if err != nil {
		return 0, err
	}
	return cfg.ConfVersion, nil
}
