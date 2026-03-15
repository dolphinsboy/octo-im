package v3

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	"github.com/cockroachdb/pebble"
)

type pebbleChannelClusterConfigStore struct {
	slotID  uint32
	buckets *PebbleBucketManager
	now     func() time.Time
}

var _ ChannelClusterConfigStore = (*pebbleChannelClusterConfigStore)(nil)

func (p *pebbleChannelClusterConfigStore) Get(channelID string, channelType uint8) (wkdb.ChannelClusterConfig, error) {
	bucket, err := p.bucket()
	if err != nil {
		return wkdb.ChannelClusterConfig{}, err
	}
	value, closer, err := bucket.DB().Get(v3key.EncodeChannelClusterConfigRowKey(p.slotID, channelID, channelType))
	if err != nil {
		if err == pebble.ErrNotFound {
			return wkdb.ChannelClusterConfig{}, wkdb.ErrNotFound
		}
		return wkdb.ChannelClusterConfig{}, err
	}
	defer closer.Close()
	return decodeChannelClusterConfigValue(value)
}

func (p *pebbleChannelClusterConfigStore) Put(cfg wkdb.ChannelClusterConfig) error {
	if strings.TrimSpace(cfg.ChannelId) == "" {
		return fmt.Errorf("channel id is required")
	}
	bucket, err := p.bucket()
	if err != nil {
		return err
	}
	oldCfg, err := p.Get(cfg.ChannelId, cfg.ChannelType)
	if err != nil && err != wkdb.ErrNotFound {
		return err
	}
	existed := err == nil
	now := p.currentTime()
	if cfg.Id == 0 {
		cfg.Id = hashString(wkdb.ChannelToKey(cfg.ChannelId, cfg.ChannelType))
	}
	if existed && oldCfg.CreatedAt != nil {
		cfg.CreatedAt = cloneTimePtr(oldCfg.CreatedAt)
	} else if cfg.CreatedAt == nil {
		cfg.CreatedAt = cloneTimePtr(&now)
	}
	if cfg.UpdatedAt == nil {
		cfg.UpdatedAt = cloneTimePtr(&now)
	}

	batch := bucket.DB().NewBatch()
	defer batch.Close()
	if existed && oldCfg.CreatedAt != nil {
		if err := batch.Delete(v3key.EncodeChannelClusterConfigCreatedAtSecondIndexKey(p.slotID, uint64(oldCfg.CreatedAt.UnixNano()), oldCfg.ChannelId, oldCfg.ChannelType), p.writeOptions()); err != nil {
			return err
		}
	}
	value, err := encodeChannelClusterConfigValue(cfg)
	if err != nil {
		return err
	}
	if err := batch.Set(v3key.EncodeChannelClusterConfigRowKey(p.slotID, cfg.ChannelId, cfg.ChannelType), value, p.writeOptions()); err != nil {
		return err
	}
	if cfg.CreatedAt != nil {
		if err := batch.Set(v3key.EncodeChannelClusterConfigCreatedAtSecondIndexKey(p.slotID, uint64(cfg.CreatedAt.UnixNano()), cfg.ChannelId, cfg.ChannelType), nil, p.writeOptions()); err != nil {
			return err
		}
	}
	return batch.Commit(p.writeOptions())
}

func (p *pebbleChannelClusterConfigStore) Search(req wkdb.ChannelClusterConfigSearchReq) ([]wkdb.ChannelClusterConfig, error) {
	if req.ChannelId != "" {
		cfg, err := p.Get(req.ChannelId, req.ChannelType)
		if err != nil {
			if err == wkdb.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}
		if !matchChannelClusterConfigSearch(cfg, req) {
			return nil, nil
		}
		return []wkdb.ChannelClusterConfig{cloneChannelClusterConfig(cfg)}, nil
	}

	bucket, err := p.bucket()
	if err != nil {
		return nil, err
	}
	lower, upper := v3key.SlotTableRange(p.slotID, v3key.ScopeSlotPrimary, v3key.TableChannelClusterConfig)
	iter := bucket.DB().NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	cfgs := make([]wkdb.ChannelClusterConfig, 0)
	for iter.First(); iter.Valid(); iter.Next() {
		cfg, err := decodeChannelClusterConfigValue(iter.Value())
		if err != nil {
			return nil, err
		}
		if !matchChannelClusterConfigSearch(cfg, req) {
			continue
		}
		cfgs = append(cfgs, cfg)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
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

func (p *pebbleChannelClusterConfigStore) bucket() (*PebbleBucket, error) {
	return p.buckets.Bucket(p.slotID)
}

func (p *pebbleChannelClusterConfigStore) currentTime() time.Time {
	if p.now != nil {
		return p.now()
	}
	return time.Now()
}

func (p *pebbleChannelClusterConfigStore) writeOptions() *pebble.WriteOptions {
	bucket, _ := p.bucket()
	if bucket == nil {
		return pebble.Sync
	}
	return bucket.SnapshotStore().writeOptions
}

func encodeChannelClusterConfigValue(cfg wkdb.ChannelClusterConfig) ([]byte, error) {
	if strings.TrimSpace(cfg.ChannelId) == "" {
		return nil, fmt.Errorf("channel id is required")
	}
	if cfg.CreatedAt == nil || cfg.UpdatedAt == nil {
		return nil, fmt.Errorf("channel cluster config timestamps are required")
	}
	return cfg.Marshal()
}

func decodeChannelClusterConfigValue(data []byte) (wkdb.ChannelClusterConfig, error) {
	var cfg wkdb.ChannelClusterConfig
	if err := cfg.Unmarshal(data); err != nil {
		return wkdb.ChannelClusterConfig{}, err
	}
	if cfg.Id == 0 {
		cfg.Id = hashString(wkdb.ChannelToKey(cfg.ChannelId, cfg.ChannelType))
	}
	return cfg, nil
}

func matchChannelClusterConfigSearch(cfg wkdb.ChannelClusterConfig, req wkdb.ChannelClusterConfigSearchReq) bool {
	if req.ChannelId != "" && req.ChannelId != cfg.ChannelId {
		return false
	}
	if req.ChannelType != 0 && req.ChannelType != cfg.ChannelType {
		return false
	}
	// SlotLeaderId is cluster-topology data rather than row data in the slot store.
	// The slot-scoped store therefore cannot authoritatively filter by it.
	ts := channelClusterConfigSortTime(cfg)
	if req.OffsetCreatedAt <= 0 {
		return true
	}
	if req.Pre {
		return ts > req.OffsetCreatedAt
	}
	return ts < req.OffsetCreatedAt
}

func channelClusterConfigSortTime(cfg wkdb.ChannelClusterConfig) int64 {
	if cfg.CreatedAt != nil {
		return cfg.CreatedAt.UnixNano()
	}
	return 0
}

func limitChannelClusterConfigs(cfgs []wkdb.ChannelClusterConfig, limit int, pre bool) []wkdb.ChannelClusterConfig {
	if len(cfgs) == 0 {
		return nil
	}
	if limit > 0 && len(cfgs) > limit {
		if pre {
			cfgs = cfgs[len(cfgs)-limit:]
		} else {
			cfgs = cfgs[:limit]
		}
	}
	out := make([]wkdb.ChannelClusterConfig, 0, len(cfgs))
	for _, cfg := range cfgs {
		out = append(out, cloneChannelClusterConfig(cfg))
	}
	return out
}

func cloneChannelClusterConfig(cfg wkdb.ChannelClusterConfig) wkdb.ChannelClusterConfig {
	cfg.Replicas = append([]uint64(nil), cfg.Replicas...)
	cfg.Learners = append([]uint64(nil), cfg.Learners...)
	cfg.CreatedAt = cloneTimePtr(cfg.CreatedAt)
	cfg.UpdatedAt = cloneTimePtr(cfg.UpdatedAt)
	return cfg
}
