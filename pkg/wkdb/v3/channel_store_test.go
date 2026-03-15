package v3

import (
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/stretchr/testify/require"
)

func TestPebbleChannelStorePutGetExistsDelete(t *testing.T) {
	now := time.Unix(1710000000, 0)
	db := newTestPebbleDBWithNow(t, func() time.Time { return now })
	defer db.Close()

	store := db.Slots().Scope(7).Channels()

	exists, err := store.Exists("group-1", 2)
	require.NoError(t, err)
	require.False(t, exists)

	require.NoError(t, store.Put(wkdb.ChannelInfo{
		ChannelId:       "group-1",
		ChannelType:     2,
		Ban:             true,
		Large:           true,
		Webhook:         "https://example.com/hook",
		AllowStranger:   true,
		SubscriberCount: 5,
	}))

	info, err := store.Get("group-1", 2)
	require.NoError(t, err)
	require.True(t, info.Ban)
	require.True(t, info.Large)
	require.Equal(t, "https://example.com/hook", info.Webhook)
	require.True(t, info.AllowStranger)
	require.Equal(t, 5, info.SubscriberCount)
	require.Equal(t, now.UnixNano(), info.CreatedAt.UnixNano())
	require.Equal(t, now.UnixNano(), info.UpdatedAt.UnixNano())

	exists, err = store.Exists("group-1", 2)
	require.NoError(t, err)
	require.True(t, exists)

	require.NoError(t, store.Delete("group-1", 2))
	_, err = store.Get("group-1", 2)
	require.ErrorIs(t, err, wkdb.ErrNotFound)
}

func TestPebbleChannelStoreUpdateAndSearch(t *testing.T) {
	current := time.Unix(1710000000, 0)
	db := newTestPebbleDBWithNow(t, func() time.Time { return current })
	defer db.Close()

	store := db.Slots().Scope(7).Channels()
	created1 := time.Unix(1710000000, 100)
	created2 := time.Unix(1710000000, 200)
	created3 := time.Unix(1710000000, 300)

	require.NoError(t, store.Put(wkdb.ChannelInfo{
		Id:              1,
		ChannelId:       "group-1",
		ChannelType:     1,
		Ban:             true,
		SubscriberCount: 1,
		CreatedAt:       &created1,
		UpdatedAt:       &created1,
	}))
	require.NoError(t, store.Put(wkdb.ChannelInfo{
		Id:             2,
		ChannelId:      "group-2",
		ChannelType:    1,
		Disband:        true,
		AllowlistCount: 2,
		CreatedAt:      &created2,
		UpdatedAt:      &created2,
	}))
	require.NoError(t, store.Put(wkdb.ChannelInfo{
		Id:            3,
		ChannelId:     "group-3",
		ChannelType:   2,
		DenylistCount: 3,
		CreatedAt:     &created3,
		UpdatedAt:     &created3,
	}))

	ban := true
	channels, err := store.Search(wkdb.ChannelSearchReq{Ban: &ban, Limit: 10})
	require.NoError(t, err)
	require.Len(t, channels, 1)
	require.Equal(t, "group-1", channels[0].ChannelId)

	minAllow := 2
	channels, err = store.Search(wkdb.ChannelSearchReq{AllowlistCountGte: &minAllow, Limit: 10})
	require.NoError(t, err)
	require.Len(t, channels, 1)
	require.Equal(t, "group-2", channels[0].ChannelId)

	minDeny := 3
	channels, err = store.Search(wkdb.ChannelSearchReq{DenylistCountGte: &minDeny, Limit: 10})
	require.NoError(t, err)
	require.Len(t, channels, 1)
	require.Equal(t, "group-3", channels[0].ChannelId)

	channels, err = store.Search(wkdb.ChannelSearchReq{
		Limit:           10,
		OffsetCreatedAt: created3.UnixNano(),
	})
	require.NoError(t, err)
	require.Len(t, channels, 2)
	require.Equal(t, "group-2", channels[0].ChannelId)
	require.Equal(t, "group-1", channels[1].ChannelId)

	current = current.Add(time.Minute)
	overrideCreatedAt := created3.Add(time.Hour)
	require.NoError(t, store.Put(wkdb.ChannelInfo{
		Id:          22,
		ChannelId:   "group-2",
		ChannelType: 1,
		Ban:         true,
		CreatedAt:   &overrideCreatedAt,
	}))

	info, err := store.Get("group-2", 1)
	require.NoError(t, err)
	require.Equal(t, uint64(22), info.Id)
	require.True(t, info.Ban)
	require.False(t, info.Disband)
	require.Equal(t, 2, info.AllowlistCount)
	require.Equal(t, created2.UnixNano(), info.CreatedAt.UnixNano())
	require.Equal(t, current.UnixNano(), info.UpdatedAt.UnixNano())
}
