package v3

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
)

type PebbleBucketManagerOptions struct {
	DataDir       string
	Router        BucketRouter
	PebbleOptions *pebble.Options
	WriteOptions  *pebble.WriteOptions
	FS            vfs.FS
	ReadOnly      bool
}

type PebbleBucket struct {
	id            uint32
	db            *pebble.DB
	snapshotStore *PebbleSlotSnapshotStore
}

func (p *PebbleBucket) ID() uint32 {
	return p.id
}

func (p *PebbleBucket) DB() *pebble.DB {
	return p.db
}

func (p *PebbleBucket) SnapshotStore() *PebbleSlotSnapshotStore {
	return p.snapshotStore
}

type PebbleBucketManager struct {
	opts    PebbleBucketManagerOptions
	buckets []*PebbleBucket
}

var _ SlotSnapshotScanner = (*PebbleBucketManager)(nil)
var _ SlotSnapshotReplacer = (*PebbleBucketManager)(nil)
var _ SlotSnapshotDeleter = (*PebbleBucketManager)(nil)

func NewPebbleBucketManager(opts PebbleBucketManagerOptions) (*PebbleBucketManager, error) {
	if opts.Router == nil {
		return nil, fmt.Errorf("pebble bucket manager requires router")
	}
	if opts.DataDir == "" {
		return nil, fmt.Errorf("pebble bucket manager requires data dir")
	}
	if opts.FS == nil {
		opts.FS = vfs.Default
	}
	if opts.PebbleOptions == nil {
		opts.PebbleOptions = &pebble.Options{}
	}
	if opts.WriteOptions == nil {
		opts.WriteOptions = pebble.Sync
	}
	return &PebbleBucketManager{opts: opts}, nil
}

func (p *PebbleBucketManager) Open() error {
	if len(p.buckets) > 0 {
		return nil
	}

	bucketRoot := filepath.Join(p.opts.DataDir, "buckets")
	if p.opts.ReadOnly {
		info, err := p.opts.FS.Stat(bucketRoot)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("bucket root is not a directory: %s", bucketRoot)
		}
	} else {
		if err := p.opts.FS.MkdirAll(bucketRoot, 0o755); err != nil {
			return err
		}
	}

	buckets := make([]*PebbleBucket, 0, p.opts.Router.BucketCount())
	for bucketID := uint32(0); bucketID < p.opts.Router.BucketCount(); bucketID++ {
		db, err := pebble.Open(p.bucketPath(bucketID), p.openOptions())
		if err != nil {
			_ = closePebbleBuckets(buckets)
			return err
		}
		snapshotStore, err := NewPebbleSlotSnapshotStore(db, PebbleSlotSnapshotStoreOptions{
			WriteOptions: p.opts.WriteOptions,
		})
		if err != nil {
			_ = db.Close()
			_ = closePebbleBuckets(buckets)
			return err
		}
		buckets = append(buckets, &PebbleBucket{
			id:            bucketID,
			db:            db,
			snapshotStore: snapshotStore,
		})
	}
	p.buckets = buckets
	return nil
}

func (p *PebbleBucketManager) Close() error {
	err := closePebbleBuckets(p.buckets)
	p.buckets = nil
	return err
}

func (p *PebbleBucketManager) BucketCount() uint32 {
	return p.opts.Router.BucketCount()
}

func (p *PebbleBucketManager) BucketForSlot(slotID uint32) uint32 {
	return p.opts.Router.BucketForSlot(slotID)
}

func (p *PebbleBucketManager) Bucket(slotID uint32) (*PebbleBucket, error) {
	if len(p.buckets) == 0 {
		return nil, fmt.Errorf("pebble bucket manager is not open")
	}
	bucketID := p.BucketForSlot(slotID)
	if bucketID >= uint32(len(p.buckets)) {
		return nil, fmt.Errorf("bucket id out of range: %d", bucketID)
	}
	return p.buckets[bucketID], nil
}

func (p *PebbleBucketManager) BucketByID(bucketID uint32) (*PebbleBucket, error) {
	if len(p.buckets) == 0 {
		return nil, fmt.Errorf("pebble bucket manager is not open")
	}
	if bucketID >= uint32(len(p.buckets)) {
		return nil, fmt.Errorf("bucket id out of range: %d", bucketID)
	}
	return p.buckets[bucketID], nil
}

func (p *PebbleBucketManager) SnapshotStore(slotID uint32) (*PebbleSlotSnapshotStore, error) {
	bucket, err := p.Bucket(slotID)
	if err != nil {
		return nil, err
	}
	return bucket.SnapshotStore(), nil
}

func (p *PebbleBucketManager) ScanSlotKV(ctx context.Context, slotID uint32, fn func(key, value []byte) error) error {
	store, err := p.SnapshotStore(slotID)
	if err != nil {
		return err
	}
	return store.ScanSlotKV(ctx, slotID, fn)
}

func (p *PebbleBucketManager) ReplaceSlotKV(ctx context.Context, slotID uint32, fn func(put func(key, value []byte) error) error) error {
	store, err := p.SnapshotStore(slotID)
	if err != nil {
		return err
	}
	return store.ReplaceSlotKV(ctx, slotID, fn)
}

func (p *PebbleBucketManager) DeleteSlotKV(ctx context.Context, slotID uint32) error {
	store, err := p.SnapshotStore(slotID)
	if err != nil {
		return err
	}
	return store.DeleteSlotKV(ctx, slotID)
}

func (p *PebbleBucketManager) bucketPath(bucketID uint32) string {
	return filepath.Join(p.opts.DataDir, "buckets", fmt.Sprintf("bucket-%03d", bucketID))
}

func (p *PebbleBucketManager) openOptions() *pebble.Options {
	opts := &pebble.Options{
		FS: p.opts.FS,
	}
	if p.opts.PebbleOptions != nil {
		cloned := *p.opts.PebbleOptions
		cloned.FS = p.opts.FS
		cloned.ReadOnly = p.opts.ReadOnly
		return &cloned
	}
	opts.ReadOnly = p.opts.ReadOnly
	return opts
}

func closePebbleBuckets(buckets []*PebbleBucket) error {
	var err error
	for _, bucket := range buckets {
		if bucket == nil || bucket.db == nil {
			continue
		}
		err = errors.Join(err, bucket.db.Close())
	}
	return err
}
