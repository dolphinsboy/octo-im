package v3

import (
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/stretchr/testify/require"
)

func TestPebbleChannelClusterConfigStorePutGetSearch(t *testing.T) {
	current := time.Unix(1710000000, 0)
	db := newTestPebbleDBWithNow(t, func() time.Time { return current })
	defer db.Close()

	store := db.Slots().Scope(7).ChannelClusterConfigs()
	created1 := time.Unix(1710000000, 100)
	created2 := time.Unix(1710000000, 200)
	created3 := time.Unix(1710000000, 300)

	require.NoError(t, store.Put(wkdb.ChannelClusterConfig{
		ChannelId:       "group-1",
		ChannelType:     1,
		ReplicaMaxCount: 3,
		Replicas:        []uint64{1, 2, 3},
		LeaderId:        10,
		Term:            1,
		CreatedAt:       &created1,
		UpdatedAt:       &created1,
	}))
	require.NoError(t, store.Put(wkdb.ChannelClusterConfig{
		ChannelId:       "group-2",
		ChannelType:     1,
		ReplicaMaxCount: 5,
		Replicas:        []uint64{4, 5, 6},
		Learners:        []uint64{7},
		LeaderId:        11,
		Term:            2,
		CreatedAt:       &created2,
		UpdatedAt:       &created2,
	}))
	require.NoError(t, store.Put(wkdb.ChannelClusterConfig{
		ChannelId:       "group-3",
		ChannelType:     2,
		ReplicaMaxCount: 2,
		LeaderId:        12,
		Term:            3,
		CreatedAt:       &created3,
		UpdatedAt:       &created3,
	}))

	cfg, err := store.Get("group-2", 1)
	require.NoError(t, err)
	require.Equal(t, "group-2", cfg.ChannelId)
	require.Equal(t, uint8(1), cfg.ChannelType)
	require.Equal(t, uint16(5), cfg.ReplicaMaxCount)
	require.Equal(t, []uint64{4, 5, 6}, cfg.Replicas)
	require.Equal(t, []uint64{7}, cfg.Learners)
	require.NotZero(t, cfg.Id)

	cfgs, err := store.Search(wkdb.ChannelClusterConfigSearchReq{Limit: 2})
	require.NoError(t, err)
	require.Len(t, cfgs, 2)
	require.Equal(t, "group-3", cfgs[0].ChannelId)
	require.Equal(t, "group-2", cfgs[1].ChannelId)

	cfgs, err = store.Search(wkdb.ChannelClusterConfigSearchReq{
		Limit:           10,
		OffsetCreatedAt: created3.UnixNano(),
	})
	require.NoError(t, err)
	require.Len(t, cfgs, 2)
	require.Equal(t, "group-2", cfgs[0].ChannelId)
	require.Equal(t, "group-1", cfgs[1].ChannelId)

	current = current.Add(time.Minute)
	overrideCreatedAt := created3.Add(time.Hour)
	require.NoError(t, store.Put(wkdb.ChannelClusterConfig{
		ChannelId:       "group-2",
		ChannelType:     1,
		ReplicaMaxCount: 6,
		Replicas:        []uint64{8, 9},
		LeaderId:        99,
		Term:            20,
		CreatedAt:       &overrideCreatedAt,
	}))

	cfg, err = store.Get("group-2", 1)
	require.NoError(t, err)
	require.Equal(t, uint16(6), cfg.ReplicaMaxCount)
	require.Equal(t, []uint64{8, 9}, cfg.Replicas)
	require.Equal(t, created2.UnixNano(), cfg.CreatedAt.UnixNano())
	require.Equal(t, current.UnixNano(), cfg.UpdatedAt.UnixNano())
}

func TestPebbleChannelClusterConfigStoreSearchByChannelID(t *testing.T) {
	db := newTestPebbleDBWithNow(t, time.Now)
	defer db.Close()

	store := db.Slots().Scope(7).ChannelClusterConfigs()
	require.NoError(t, store.Put(wkdb.ChannelClusterConfig{
		ChannelId:   "group-1",
		ChannelType: 2,
	}))

	cfgs, err := store.Search(wkdb.ChannelClusterConfigSearchReq{
		ChannelId:   "group-1",
		ChannelType: 2,
		Limit:       10,
	})
	require.NoError(t, err)
	require.Len(t, cfgs, 1)

	cfgs, err = store.Search(wkdb.ChannelClusterConfigSearchReq{
		ChannelId:   "missing",
		ChannelType: 2,
	})
	require.NoError(t, err)
	require.Nil(t, cfgs)
}
