package v3

import (
	"context"
	"fmt"
	"io"

	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
)

// SlotSnapshotScanner is the storage-facing read adapter used by slot snapshot export.
// Keeping this separate from Pebble-specific code makes the snapshot flow easy to unit test.
type SlotSnapshotScanner interface {
	ScanSlotKV(ctx context.Context, slotID uint32, fn func(key, value []byte) error) error
}

// SlotSnapshotReplacer is the storage-facing write adapter used by slot snapshot import.
// Implementations should apply the replacement atomically for one slot when possible.
type SlotSnapshotReplacer interface {
	ReplaceSlotKV(ctx context.Context, slotID uint32, fn func(put func(key, value []byte) error) error) error
}

func ExportSlotSnapshot(ctx context.Context, slotID uint32, exportedAt int64, scanner SlotSnapshotScanner, w io.Writer) (SlotSnapshotMeta, error) {
	if scanner == nil {
		return SlotSnapshotMeta{}, fmt.Errorf("slot snapshot export requires scanner")
	}
	if err := ctx.Err(); err != nil {
		return SlotSnapshotMeta{}, err
	}

	writer, err := NewSlotSnapshotWriter(w, SlotSnapshotHeader{
		Format:     SlotSnapshotFormatRawKVV1,
		SlotID:     slotID,
		ExportedAt: exportedAt,
	})
	if err != nil {
		return SlotSnapshotMeta{}, err
	}

	err = scanner.ScanSlotKV(ctx, slotID, func(key, value []byte) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := validateSlotSnapshotRecordKey(slotID, key); err != nil {
			return err
		}
		return writer.WriteRecord(key, value)
	})
	if err != nil {
		return SlotSnapshotMeta{}, fmt.Errorf("export slot snapshot: %w", err)
	}
	return writer.Close()
}

func ImportSlotSnapshot(ctx context.Context, slotID uint32, r io.Reader, meta SlotSnapshotMeta, replacer SlotSnapshotReplacer) error {
	if replacer == nil {
		return fmt.Errorf("slot snapshot import requires replacer")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if meta.SlotID != slotID {
		return fmt.Errorf("slot snapshot meta slot mismatch: got %d want %d", meta.SlotID, slotID)
	}

	reader, err := NewSlotSnapshotReader(r)
	if err != nil {
		return err
	}
	if err := validateSlotSnapshotHeader(reader.Header(), meta); err != nil {
		return err
	}

	return replacer.ReplaceSlotKV(ctx, slotID, func(put func(key, value []byte) error) error {
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			record, err := reader.Next()
			if err == io.EOF {
				return validateSlotSnapshotMeta(reader.Meta(), meta)
			}
			if err != nil {
				return err
			}
			if err := validateSlotSnapshotRecordKey(slotID, record.Key); err != nil {
				return err
			}
			if err := put(record.Key, record.Value); err != nil {
				return err
			}
		}
	})
}

func VerifySlotSnapshot(ctx context.Context, slotID uint32, r io.Reader, meta SlotSnapshotMeta) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if meta.SlotID != slotID {
		return fmt.Errorf("slot snapshot meta slot mismatch: got %d want %d", meta.SlotID, slotID)
	}

	reader, err := NewSlotSnapshotReader(r)
	if err != nil {
		return err
	}
	if err := validateSlotSnapshotHeader(reader.Header(), meta); err != nil {
		return err
	}
	return reader.Verify(meta)
}

func validateSlotSnapshotHeader(header SlotSnapshotHeader, meta SlotSnapshotMeta) error {
	if header.SlotID != meta.SlotID {
		return fmt.Errorf("slot snapshot header slot mismatch: got %d want %d", header.SlotID, meta.SlotID)
	}
	if header.Format != meta.Format {
		return fmt.Errorf("slot snapshot header format mismatch: got %d want %d", header.Format, meta.Format)
	}
	if header.ExportedAt != meta.ExportedAt {
		return fmt.Errorf("slot snapshot header exported_at mismatch: got %d want %d", header.ExportedAt, meta.ExportedAt)
	}
	return nil
}

func validateSlotSnapshotRecordKey(slotID uint32, rawKey []byte) error {
	decoded, err := v3key.Decode(rawKey)
	if err != nil {
		return fmt.Errorf("decode slot snapshot record key: %w", err)
	}
	if !decoded.Scope.IsSlot() {
		return fmt.Errorf("slot snapshot record must be slot-owned key")
	}
	if decoded.SlotID != slotID {
		return fmt.Errorf("slot snapshot record slot mismatch: got %d want %d", decoded.SlotID, slotID)
	}
	return nil
}
