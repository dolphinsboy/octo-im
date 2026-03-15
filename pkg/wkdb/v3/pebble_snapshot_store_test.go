package v3

import (
	"bytes"
	"context"
	"testing"

	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/stretchr/testify/require"
)

func TestPebbleSlotSnapshotStoreScanReplaceDelete(t *testing.T) {
	db := newTestPebbleDB(t)
	defer db.Close()

	store, err := NewPebbleSlotSnapshotStore(db, PebbleSlotSnapshotStoreOptions{
		WriteOptions: pebble.NoSync,
	})
	require.NoError(t, err)

	slot7Row := v3key.EncodeConversationRowKey(7, "user-1", "group-100", 2)
	slot7Idx := v3key.EncodeConversationUpdatedAtSecondIndexKey(7, "user-1", 100, "group-100", 2)
	slot8Row := v3key.EncodeConversationRowKey(8, "user-2", "group-200", 2)

	require.NoError(t, db.Set(slot7Row, []byte("row"), pebble.NoSync))
	require.NoError(t, db.Set(slot7Idx, []byte("idx"), pebble.NoSync))
	require.NoError(t, db.Set(slot8Row, []byte("other"), pebble.NoSync))

	records := make([]SlotSnapshotRecord, 0)
	err = store.ScanSlotKV(context.Background(), 7, func(key, value []byte) error {
		records = append(records, SlotSnapshotRecord{Key: key, Value: value})
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []SlotSnapshotRecord{
		{Key: slot7Row, Value: []byte("row")},
		{Key: slot7Idx, Value: []byte("idx")},
	}, records)

	err = store.ReplaceSlotKV(context.Background(), 7, func(put func(key, value []byte) error) error {
		require.NoError(t, put(v3key.EncodeConversationRowKey(7, "user-9", "group-900", 2), []byte("new-row")))
		require.NoError(t, put(v3key.EncodeMessageEventSeqAuxKey(7, "group-900", 2, "client-9"), []byte("new-aux")))
		return nil
	})
	require.NoError(t, err)

	records = records[:0]
	err = store.ScanSlotKV(context.Background(), 7, func(key, value []byte) error {
		records = append(records, SlotSnapshotRecord{Key: key, Value: value})
		return nil
	})
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.Equal(t, []byte("other"), mustGetPebbleValue(t, db, slot8Row))

	require.NoError(t, store.DeleteSlotKV(context.Background(), 7))
	records = records[:0]
	err = store.ScanSlotKV(context.Background(), 7, func(key, value []byte) error {
		records = append(records, SlotSnapshotRecord{Key: key, Value: value})
		return nil
	})
	require.NoError(t, err)
	require.Empty(t, records)
	require.Equal(t, []byte("other"), mustGetPebbleValue(t, db, slot8Row))
}

func TestRawKVSnapshotterWithPebbleStore(t *testing.T) {
	db := newTestPebbleDB(t)
	defer db.Close()

	store, err := NewPebbleSlotSnapshotStore(db, PebbleSlotSnapshotStoreOptions{
		WriteOptions: pebble.NoSync,
	})
	require.NoError(t, err)

	slot7Row := v3key.EncodeConversationRowKey(7, "user-1", "group-100", 2)
	slot7Idx := v3key.EncodeConversationUpdatedAtSecondIndexKey(7, "user-1", 100, "group-100", 2)
	slot8Row := v3key.EncodeConversationRowKey(8, "user-2", "group-200", 2)
	require.NoError(t, db.Set(slot7Row, []byte("row"), pebble.NoSync))
	require.NoError(t, db.Set(slot7Idx, []byte("idx"), pebble.NoSync))
	require.NoError(t, db.Set(slot8Row, []byte("other"), pebble.NoSync))

	snapshotter, err := NewRawKVSnapshotter(store, store, store, RawKVSnapshotterOptions{
		Now: func() int64 { return 1710000000 },
	})
	require.NoError(t, err)

	var buf bytes.Buffer
	meta, err := snapshotter.ExportSlotKV(context.Background(), 7, &buf)
	require.NoError(t, err)
	require.Equal(t, int64(1710000000), meta.ExportedAt)
	require.Equal(t, uint64(2), meta.RecordCount)

	require.NoError(t, snapshotter.VerifySlotKV(context.Background(), 7, bytes.NewReader(buf.Bytes()), meta))

	require.NoError(t, store.ReplaceSlotKV(context.Background(), 7, func(put func(key, value []byte) error) error {
		return put(v3key.EncodeConversationRowKey(7, "stale-user", "stale-channel", 2), []byte("stale"))
	}))

	require.NoError(t, snapshotter.ImportSlotKV(context.Background(), 7, bytes.NewReader(buf.Bytes()), meta))
	require.Equal(t, []byte("other"), mustGetPebbleValue(t, db, slot8Row))

	records := make([]SlotSnapshotRecord, 0)
	err = store.ScanSlotKV(context.Background(), 7, func(key, value []byte) error {
		records = append(records, SlotSnapshotRecord{Key: key, Value: value})
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []SlotSnapshotRecord{
		{Key: slot7Row, Value: []byte("row")},
		{Key: slot7Idx, Value: []byte("idx")},
	}, records)
}

func newTestPebbleDB(t *testing.T) *pebble.DB {
	t.Helper()

	db, err := pebble.Open("test", &pebble.Options{
		FS:                 vfs.NewMem(),
		FormatMajorVersion: pebble.FormatNewest,
	})
	require.NoError(t, err)
	return db
}

func mustGetPebbleValue(t *testing.T, db *pebble.DB, key []byte) []byte {
	t.Helper()

	value, closer, err := db.Get(key)
	require.NoError(t, err)
	defer closer.Close()
	return append([]byte(nil), value...)
}
