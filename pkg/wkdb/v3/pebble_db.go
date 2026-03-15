package v3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
)

type PebbleDBOptions struct {
	DataDir       string
	Router        BucketRouter
	PebbleOptions *pebble.Options
	WriteOptions  *pebble.WriteOptions
	FS            vfs.FS
	ReadOnly      bool

	ClearSlotCachesFn   func(slotID uint32)
	RebuildDerivedState func(ctx context.Context, slotID uint32) error
	SnapshotNow         func() int64
	Now                 func() time.Time
}

type PebbleDB struct {
	dataDir       string
	fs            vfs.FS
	pebbleOptions *pebble.Options
	writeOptions  *pebble.WriteOptions
	readOnly      bool
	now           func() time.Time

	buckets     *PebbleBucketManager
	metaPebble  *pebble.DB
	localPebble *pebble.DB
	snapshotter *RawKVSnapshotter
	slots       *pebbleSlotDB
	meta        MetaDB
	local       LocalDB
	maintenance *pebbleMaintenance
}

var _ DB = (*PebbleDB)(nil)

func NewPebbleDB(opts PebbleDBOptions) (*PebbleDB, error) {
	if opts.FS == nil {
		opts.FS = vfs.Default
	}
	if opts.PebbleOptions == nil {
		opts.PebbleOptions = &pebble.Options{}
	}
	if opts.WriteOptions == nil {
		opts.WriteOptions = pebble.Sync
	}
	bucketManager, err := NewPebbleBucketManager(PebbleBucketManagerOptions{
		DataDir:       opts.DataDir,
		Router:        opts.Router,
		PebbleOptions: opts.PebbleOptions,
		WriteOptions:  opts.WriteOptions,
		FS:            opts.FS,
		ReadOnly:      opts.ReadOnly,
	})
	if err != nil {
		return nil, err
	}

	snapshotter, err := NewRawKVSnapshotter(bucketManager, bucketManager, bucketManager, RawKVSnapshotterOptions{
		Now: opts.SnapshotNow,
	})
	if err != nil {
		return nil, err
	}

	db := &PebbleDB{
		dataDir:       opts.DataDir,
		fs:            opts.FS,
		pebbleOptions: opts.PebbleOptions,
		writeOptions:  opts.WriteOptions,
		readOnly:      opts.ReadOnly,
		buckets:       bucketManager,
		snapshotter:   snapshotter,
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	db.now = opts.Now
	db.meta = &pebbleMetaDB{owner: db}
	db.local = &pebbleLocalDB{owner: db}
	db.slots = &pebbleSlotDB{db: db, now: opts.Now}
	db.maintenance = &pebbleMaintenance{
		router:                opts.Router,
		clearSlotCachesFn:     opts.ClearSlotCachesFn,
		rebuildDerivedStateFn: opts.RebuildDerivedState,
	}
	return db, nil
}

func (p *PebbleDB) Open() error {
	if err := p.buckets.Open(); err != nil {
		return err
	}
	if err := p.openStandaloneStores(); err != nil {
		_ = p.buckets.Close()
		return err
	}
	return nil
}

func (p *PebbleDB) Close() error {
	var err error
	if p.localPebble != nil {
		err = errors.Join(err, p.localPebble.Close())
		p.localPebble = nil
	}
	if p.metaPebble != nil {
		err = errors.Join(err, p.metaPebble.Close())
		p.metaPebble = nil
	}
	err = errors.Join(err, p.buckets.Close())
	return err
}

func (p *PebbleDB) Slots() SlotDB {
	return p.slots
}

func (p *PebbleDB) Meta() MetaDB {
	return p.meta
}

func (p *PebbleDB) Local() LocalDB {
	return p.local
}

func (p *PebbleDB) Snapshotter() SlotSnapshotter {
	return p.snapshotter
}

func (p *PebbleDB) Maintenance() Maintenance {
	return p.maintenance
}

type pebbleSlotDB struct {
	db  *PebbleDB
	now func() time.Time
}

func (p *pebbleSlotDB) Scope(slotID uint32) SlotScope {
	return &pebbleSlotScope{
		slotID:  slotID,
		buckets: p.db.buckets,
		now:     p.now,
	}
}

type pebbleSlotScope struct {
	slotID  uint32
	buckets *PebbleBucketManager
	now     func() time.Time
}

func (p *pebbleSlotScope) SlotID() uint32 {
	return p.slotID
}

func (p *pebbleSlotScope) Users() UserStore {
	return &pebbleUserStore{
		slotID:  p.slotID,
		buckets: p.buckets,
		now:     p.now,
	}
}

func (p *pebbleSlotScope) Devices() DeviceStore {
	return &pebbleDeviceStore{
		slotID:  p.slotID,
		buckets: p.buckets,
		now:     p.now,
	}
}

func (p *pebbleSlotScope) Conversations() ConversationStore {
	return &pebbleConversationStore{
		slotID:  p.slotID,
		buckets: p.buckets,
		now:     p.now,
	}
}

func (p *pebbleSlotScope) Channels() ChannelStore {
	return &pebbleChannelStore{
		slotID:  p.slotID,
		buckets: p.buckets,
		now:     p.now,
	}
}

func (p *pebbleSlotScope) Subscribers() SubscriberStore {
	return &pebbleChannelMemberStore{
		slotID:  p.slotID,
		buckets: p.buckets,
		now:     p.now,
		table:   v3key.TableSubscriber,
		counter: channelMemberCounterSubscriber,
	}
}

func (p *pebbleSlotScope) Allowlists() AllowlistStore {
	return &pebbleChannelMemberStore{
		slotID:  p.slotID,
		buckets: p.buckets,
		now:     p.now,
		table:   v3key.TableAllowlist,
		counter: channelMemberCounterAllowlist,
	}
}

func (p *pebbleSlotScope) Denylists() DenylistStore {
	return &pebbleChannelMemberStore{
		slotID:  p.slotID,
		buckets: p.buckets,
		now:     p.now,
		table:   v3key.TableDenylist,
		counter: channelMemberCounterDenylist,
	}
}

func (p *pebbleSlotScope) ChannelClusterConfigs() ChannelClusterConfigStore {
	return &pebbleChannelClusterConfigStore{
		slotID:  p.slotID,
		buckets: p.buckets,
		now:     p.now,
	}
}

func (p *pebbleSlotScope) MessageEvents() MessageEventStore {
	return &pebbleMessageEventStore{
		slotID:  p.slotID,
		buckets: p.buckets,
	}
}

func (p *pebbleSlotScope) Raw() SlotRawKV {
	return &pebbleSlotRawKV{
		slotID:  p.slotID,
		buckets: p.buckets,
	}
}

type pebbleSlotRawKV struct {
	slotID  uint32
	buckets *PebbleBucketManager
}

func (p *pebbleSlotRawKV) Get(key []byte) ([]byte, error) {
	if err := validateSlotSnapshotRecordKey(p.slotID, key); err != nil {
		return nil, err
	}
	bucket, err := p.buckets.Bucket(p.slotID)
	if err != nil {
		return nil, err
	}
	value, closer, err := bucket.DB().Get(key)
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	return append([]byte(nil), value...), nil
}

func (p *pebbleSlotRawKV) Range(ctx context.Context, fn func(key, value []byte) error) error {
	return p.buckets.ScanSlotKV(ctx, p.slotID, fn)
}

type pebbleMaintenance struct {
	router                BucketRouter
	clearSlotCachesFn     func(slotID uint32)
	rebuildDerivedStateFn func(ctx context.Context, slotID uint32) error
}

func (p *pebbleMaintenance) BucketCount() uint32 {
	return p.router.BucketCount()
}

func (p *pebbleMaintenance) BucketForSlot(slotID uint32) uint32 {
	return p.router.BucketForSlot(slotID)
}

func (p *pebbleMaintenance) ClearSlotCaches(slotID uint32) {
	if p.clearSlotCachesFn != nil {
		p.clearSlotCachesFn(slotID)
	}
}

func (p *pebbleMaintenance) RebuildDerivedState(ctx context.Context, slotID uint32) error {
	if p.rebuildDerivedStateFn == nil {
		return errNotImplemented("maintenance.rebuild_derived_state")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return p.rebuildDerivedStateFn(ctx, slotID)
}

func (p *PebbleDB) bucket(slotID uint32) (*PebbleBucket, error) {
	if p.buckets == nil {
		return nil, fmt.Errorf("pebble db buckets are not initialized")
	}
	return p.buckets.Bucket(slotID)
}

func (p *PebbleDB) BucketByID(bucketID uint32) (*PebbleBucket, error) {
	if p.buckets == nil {
		return nil, fmt.Errorf("pebble db buckets are not initialized")
	}
	return p.buckets.BucketByID(bucketID)
}

func (p *PebbleDB) exportSlot(ctx context.Context, slotID uint32, w io.Writer) (SlotSnapshotMeta, error) {
	return p.snapshotter.ExportSlotKV(ctx, slotID, w)
}

func (p *PebbleDB) openStandaloneStores() error {
	metaWasOpen := p.metaPebble != nil
	metaDB, err := p.openStandaloneDB(filepath.Join(p.dataDir, "meta"))
	if err != nil {
		return err
	}
	localDB, err := p.openStandaloneDB(filepath.Join(p.dataDir, "local"))
	if err != nil {
		if !metaWasOpen {
			_ = metaDB.Close()
		}
		return err
	}
	p.metaPebble = metaDB
	p.localPebble = localDB
	return nil
}

func (p *PebbleDB) openStandaloneDB(path string) (*pebble.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("standalone pebble db path is required")
	}
	if path == filepath.Join(p.dataDir, "meta") && p.metaPebble != nil {
		return p.metaPebble, nil
	}
	if path == filepath.Join(p.dataDir, "local") && p.localPebble != nil {
		return p.localPebble, nil
	}
	return pebble.Open(path, p.openOptions())
}

func (p *PebbleDB) openOptions() *pebble.Options {
	if p.pebbleOptions != nil {
		cloned := *p.pebbleOptions
		cloned.FS = p.fs
		cloned.ReadOnly = p.readOnly
		return &cloned
	}
	return &pebble.Options{
		FS:       p.fs,
		ReadOnly: p.readOnly,
	}
}

func (p *PebbleDB) metaDB() (*pebble.DB, error) {
	if p.metaPebble == nil {
		return nil, fmt.Errorf("pebble meta db is not open")
	}
	return p.metaPebble, nil
}

func (p *PebbleDB) localDB() (*pebble.DB, error) {
	if p.localPebble == nil {
		return nil, fmt.Errorf("pebble local db is not open")
	}
	return p.localPebble, nil
}

func (p *PebbleDB) standaloneWriteOptions() *pebble.WriteOptions {
	if p.writeOptions != nil {
		return p.writeOptions
	}
	return pebble.Sync
}

func (p *PebbleDB) currentTime() time.Time {
	if p.now != nil {
		return p.now()
	}
	return time.Now()
}
