package v3

import (
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/stretchr/testify/require"
)

func TestPebbleUserStorePutGetExistsDelete(t *testing.T) {
	now := time.Unix(1710000000, 0)
	db := newTestPebbleDBWithNow(t, func() time.Time { return now })
	defer db.Close()

	store := db.Slots().Scope(7).Users()

	exists, err := store.Exists("user-1")
	require.NoError(t, err)
	require.False(t, exists)

	require.NoError(t, store.Put(wkdb.User{
		Uid:         "user-1",
		DeviceCount: 3,
		PluginNo:    "plugin-a",
	}))

	user, err := store.Get("user-1")
	require.NoError(t, err)
	require.Equal(t, "user-1", user.Uid)
	require.Equal(t, uint32(3), user.DeviceCount)
	require.Equal(t, "plugin-a", user.PluginNo)
	require.NotNil(t, user.CreatedAt)
	require.NotNil(t, user.UpdatedAt)
	require.Equal(t, now.UnixNano(), user.CreatedAt.UnixNano())
	require.Equal(t, now.UnixNano(), user.UpdatedAt.UnixNano())

	exists, err = store.Exists("user-1")
	require.NoError(t, err)
	require.True(t, exists)

	require.NoError(t, store.Delete("user-1"))
	_, err = store.Get("user-1")
	require.ErrorIs(t, err, wkdb.ErrNotFound)

	exists, err = store.Exists("user-1")
	require.NoError(t, err)
	require.False(t, exists)
	require.NoError(t, store.Delete("user-1"))
}

func TestPebbleUserStoreUpdateAndSearch(t *testing.T) {
	current := time.Unix(1710000000, 0)
	db := newTestPebbleDBWithNow(t, func() time.Time { return current })
	defer db.Close()

	store := db.Slots().Scope(7).Users()
	created1 := time.Unix(1710000000, 100)
	created2 := time.Unix(1710000000, 200)
	created3 := time.Unix(1710000000, 300)

	require.NoError(t, store.Put(wkdb.User{
		Id:        1,
		Uid:       "user-1",
		PluginNo:  "plugin-a",
		CreatedAt: &created1,
		UpdatedAt: &created1,
	}))
	require.NoError(t, store.Put(wkdb.User{
		Id:          2,
		Uid:         "user-2",
		DeviceCount: 2,
		CreatedAt:   &created2,
		UpdatedAt:   &created2,
	}))
	require.NoError(t, store.Put(wkdb.User{
		Id:          3,
		Uid:         "user-3",
		DeviceCount: 5,
		CreatedAt:   &created3,
		UpdatedAt:   &created3,
	}))

	users, err := store.Search(wkdb.UserSearchReq{Limit: 2})
	require.NoError(t, err)
	require.Len(t, users, 2)
	require.Equal(t, "user-3", users[0].Uid)
	require.Equal(t, "user-2", users[1].Uid)

	users, err = store.Search(wkdb.UserSearchReq{
		Limit:           10,
		OffsetCreatedAt: created3.UnixNano(),
	})
	require.NoError(t, err)
	require.Len(t, users, 2)
	require.Equal(t, "user-2", users[0].Uid)
	require.Equal(t, "user-1", users[1].Uid)

	current = current.Add(time.Minute)
	overriddenCreatedAt := created3.Add(time.Hour)
	require.NoError(t, store.Put(wkdb.User{
		Id:          22,
		Uid:         "user-2",
		DeviceCount: 8,
		CreatedAt:   &overriddenCreatedAt,
	}))

	user, err := store.Get("user-2")
	require.NoError(t, err)
	require.Equal(t, uint64(22), user.Id)
	require.Equal(t, uint32(8), user.DeviceCount)
	require.NotNil(t, user.CreatedAt)
	require.NotNil(t, user.UpdatedAt)
	require.Equal(t, created2.UnixNano(), user.CreatedAt.UnixNano())
	require.Equal(t, current.UnixNano(), user.UpdatedAt.UnixNano())

	users, err = store.Search(wkdb.UserSearchReq{Uid: "user-2"})
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "user-2", users[0].Uid)
}
