package v3

import (
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/stretchr/testify/require"
)

func TestPebbleSubscriberStorePutListDeleteAndCount(t *testing.T) {
	now := time.Unix(1710000000, 0)
	db := newTestPebbleDBWithNow(t, func() time.Time { return now })
	defer db.Close()

	scope := db.Slots().Scope(7)
	require.NoError(t, scope.Channels().Put(wkdb.ChannelInfo{
		ChannelId:   "group-1",
		ChannelType: 2,
	}))

	store := scope.Subscribers()
	require.NoError(t, store.Put("group-1", 2, []wkdb.Member{
		{Uid: "u1"},
		{Uid: "u2", CreatedAt: unixNanoPtr(uint64(now.Add(time.Second).UnixNano())), UpdatedAt: unixNanoPtr(uint64(now.Add(time.Second).UnixNano()))},
		{Uid: "u2"},
	}))

	members, err := store.List("group-1", 2)
	require.NoError(t, err)
	require.Len(t, members, 2)
	require.Equal(t, "u1", members[0].Uid)
	require.Equal(t, "u2", members[1].Uid)
	require.NotZero(t, members[0].Id)

	channel, err := scope.Channels().Get("group-1", 2)
	require.NoError(t, err)
	require.Equal(t, 2, channel.SubscriberCount)

	require.NoError(t, store.Delete("group-1", 2, []string{"u1", "u1"}))
	members, err = store.List("group-1", 2)
	require.NoError(t, err)
	require.Len(t, members, 1)
	require.Equal(t, "u2", members[0].Uid)

	channel, err = scope.Channels().Get("group-1", 2)
	require.NoError(t, err)
	require.Equal(t, 1, channel.SubscriberCount)

	require.NoError(t, store.DeleteAll("group-1", 2))
	members, err = store.List("group-1", 2)
	require.NoError(t, err)
	require.Nil(t, members)

	channel, err = scope.Channels().Get("group-1", 2)
	require.NoError(t, err)
	require.Equal(t, 0, channel.SubscriberCount)
}

func TestPebbleAllowlistAndDenylistStoresUpdateChannelCounts(t *testing.T) {
	now := time.Unix(1710000000, 0)
	db := newTestPebbleDBWithNow(t, func() time.Time { return now })
	defer db.Close()

	scope := db.Slots().Scope(7)
	require.NoError(t, scope.Channels().Put(wkdb.ChannelInfo{
		ChannelId:   "group-2",
		ChannelType: 3,
	}))

	require.NoError(t, scope.Allowlists().Put("group-2", 3, []wkdb.Member{
		{Uid: "a1"},
		{Uid: "a2"},
	}))
	require.NoError(t, scope.Denylists().Put("group-2", 3, []wkdb.Member{
		{Uid: "d1"},
	}))

	channel, err := scope.Channels().Get("group-2", 3)
	require.NoError(t, err)
	require.Equal(t, 2, channel.AllowlistCount)
	require.Equal(t, 1, channel.DenylistCount)

	allowMembers, err := scope.Allowlists().List("group-2", 3)
	require.NoError(t, err)
	require.Len(t, allowMembers, 2)
	denyMembers, err := scope.Denylists().List("group-2", 3)
	require.NoError(t, err)
	require.Len(t, denyMembers, 1)

	require.NoError(t, scope.Allowlists().Delete("group-2", 3, []string{"a2"}))
	require.NoError(t, scope.Denylists().DeleteAll("group-2", 3))

	channel, err = scope.Channels().Get("group-2", 3)
	require.NoError(t, err)
	require.Equal(t, 1, channel.AllowlistCount)
	require.Equal(t, 0, channel.DenylistCount)
}

func TestPebbleChannelMemberStorePutRequiresChannel(t *testing.T) {
	db := newTestPebbleDBWithNow(t, time.Now)
	defer db.Close()

	err := db.Slots().Scope(7).Subscribers().Put("missing", 1, []wkdb.Member{{Uid: "u1"}})
	require.ErrorIs(t, err, wkdb.ErrNotFound)
}
