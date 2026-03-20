package store

import (
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/stretchr/testify/require"
)

type stubChannelClusterConfigStore struct {
	cfg             wkdb.ChannelClusterConfig
	cfgsBySlot      []wkdb.ChannelClusterConfig
	lastCfgs        []wkdb.ChannelClusterConfig
	lastChannelID   string
	lastChannelType uint8
	lastSlotID      uint32
	lastOp          string
}

var _ ChannelClusterConfigStore = (*stubChannelClusterConfigStore)(nil)

func (s *stubChannelClusterConfigStore) SaveChannelClusterConfigs(cfgs []wkdb.ChannelClusterConfig) error {
	s.lastOp = "save"
	s.lastCfgs = append([]wkdb.ChannelClusterConfig(nil), cfgs...)
	return nil
}

func (s *stubChannelClusterConfigStore) GetChannelClusterConfig(channelID string, channelType uint8) (wkdb.ChannelClusterConfig, error) {
	s.lastOp = "get"
	s.lastChannelID = channelID
	s.lastChannelType = channelType
	return s.cfg, nil
}

func (s *stubChannelClusterConfigStore) GetChannelClusterConfigWithSlotId(slotID uint32) ([]wkdb.ChannelClusterConfig, error) {
	s.lastOp = "get_by_slot"
	s.lastSlotID = slotID
	return append([]wkdb.ChannelClusterConfig(nil), s.cfgsBySlot...), nil
}

func (s *stubChannelClusterConfigStore) GetChannelClusterConfigVersion(channelID string, channelType uint8) (uint64, error) {
	s.lastOp = "get_version"
	s.lastChannelID = channelID
	s.lastChannelType = channelType
	return s.cfg.ConfVersion, nil
}

func TestStoreDerivesChannelClusterConfigStoreFromHybridDB(t *testing.T) {
	hybrid, _ := newHybridMetaLocalTestDB(t)
	now := time.Unix(1710000000, 0)
	cfg := wkdb.ChannelClusterConfig{
		Id:          1,
		ChannelId:   "channel-1",
		ChannelType: 2,
		LeaderId:    100,
		ConfVersion: 7,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}

	require.NoError(t, hybrid.slotDB.Slots().Scope(0).ChannelClusterConfigs().Put(cfg))

	s := New(NewOptions(WithCompatDBRuntime(hybrid)))
	require.NotNil(t, s.channelClusterConfigStore)

	got, err := s.GetChannelClusterConfig("channel-1", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(100), got.LeaderId)

	version, err := s.GetChannelClusterConfigVersion("channel-1", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(7), version)

	cfgs, err := s.GetChannelClusterConfigWithSlotId(0)
	require.NoError(t, err)
	require.Len(t, cfgs, 1)
	require.Equal(t, "channel-1", cfgs[0].ChannelId)
}

func TestStoreChannelClusterConfigReadsUseStoreBoundary(t *testing.T) {
	now := time.Unix(1710000000, 0)
	cfg := wkdb.ChannelClusterConfig{
		Id:          1,
		ChannelId:   "channel-1",
		ChannelType: 2,
		LeaderId:    100,
		ConfVersion: 7,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	stub := &stubChannelClusterConfigStore{
		cfg:        cfg,
		cfgsBySlot: []wkdb.ChannelClusterConfig{cfg},
	}
	s := &Store{channelClusterConfigStore: stub}

	got, err := s.GetChannelClusterConfig("channel-1", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(100), got.LeaderId)
	require.Equal(t, "get", stub.lastOp)

	version, err := s.GetChannelClusterConfigVersion("channel-1", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(7), version)
	require.Equal(t, "get_version", stub.lastOp)

	cfgs, err := s.GetChannelClusterConfigWithSlotId(11)
	require.NoError(t, err)
	require.Len(t, cfgs, 1)
	require.Equal(t, uint32(11), stub.lastSlotID)
	require.Equal(t, "get_by_slot", stub.lastOp)
}

func TestStoreApplyChannelClusterConfigSavesUsesStoreBoundary(t *testing.T) {
	now := time.Unix(1710000000, 0)
	cfg1 := wkdb.ChannelClusterConfig{
		Id:          1,
		ChannelId:   "channel-1",
		ChannelType: 2,
		ConfVersion: 10,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	cfg2 := wkdb.ChannelClusterConfig{
		Id:          2,
		ChannelId:   "channel-2",
		ChannelType: 2,
		ConfVersion: 11,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}

	errCh1 := make(chan error, 1)
	errCh2 := make(chan error, 1)
	stub := &stubChannelClusterConfigStore{}
	s := &Store{channelClusterConfigStore: stub}

	require.NoError(t, s.handleChannelClusterConfigSaves([]*channelCfgReq{
		{cfg: cfg1, errCh: errCh1},
		{cfg: cfg2, errCh: errCh2},
	}))
	require.Equal(t, "save", stub.lastOp)
	require.Len(t, stub.lastCfgs, 2)
	require.Equal(t, "channel-1", stub.lastCfgs[0].ChannelId)
	require.Equal(t, "channel-2", stub.lastCfgs[1].ChannelId)
	require.NoError(t, <-errCh1)
	require.NoError(t, <-errCh2)
}

func TestStoreSaveChannelClusterConfigRoutesByChannelSlot(t *testing.T) {
	s, slot := newTestStoreWithSlot(map[string]uint32{
		"channel-1": 11,
	})
	now := time.Unix(1710000000, 0)

	version, err := s.SaveChannelClusterConfig(wkdb.ChannelClusterConfig{
		Id:          1,
		ChannelId:   "channel-1",
		ChannelType: 2,
		ConfVersion: 3,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(0), version)
	require.Equal(t, []string{"channel-1"}, slot.getSlotIdCalls)
	require.Equal(t, []uint32{11}, slot.proposedUntilAppliedSlots)
}

func TestStoreStartStopReinitializesRuntimeWorkers(t *testing.T) {
	now := time.Unix(1710000000, 0)
	stub := &stubChannelClusterConfigStore{}
	s := New(NewOptions(WithChannelClusterConfigStore(stub)))

	pushCfg := func(cfg wkdb.ChannelClusterConfig) {
		rt := s.runtimeSnapshot()
		require.NotNil(t, rt)
		errCh := make(chan error, 1)
		rt.channelCfgCh <- &channelCfgReq{cfg: cfg, errCh: errCh}
		require.NoError(t, <-errCh)
		require.Equal(t, "save", stub.lastOp)
		require.Len(t, stub.lastCfgs, 1)
		require.Equal(t, cfg.ChannelId, stub.lastCfgs[0].ChannelId)
	}

	require.NoError(t, s.Start())
	require.NoError(t, s.Start())
	pushCfg(wkdb.ChannelClusterConfig{
		Id:          1,
		ChannelId:   "channel-1",
		ChannelType: 2,
		ConfVersion: 10,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	})

	s.Stop()
	require.Nil(t, s.runtimeSnapshot())

	require.NoError(t, s.Start())
	pushCfg(wkdb.ChannelClusterConfig{
		Id:          2,
		ChannelId:   "channel-2",
		ChannelType: 2,
		ConfVersion: 11,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	})
	s.Stop()
	require.Nil(t, s.runtimeSnapshot())
}
