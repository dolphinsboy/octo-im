package v3

import (
	"context"
	"testing"

	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/stretchr/testify/require"
)

func TestPebbleBucketManagerOpenAndRoute(t *testing.T) {
	router, err := NewStaticBucketRouter(4)
	require.NoError(t, err)

	manager, err := NewPebbleBucketManager(PebbleBucketManagerOptions{
		DataDir:       "/wkdb-v3",
		Router:        router,
		FS:            vfs.NewMem(),
		WriteOptions:  pebble.NoSync,
		PebbleOptions: &pebble.Options{FormatMajorVersion: pebble.FormatNewest},
	})
	require.NoError(t, err)
	require.NoError(t, manager.Open())
	defer manager.Close()

	require.Equal(t, uint32(4), manager.BucketCount())
	require.Equal(t, uint32(3), manager.BucketForSlot(7))

	bucket3, err := manager.Bucket(7)
	require.NoError(t, err)
	require.Equal(t, uint32(3), bucket3.ID())

	bucket0, err := manager.Bucket(8)
	require.NoError(t, err)
	require.Equal(t, uint32(0), bucket0.ID())

	require.NoError(t, bucket3.DB().Set(v3key.EncodeConversationRowKey(7, "user-1", "group-100", 2), []byte("slot-7"), pebble.NoSync))
	require.NoError(t, bucket0.DB().Set(v3key.EncodeConversationRowKey(8, "user-2", "group-200", 2), []byte("slot-8"), pebble.NoSync))

	records := make([]SlotSnapshotRecord, 0)
	err = bucket3.SnapshotStore().ScanSlotKV(context.Background(), 7, func(key, value []byte) error {
		records = append(records, SlotSnapshotRecord{Key: key, Value: value})
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []SlotSnapshotRecord{{
		Key:   v3key.EncodeConversationRowKey(7, "user-1", "group-100", 2),
		Value: []byte("slot-7"),
	}}, records)
}

func TestPebbleBucketManagerBucketByIDRequiresOpen(t *testing.T) {
	router, err := NewStaticBucketRouter(2)
	require.NoError(t, err)

	manager, err := NewPebbleBucketManager(PebbleBucketManagerOptions{
		DataDir:       "/wkdb-v3",
		Router:        router,
		FS:            vfs.NewMem(),
		PebbleOptions: &pebble.Options{FormatMajorVersion: pebble.FormatNewest},
	})
	require.NoError(t, err)

	_, err = manager.BucketByID(0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not open")
}
