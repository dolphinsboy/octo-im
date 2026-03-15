package store

import (
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
	"github.com/WuKongIM/WuKongIM/pkg/wkutil"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/stretchr/testify/require"
)

func TestSlotAdminSearchStoreAggregatesAcrossSlots(t *testing.T) {
	slotCount := uint32(2)
	db, routeSlot := newAdminSearchTestDB(t, slotCount)
	searchStore := NewSlotAdminSearchStore(db, slotCount, routeSlot)

	user1Time := time.Unix(1710000000, 0)
	user2Time := time.Unix(1710000100, 0)

	require.NoError(t, db.Slots().Scope(routeSlot("user-1")).Users().Put(wkdb.User{
		Id:        1,
		Uid:       "user-1",
		CreatedAt: &user1Time,
		UpdatedAt: &user1Time,
	}))
	require.NoError(t, db.Slots().Scope(routeSlot("user-2")).Users().Put(wkdb.User{
		Id:        2,
		Uid:       "user-2",
		CreatedAt: &user2Time,
		UpdatedAt: &user2Time,
	}))

	require.NoError(t, db.Slots().Scope(routeSlot("user-1")).Devices().Put(wkdb.Device{
		Id:         1,
		Uid:        "user-1",
		DeviceFlag: 1,
		CreatedAt:  &user1Time,
		UpdatedAt:  &user1Time,
	}))
	require.NoError(t, db.Slots().Scope(routeSlot("user-2")).Devices().Put(wkdb.Device{
		Id:         2,
		Uid:        "user-2",
		DeviceFlag: 2,
		CreatedAt:  &user2Time,
		UpdatedAt:  &user2Time,
	}))

	require.NoError(t, db.Slots().Scope(routeSlot("channel-1")).Channels().Put(wkdb.ChannelInfo{
		Id:          1,
		ChannelId:   "channel-1",
		ChannelType: 2,
		CreatedAt:   &user1Time,
		UpdatedAt:   &user1Time,
	}))
	require.NoError(t, db.Slots().Scope(routeSlot("channel-2")).Channels().Put(wkdb.ChannelInfo{
		Id:          2,
		ChannelId:   "channel-2",
		ChannelType: 2,
		CreatedAt:   &user2Time,
		UpdatedAt:   &user2Time,
	}))

	require.NoError(t, db.Slots().Scope(routeSlot("user-1")).Conversations().Put("user-1", []wkdb.Conversation{{
		Id:          1,
		Uid:         "user-1",
		ChannelId:   "channel-1",
		ChannelType: 2,
		CreatedAt:   &user1Time,
		UpdatedAt:   &user1Time,
	}}))
	require.NoError(t, db.Slots().Scope(routeSlot("user-2")).Conversations().Put("user-2", []wkdb.Conversation{{
		Id:          2,
		Uid:         "user-2",
		ChannelId:   "channel-2",
		ChannelType: 2,
		CreatedAt:   &user2Time,
		UpdatedAt:   &user2Time,
	}}))

	require.NoError(t, db.Slots().Scope(routeSlot("channel-1")).ChannelClusterConfigs().Put(wkdb.ChannelClusterConfig{
		Id:          1,
		ChannelId:   "channel-1",
		ChannelType: 2,
		LeaderId:    100,
		CreatedAt:   &user1Time,
		UpdatedAt:   &user1Time,
	}))
	require.NoError(t, db.Slots().Scope(routeSlot("channel-2")).ChannelClusterConfigs().Put(wkdb.ChannelClusterConfig{
		Id:          2,
		ChannelId:   "channel-2",
		ChannelType: 2,
		LeaderId:    200,
		CreatedAt:   &user2Time,
		UpdatedAt:   &user2Time,
	}))

	users, err := searchStore.SearchUsers(wkdb.UserSearchReq{Limit: 1})
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "user-2", users[0].Uid)

	userCount, err := searchStore.CountUsers()
	require.NoError(t, err)
	require.Equal(t, 2, userCount)

	devices, err := searchStore.SearchDevices(wkdb.DeviceSearchReq{Limit: 2})
	require.NoError(t, err)
	require.Len(t, devices, 2)
	require.Equal(t, "user-2", devices[0].Uid)

	deviceCount, err := searchStore.CountDevices()
	require.NoError(t, err)
	require.Equal(t, 2, deviceCount)

	channels, err := searchStore.SearchChannels(wkdb.ChannelSearchReq{Limit: 1})
	require.NoError(t, err)
	require.Len(t, channels, 1)
	require.Equal(t, "channel-2", channels[0].ChannelId)

	conversations, err := searchStore.SearchConversations(wkdb.ConversationSearchReq{Limit: 1, CurrentPage: 1})
	require.NoError(t, err)
	require.Len(t, conversations, 1)
	require.Equal(t, "user-2", conversations[0].Uid)

	conversationCount, err := searchStore.CountConversations()
	require.NoError(t, err)
	require.Equal(t, 2, conversationCount)

	cfgs, err := searchStore.SearchChannelClusterConfigs(wkdb.ChannelClusterConfigSearchReq{Limit: 1})
	require.NoError(t, err)
	require.Len(t, cfgs, 1)
	require.Equal(t, "channel-2", cfgs[0].ChannelId)

	cfgs, err = searchStore.SearchChannelClusterConfigs(wkdb.ChannelClusterConfigSearchReq{Limit: 10}, func(cfg wkdb.ChannelClusterConfig) bool {
		return cfg.LeaderId == 100
	})
	require.NoError(t, err)
	require.Len(t, cfgs, 1)
	require.Equal(t, "channel-1", cfgs[0].ChannelId)
}

func newAdminSearchTestDB(t *testing.T, slotCount uint32) (wkdbv3.DB, func(string) uint32) {
	t.Helper()

	router, err := wkdbv3.NewStaticBucketRouter(2)
	require.NoError(t, err)

	db, err := wkdbv3.NewPebbleDB(wkdbv3.PebbleDBOptions{
		DataDir:      "/wkdb-v3-search",
		Router:       router,
		FS:           vfs.NewMem(),
		WriteOptions: pebble.NoSync,
		PebbleOptions: &pebble.Options{
			FormatMajorVersion: pebble.FormatNewest,
		},
		SnapshotNow: func() int64 { return 1710000000 },
		Now:         func() time.Time { return time.Unix(1710000000, 0) },
	})
	require.NoError(t, err)
	require.NoError(t, db.Open())
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	routeSlot := func(key string) uint32 {
		return wkutil.GetSlotNum(int(slotCount), key)
	}
	return db, routeSlot
}
