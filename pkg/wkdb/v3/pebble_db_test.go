package v3

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	wkproto "github.com/WuKongIM/WuKongIMGoProto"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/stretchr/testify/require"
)

func TestPebbleDBOpenRawAndSnapshotter(t *testing.T) {
	router, err := NewStaticBucketRouter(4)
	require.NoError(t, err)

	var cleared []uint32
	var rebuilt []uint32
	db, err := NewPebbleDB(PebbleDBOptions{
		DataDir:      "/wkdb-v3",
		Router:       router,
		FS:           vfs.NewMem(),
		WriteOptions: pebble.NoSync,
		PebbleOptions: &pebble.Options{
			FormatMajorVersion: pebble.FormatNewest,
		},
		ClearSlotCachesFn: func(slotID uint32) {
			cleared = append(cleared, slotID)
		},
		RebuildDerivedState: func(ctx context.Context, slotID uint32) error {
			rebuilt = append(rebuilt, slotID)
			return nil
		},
		SnapshotNow: func() int64 { return 1710000000 },
	})
	require.NoError(t, err)
	require.NoError(t, db.Open())
	defer db.Close()

	bucket, err := db.bucket(7)
	require.NoError(t, err)
	slot7Row := v3key.EncodeConversationRowKey(7, "user-1", "group-100", 2)
	slot7Idx := v3key.EncodeConversationUpdatedAtSecondIndexKey(7, "user-1", 100, "group-100", 2)
	slot8Row := v3key.EncodeConversationRowKey(8, "user-2", "group-200", 2)
	require.NoError(t, bucket.DB().Set(slot7Row, []byte("row"), pebble.NoSync))
	require.NoError(t, bucket.DB().Set(slot7Idx, []byte("idx"), pebble.NoSync))

	otherBucket, err := db.bucket(8)
	require.NoError(t, err)
	require.NoError(t, otherBucket.DB().Set(slot8Row, []byte("other"), pebble.NoSync))

	scope := db.Slots().Scope(7)
	require.Equal(t, uint32(7), scope.SlotID())
	records := make([]SlotSnapshotRecord, 0)
	err = scope.Raw().Range(context.Background(), func(key, value []byte) error {
		records = append(records, SlotSnapshotRecord{Key: key, Value: value})
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []SlotSnapshotRecord{
		{Key: slot7Row, Value: []byte("row")},
		{Key: slot7Idx, Value: []byte("idx")},
	}, records)

	got, err := scope.Raw().Get(slot7Row)
	require.NoError(t, err)
	require.Equal(t, []byte("row"), got)

	var buf bytes.Buffer
	meta, err := db.Snapshotter().ExportSlotKV(context.Background(), 7, &buf)
	require.NoError(t, err)
	require.Equal(t, int64(1710000000), meta.ExportedAt)
	require.Equal(t, uint64(2), meta.RecordCount)

	require.NoError(t, db.Snapshotter().VerifySlotKV(context.Background(), 7, bytes.NewReader(buf.Bytes()), meta))
	require.NoError(t, db.Maintenance().RebuildDerivedState(context.Background(), 7))
	db.Maintenance().ClearSlotCaches(7)
	require.Equal(t, []uint32{7}, rebuilt)
	require.Equal(t, []uint32{7}, cleared)
}

func TestPebbleDBMetaAndLocalStoresAreAvailable(t *testing.T) {
	router, err := NewStaticBucketRouter(2)
	require.NoError(t, err)

	db, err := NewPebbleDB(PebbleDBOptions{
		DataDir:       "/wkdb-v3",
		Router:        router,
		FS:            vfs.NewMem(),
		PebbleOptions: &pebble.Options{FormatMajorVersion: pebble.FormatNewest},
	})
	require.NoError(t, err)
	require.NoError(t, db.Open())
	defer db.Close()

	_, err = db.Slots().Scope(7).Users().Get("user-1")
	require.ErrorIs(t, err, wkdb.ErrNotFound)
	devices, err := db.Slots().Scope(7).Devices().Search(wkdb.DeviceSearchReq{})
	require.NoError(t, err)
	require.Nil(t, devices)
	exists, err := db.Slots().Scope(7).Channels().Exists("channel-1", 1)
	require.NoError(t, err)
	require.False(t, exists)
	members, err := db.Slots().Scope(7).Subscribers().List("channel-1", 1)
	require.NoError(t, err)
	require.Nil(t, members)
	_, err = db.Slots().Scope(7).ChannelClusterConfigs().Get("channel-1", 1)
	require.ErrorIs(t, err, wkdb.ErrNotFound)
	state, err := db.Slots().Scope(7).MessageEvents().GetState("channel-1", 1, "client-1", "")
	require.NoError(t, err)
	require.Nil(t, state)
	require.NoError(t, db.Meta().SystemUIDs().PutAll([]string{"system-1"}))
	uids, err := db.Meta().SystemUIDs().GetAll()
	require.NoError(t, err)
	require.Equal(t, []string{"system-1"}, uids)
	require.NoError(t, db.Local().NotifyQueue().Put([]wkdb.Message{{
		RecvPacket: wkproto.RecvPacket{
			MessageID:   1,
			ChannelID:   "channel-1",
			ChannelType: 1,
			FromUID:     "user-1",
			ClientMsgNo: "client-1",
			Timestamp:   1,
		},
	}}))
	messages, err := db.Local().NotifyQueue().List(10)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	err = db.Maintenance().RebuildDerivedState(context.Background(), 7)
	require.ErrorIs(t, err, ErrNotImplemented)
}

func TestPebbleDBRawGetRejectsWrongSlotKey(t *testing.T) {
	router, err := NewStaticBucketRouter(2)
	require.NoError(t, err)

	db, err := NewPebbleDB(PebbleDBOptions{
		DataDir:       "/wkdb-v3",
		Router:        router,
		FS:            vfs.NewMem(),
		PebbleOptions: &pebble.Options{FormatMajorVersion: pebble.FormatNewest},
	})
	require.NoError(t, err)
	require.NoError(t, db.Open())
	defer db.Close()

	_, err = db.Slots().Scope(7).Raw().Get(v3key.EncodeConversationRowKey(8, "user-1", "group-100", 2))
	require.Error(t, err)
	require.Contains(t, err.Error(), "slot mismatch")
}

func TestPebbleDBCloseIsIdempotent(t *testing.T) {
	router, err := NewStaticBucketRouter(1)
	require.NoError(t, err)

	db, err := NewPebbleDB(PebbleDBOptions{
		DataDir:       "/wkdb-v3",
		Router:        router,
		FS:            vfs.NewMem(),
		PebbleOptions: &pebble.Options{FormatMajorVersion: pebble.FormatNewest},
	})
	require.NoError(t, err)
	require.NoError(t, db.Open())
	require.NoError(t, db.Close())
	require.NoError(t, db.Close())
}

func TestErrNotImplementedWrapsSentinel(t *testing.T) {
	err := errNotImplemented("slot.users.get")
	require.True(t, errors.Is(err, ErrNotImplemented))
}
