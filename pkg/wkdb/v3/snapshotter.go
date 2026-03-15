package v3

import (
	"context"
	"fmt"
	"io"
	"time"
)

type SlotSnapshotDeleter interface {
	DeleteSlotKV(ctx context.Context, slotID uint32) error
}

type RawKVSnapshotterOptions struct {
	Now func() int64
}

type RawKVSnapshotter struct {
	scanner  SlotSnapshotScanner
	replacer SlotSnapshotReplacer
	deleter  SlotSnapshotDeleter
	now      func() int64
}

var _ SlotSnapshotter = (*RawKVSnapshotter)(nil)

func NewRawKVSnapshotter(scanner SlotSnapshotScanner, replacer SlotSnapshotReplacer, deleter SlotSnapshotDeleter, opts RawKVSnapshotterOptions) (*RawKVSnapshotter, error) {
	if scanner == nil {
		return nil, fmt.Errorf("raw kv snapshotter requires scanner")
	}
	if replacer == nil {
		return nil, fmt.Errorf("raw kv snapshotter requires replacer")
	}
	if deleter == nil {
		return nil, fmt.Errorf("raw kv snapshotter requires deleter")
	}

	now := opts.Now
	if now == nil {
		now = func() int64 { return time.Now().Unix() }
	}

	return &RawKVSnapshotter{
		scanner:  scanner,
		replacer: replacer,
		deleter:  deleter,
		now:      now,
	}, nil
}

func (r *RawKVSnapshotter) ExportSlotKV(ctx context.Context, slotID uint32, w io.Writer) (SlotSnapshotMeta, error) {
	return ExportSlotSnapshot(ctx, slotID, r.now(), r.scanner, w)
}

func (r *RawKVSnapshotter) ImportSlotKV(ctx context.Context, slotID uint32, reader io.Reader, meta SlotSnapshotMeta) error {
	return ImportSlotSnapshot(ctx, slotID, reader, meta, r.replacer)
}

func (r *RawKVSnapshotter) DeleteSlotKV(ctx context.Context, slotID uint32) error {
	return r.deleter.DeleteSlotKV(ctx, slotID)
}

func (r *RawKVSnapshotter) VerifySlotKV(ctx context.Context, slotID uint32, reader io.Reader, meta SlotSnapshotMeta) error {
	return VerifySlotSnapshot(ctx, slotID, reader, meta)
}
