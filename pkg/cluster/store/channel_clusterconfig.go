package store

import (
	"fmt"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
)

func (s *Store) requireChannelClusterConfigStore() (ChannelClusterConfigStore, error) {
	if s.channelClusterConfigStore == nil {
		return nil, fmt.Errorf("channel cluster config store is not configured")
	}
	return s.channelClusterConfigStore, nil
}

func (s *Store) SaveChannelClusterConfig(cfg wkdb.ChannelClusterConfig) (version uint64, err error) {
	cfgData, err := cfg.Marshal()
	if err != nil {
		return 0, err
	}

	data, err := EncodeCMDChannelClusterConfigSave(cfg.ChannelId, cfg.ChannelType, cfgData)
	if err != nil {
		return 0, err
	}
	cmd := NewCMD(CMDChannelClusterConfigSave, data)
	cmdData, err := cmd.Marshal()
	if err != nil {
		return 0, err
	}
	slotId := s.opts.Slot.GetSlotId(cfg.ChannelId)
	resp, err := s.opts.Slot.ProposeUntilApplied(slotId, cmdData)
	if err != nil {
		return 0, err
	}
	return resp.Index, nil
}

func (s *Store) GetChannelClusterConfig(channelID string, channelType uint8) (wkdb.ChannelClusterConfig, error) {
	channelClusterConfigStore, err := s.requireChannelClusterConfigStore()
	if err != nil {
		return wkdb.ChannelClusterConfig{}, err
	}
	return channelClusterConfigStore.GetChannelClusterConfig(channelID, channelType)
}

func (s *Store) GetChannelClusterConfigWithSlotId(slotID uint32) ([]wkdb.ChannelClusterConfig, error) {
	channelClusterConfigStore, err := s.requireChannelClusterConfigStore()
	if err != nil {
		return nil, err
	}
	return channelClusterConfigStore.GetChannelClusterConfigWithSlotId(slotID)
}

func (s *Store) GetChannelClusterConfigVersion(channelID string, channelType uint8) (uint64, error) {
	channelClusterConfigStore, err := s.requireChannelClusterConfigStore()
	if err != nil {
		return 0, err
	}
	return channelClusterConfigStore.GetChannelClusterConfigVersion(channelID, channelType)
}
