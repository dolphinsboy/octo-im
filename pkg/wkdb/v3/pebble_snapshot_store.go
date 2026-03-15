package v3

import (
	"bytes"
	"context"
	"fmt"

	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	"github.com/cockroachdb/pebble"
)

type PebbleSlotSnapshotStoreOptions struct {
	WriteOptions *pebble.WriteOptions
}

type PebbleSlotSnapshotStore struct {
	db           *pebble.DB
	writeOptions *pebble.WriteOptions
}

func NewPebbleSlotSnapshotStore(db *pebble.DB, opts PebbleSlotSnapshotStoreOptions) (*PebbleSlotSnapshotStore, error) {
	if db == nil {
		return nil, fmt.Errorf("pebble slot snapshot store requires db")
	}
	writeOptions := opts.WriteOptions
	if writeOptions == nil {
		writeOptions = pebble.Sync
	}
	return &PebbleSlotSnapshotStore{
		db:           db,
		writeOptions: writeOptions,
	}, nil
}

func (p *PebbleSlotSnapshotStore) ScanSlotKV(ctx context.Context, slotID uint32, fn func(key, value []byte) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	lower, upper := v3key.SlotRange(slotID)

	snapshot := p.db.NewSnapshot()
	defer snapshot.Close()

	iter := snapshot.NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		if err := ctx.Err(); err != nil {
			return err
		}
		key := append([]byte(nil), iter.Key()...)
		value := append([]byte(nil), iter.Value()...)
		if !bytes.HasPrefix(key, lower) {
			return fmt.Errorf("scan slot kv encountered out-of-range key")
		}
		if err := fn(key, value); err != nil {
			return err
		}
	}
	return iter.Error()
}

func (p *PebbleSlotSnapshotStore) ReplaceSlotKV(ctx context.Context, slotID uint32, fn func(put func(key, value []byte) error) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	lower, upper := v3key.SlotRange(slotID)

	batch := p.db.NewBatch()
	defer batch.Close()

	if err := batch.DeleteRange(lower, upper, p.writeOptions); err != nil {
		return err
	}

	err := fn(func(key, value []byte) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := validateSlotSnapshotRecordKey(slotID, key); err != nil {
			return err
		}
		return batch.Set(key, value, p.writeOptions)
	})
	if err != nil {
		return err
	}
	return batch.Commit(p.writeOptions)
}

func (p *PebbleSlotSnapshotStore) DeleteSlotKV(ctx context.Context, slotID uint32) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	lower, upper := v3key.SlotRange(slotID)

	batch := p.db.NewBatch()
	defer batch.Close()

	if err := batch.DeleteRange(lower, upper, p.writeOptions); err != nil {
		return err
	}
	return batch.Commit(p.writeOptions)
}
