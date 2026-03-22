package store

import (
	"context"
	"path"

	"github.com/WuKongIM/WuKongIM/pkg/cluster/channel"
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
	"github.com/WuKongIM/WuKongIM/pkg/wkutil"
	"github.com/cockroachdb/pebble"
)

type HybridRuntimeOptions struct {
	DataDir             string
	NodeID              uint64
	SlotCount           uint32
	ShardNum            int
	MemTableSize        int
	ClearSlotCaches     func(slotID uint32)
	RebuildDerivedState func(ctx context.Context, slotID uint32) error
}

type HybridRuntime struct {
	Lifecycle                 StorageLifecycle
	SlotSnapshotBackend       SlotSnapshotBackend
	PrimaryKeyAllocator       PrimaryKeyAllocator
	UserDeviceStore           UserDeviceStore
	ConversationStore         ConversationStore
	ChannelStateStore         ChannelStateStore
	ChannelClusterConfigStore ChannelClusterConfigStore
	ChannelLogStore           channel.ChannelLogStore
	MessageQueryStore         MessageQueryStore
	MessageIndexStore         MessageIndexStore
	MessageSearchStore        MessageSearchStore
	NotifyQueueStore          NotifyQueueStore
	MessageEventStore         MessageEventStore
	MetaStore                 MetaStore
	AdminSearchStore          AdminSearchStore
}

func NewHybridRuntime(opts HybridRuntimeOptions) (*HybridRuntime, error) {
	legacyDB := wkdb.NewWukongDB(
		wkdb.NewOptions(
			wkdb.WithShardNum(opts.ShardNum),
			wkdb.WithDir(path.Join(opts.DataDir, "db")),
			wkdb.WithNodeId(opts.NodeID),
			wkdb.WithMemTableSize(opts.MemTableSize),
			wkdb.WithSlotCount(int(opts.SlotCount)),
		),
	)

	router, err := wkdbv3.NewStaticBucketRouter(uint32(opts.ShardNum))
	if err != nil {
		return nil, err
	}

	clearSlotCachesFn := opts.ClearSlotCaches
	if clearSlotCachesFn == nil {
		clearSlotCachesFn = func(slotID uint32) {}
	}
	rebuildDerivedStateFn := opts.RebuildDerivedState
	if rebuildDerivedStateFn == nil {
		rebuildDerivedStateFn = func(ctx context.Context, slotID uint32) error { return nil }
	}

	slotStateDB, err := wkdbv3.NewPebbleDB(wkdbv3.PebbleDBOptions{
		DataDir:             path.Join(opts.DataDir, "dbv3"),
		Router:              router,
		WriteOptions:        pebble.Sync,
		PebbleOptions:       &pebble.Options{FormatMajorVersion: pebble.FormatNewest, MemTableSize: opts.MemTableSize},
		ClearSlotCachesFn:   clearSlotCachesFn,
		RebuildDerivedState: rebuildDerivedStateFn,
	})
	if err != nil {
		return nil, err
	}

	routeSlot := func(key string) uint32 {
		return wkutil.GetSlotNum(int(opts.SlotCount), key)
	}

	db := NewHybridDB(legacyDB, slotStateDB, opts.SlotCount, routeSlot)
	primaryKeyAllocator, err := NewSnowflakePrimaryKeyAllocator(opts.NodeID)
	if err != nil {
		return nil, err
	}
	messageStore := NewV3MessageStore(slotStateDB.ChannelLogs())
	return &HybridRuntime{
		Lifecycle:                 db,
		SlotSnapshotBackend:       db,
		PrimaryKeyAllocator:       primaryKeyAllocator,
		UserDeviceStore:           NewSlotUserDeviceStore(slotStateDB, routeSlot),
		ConversationStore:         NewSlotConversationStore(slotStateDB, opts.SlotCount, routeSlot, primaryKeyAllocator.NextPrimaryKey),
		ChannelStateStore:         NewSlotChannelStateStore(slotStateDB, routeSlot),
		ChannelClusterConfigStore: NewSlotChannelClusterConfigStore(slotStateDB, opts.SlotCount, routeSlot),
		ChannelLogStore:           messageStore,
		MessageQueryStore:         messageStore,
		MessageIndexStore:         messageStore,
		MessageSearchStore:        messageStore,
		NotifyQueueStore:          NewLocalNotifyQueueStore(slotStateDB),
		MessageEventStore:         NewSlotMessageEventStore(slotStateDB, routeSlot),
		MetaStore:                 db,
		AdminSearchStore:          NewSlotAdminSearchStore(slotStateDB, opts.SlotCount, routeSlot),
	}, nil
}
