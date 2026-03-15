package v3

import (
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/stretchr/testify/require"
)

func TestPebbleDeviceStorePutGetListDelete(t *testing.T) {
	now := time.Unix(1710000000, 0)
	db := newTestPebbleDBWithNow(t, func() time.Time { return now })
	defer db.Close()

	store := db.Slots().Scope(7).Devices()
	require.NoError(t, store.Put(wkdb.Device{
		Id:          1,
		Uid:         "user-1",
		Token:       "token-1",
		DeviceFlag:  1,
		DeviceLevel: 1,
	}))
	require.NoError(t, store.Put(wkdb.Device{
		Id:          2,
		Uid:         "user-1",
		Token:       "token-2",
		DeviceFlag:  2,
		DeviceLevel: 2,
		CreatedAt:   unixNanoPtr(uint64(now.Add(time.Second).UnixNano())),
		UpdatedAt:   unixNanoPtr(uint64(now.Add(time.Second).UnixNano())),
	}))
	require.NoError(t, store.Put(wkdb.Device{
		Id:          3,
		Uid:         "user-2",
		Token:       "token-3",
		DeviceFlag:  1,
		DeviceLevel: 3,
	}))

	device, err := store.Get("user-1", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(2), device.Id)
	require.Equal(t, "token-2", device.Token)
	require.NotNil(t, device.CreatedAt)

	devices, err := store.ListByUID("user-1")
	require.NoError(t, err)
	require.Len(t, devices, 2)
	require.Equal(t, uint64(2), devices[0].DeviceFlag)
	require.Equal(t, uint64(1), devices[1].DeviceFlag)

	require.NoError(t, store.Delete("user-1", 2))
	_, err = store.Get("user-1", 2)
	require.ErrorIs(t, err, wkdb.ErrNotFound)

	devices, err = store.ListByUID("user-1")
	require.NoError(t, err)
	require.Len(t, devices, 1)
	require.Equal(t, uint64(1), devices[0].DeviceFlag)
	require.NoError(t, store.Delete("user-1", 2))
}

func TestPebbleDeviceStoreUpdateAndSearch(t *testing.T) {
	current := time.Unix(1710000000, 0)
	db := newTestPebbleDBWithNow(t, func() time.Time { return current })
	defer db.Close()

	store := db.Slots().Scope(7).Devices()
	created1 := time.Unix(1710000000, 100)
	created2 := time.Unix(1710000000, 200)
	created3 := time.Unix(1710000000, 300)

	require.NoError(t, store.Put(wkdb.Device{
		Id:          1,
		Uid:         "user-1",
		Token:       "token-1",
		DeviceFlag:  1,
		DeviceLevel: 1,
		CreatedAt:   &created1,
		UpdatedAt:   &created1,
	}))
	require.NoError(t, store.Put(wkdb.Device{
		Id:          2,
		Uid:         "user-1",
		Token:       "token-2",
		DeviceFlag:  2,
		DeviceLevel: 2,
		CreatedAt:   &created2,
		UpdatedAt:   &created2,
	}))
	require.NoError(t, store.Put(wkdb.Device{
		Id:          3,
		Uid:         "user-2",
		Token:       "token-3",
		DeviceFlag:  1,
		DeviceLevel: 3,
		CreatedAt:   &created3,
		UpdatedAt:   &created3,
	}))

	devices, err := store.Search(wkdb.DeviceSearchReq{
		Uid:   "user-1",
		Limit: 1,
	})
	require.NoError(t, err)
	require.Len(t, devices, 1)
	require.Equal(t, uint64(2), devices[0].DeviceFlag)

	devices, err = store.Search(wkdb.DeviceSearchReq{
		Uid:             "user-1",
		Limit:           10,
		OffsetCreatedAt: created2.UnixNano(),
	})
	require.NoError(t, err)
	require.Len(t, devices, 1)
	require.Equal(t, uint64(1), devices[0].DeviceFlag)

	devices, err = store.Search(wkdb.DeviceSearchReq{
		Uid:             "user-1",
		DeviceFlag:      2,
		Limit:           10,
		OffsetCreatedAt: created2.UnixNano(),
	})
	require.NoError(t, err)
	require.Nil(t, devices)

	devices, err = store.Search(wkdb.DeviceSearchReq{
		DeviceFlag: 1,
		Limit:      10,
	})
	require.NoError(t, err)
	require.Len(t, devices, 2)
	require.Equal(t, "user-2", devices[0].Uid)
	require.Equal(t, "user-1", devices[1].Uid)

	current = current.Add(time.Minute)
	overriddenCreatedAt := created3.Add(time.Hour)
	require.NoError(t, store.Put(wkdb.Device{
		Id:          22,
		Uid:         "user-1",
		Token:       "token-2b",
		DeviceFlag:  2,
		DeviceLevel: 9,
		CreatedAt:   &overriddenCreatedAt,
	}))

	device, err := store.Get("user-1", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(22), device.Id)
	require.Equal(t, "token-2b", device.Token)
	require.Equal(t, uint8(9), device.DeviceLevel)
	require.NotNil(t, device.CreatedAt)
	require.NotNil(t, device.UpdatedAt)
	require.Equal(t, created2.UnixNano(), device.CreatedAt.UnixNano())
	require.Equal(t, current.UnixNano(), device.UpdatedAt.UnixNano())
}
