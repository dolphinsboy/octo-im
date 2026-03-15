package v3

import (
	"bytes"
	"context"
	"testing"

	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	"github.com/stretchr/testify/require"
)

type fakeSlotSnapshotScanner struct {
	records map[uint32][]SlotSnapshotRecord
}

func (f *fakeSlotSnapshotScanner) ScanSlotKV(ctx context.Context, slotID uint32, fn func(key, value []byte) error) error {
	for _, record := range f.records[slotID] {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := fn(record.Key, record.Value); err != nil {
			return err
		}
	}
	return nil
}

type fakeSlotSnapshotReplacer struct {
	records map[uint32][]SlotSnapshotRecord
}

func (f *fakeSlotSnapshotReplacer) ReplaceSlotKV(ctx context.Context, slotID uint32, fn func(put func(key, value []byte) error) error) error {
	pending := make([]SlotSnapshotRecord, 0)
	err := fn(func(key, value []byte) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		pending = append(pending, SlotSnapshotRecord{
			Key:   append([]byte(nil), key...),
			Value: append([]byte(nil), value...),
		})
		return nil
	})
	if err != nil {
		return err
	}
	if f.records == nil {
		f.records = make(map[uint32][]SlotSnapshotRecord)
	}
	f.records[slotID] = pending
	return nil
}

func TestExportImportSlotSnapshotRoundTrip(t *testing.T) {
	sourceRecords := []SlotSnapshotRecord{
		{
			Key:   v3key.EncodeConversationRowKey(7, "user-1", "group-100", 2),
			Value: []byte("row"),
		},
		{
			Key:   v3key.EncodeConversationUpdatedAtSecondIndexKey(7, "user-1", 100, "group-100", 2),
			Value: []byte("idx"),
		},
		{
			Key:   v3key.EncodeMessageEventSeqAuxKey(7, "group-100", 2, "client-1"),
			Value: []byte("aux"),
		},
	}
	scanner := &fakeSlotSnapshotScanner{
		records: map[uint32][]SlotSnapshotRecord{
			7: sourceRecords,
		},
	}

	var buf bytes.Buffer
	meta, err := ExportSlotSnapshot(context.Background(), 7, 1710000000, scanner, &buf)
	require.NoError(t, err)
	require.Equal(t, uint64(3), meta.RecordCount)

	require.NoError(t, VerifySlotSnapshot(context.Background(), 7, bytes.NewReader(buf.Bytes()), meta))

	replacer := &fakeSlotSnapshotReplacer{
		records: map[uint32][]SlotSnapshotRecord{
			7: {{
				Key:   v3key.EncodeConversationRowKey(7, "old-user", "old-channel", 2),
				Value: []byte("stale"),
			}},
			8: {{
				Key:   v3key.EncodeConversationRowKey(8, "other-user", "other-channel", 2),
				Value: []byte("other"),
			}},
		},
	}

	err = ImportSlotSnapshot(context.Background(), 7, bytes.NewReader(buf.Bytes()), meta, replacer)
	require.NoError(t, err)
	require.Equal(t, sourceRecords, replacer.records[7])
	require.Len(t, replacer.records[8], 1)
	require.Equal(t, []byte("other"), replacer.records[8][0].Value)
}

func TestExportSlotSnapshotRejectsWrongSlotRecord(t *testing.T) {
	scanner := &fakeSlotSnapshotScanner{
		records: map[uint32][]SlotSnapshotRecord{
			7: {{
				Key:   v3key.EncodeConversationRowKey(8, "user-1", "group-100", 2),
				Value: []byte("wrong-slot"),
			}},
		},
	}

	var buf bytes.Buffer
	_, err := ExportSlotSnapshot(context.Background(), 7, 1710000000, scanner, &buf)
	require.Error(t, err)
	require.Contains(t, err.Error(), "slot mismatch")
}

func TestImportSlotSnapshotRejectsWrongSlotRecord(t *testing.T) {
	var buf bytes.Buffer
	writer, err := NewSlotSnapshotWriter(&buf, SlotSnapshotHeader{
		SlotID:     7,
		ExportedAt: 1710000000,
	})
	require.NoError(t, err)
	require.NoError(t, writer.WriteRecord(v3key.EncodeConversationRowKey(8, "user-1", "group-100", 2), []byte("wrong-slot")))
	meta, err := writer.Close()
	require.NoError(t, err)

	replacer := &fakeSlotSnapshotReplacer{
		records: map[uint32][]SlotSnapshotRecord{
			7: {{
				Key:   v3key.EncodeConversationRowKey(7, "old-user", "old-channel", 2),
				Value: []byte("stale"),
			}},
		},
	}

	err = ImportSlotSnapshot(context.Background(), 7, bytes.NewReader(buf.Bytes()), meta, replacer)
	require.Error(t, err)
	require.Contains(t, err.Error(), "slot mismatch")
	require.Len(t, replacer.records[7], 1)
	require.Equal(t, []byte("stale"), replacer.records[7][0].Value)
}
