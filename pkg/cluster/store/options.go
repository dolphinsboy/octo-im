package store

import (
	"github.com/WuKongIM/WuKongIM/pkg/cluster/icluster"
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
)

type Options struct {
	NodeId uint64 // 节点ID

	Slot icluster.Slot

	Lifecycle StorageLifecycle

	PrimaryKeyAllocator       PrimaryKeyAllocator
	UserDeviceStore           UserDeviceStore
	ConversationStore         ConversationStore
	ChannelStateStore         ChannelStateStore
	ChannelClusterConfigStore ChannelClusterConfigStore
	MessageQueryStore         MessageQueryStore
	MessageIndexStore         MessageIndexStore
	MessageSearchStore        MessageSearchStore
	NotifyQueueStore          NotifyQueueStore
	MessageEventStore         MessageEventStore
	MetaStore                 MetaStore
	MetaCommandProposer       MetaCommandProposer
	AdminSearchStore          AdminSearchStore

	SlotSnapshotBackend SlotSnapshotBackend

	Channel icluster.Channel

	IsCmdChannel func(channel string) bool
}

func NewOptions(opt ...Option) *Options {
	opts := &Options{}
	for _, o := range opt {
		o(opts)
	}
	return opts
}

type Option func(*Options)

type StorageLifecycle interface {
	Open() error
	Close() error
}

func WithNodeId(nodeId uint64) Option {
	return func(o *Options) {
		o.NodeId = nodeId
	}
}

func WithSlot(slot icluster.Slot) Option {
	return func(o *Options) {
		o.Slot = slot
	}
}

func WithChannel(channel icluster.Channel) Option {
	return func(o *Options) {
		o.Channel = channel
	}
}

func WithCompatDBRuntime(db wkdb.DB) Option {
	return func(o *Options) {
		if db == nil {
			return
		}
		o.Lifecycle = db
		o.PrimaryKeyAllocator = db
		o.MessageQueryStore = NewLegacyMessageQueryStore(db)
		o.MessageIndexStore = NewLegacyMessageIndexStore(db)
		o.MessageSearchStore = NewLegacyMessageSearchStore(db)
		o.NotifyQueueStore = NewLegacyNotifyQueueStore(db)
		o.MetaStore = db
		if snapshotBackend, ok := any(db).(SlotSnapshotBackend); ok {
			o.SlotSnapshotBackend = snapshotBackend
		}
		if hybrid, ok := db.(*HybridDB); ok {
			o.UserDeviceStore = NewSlotUserDeviceStore(hybrid.slotDB, hybrid.routeSlot)
			o.ConversationStore = NewSlotConversationStore(hybrid.slotDB, hybrid.slotCount, hybrid.routeSlot, func() uint64 {
				return hybrid.NextPrimaryKey()
			})
			o.ChannelStateStore = NewSlotChannelStateStore(hybrid.slotDB, hybrid.routeSlot)
			o.ChannelClusterConfigStore = NewSlotChannelClusterConfigStore(hybrid.slotDB, hybrid.slotCount, hybrid.routeSlot)
			messageStore := NewV3MessageStore(hybrid.slotDB.ChannelLogs())
			o.MessageQueryStore = messageStore
			o.MessageIndexStore = messageStore
			o.MessageSearchStore = messageStore
			o.NotifyQueueStore = NewLocalNotifyQueueStore(hybrid.slotDB)
			o.MessageEventStore = NewSlotMessageEventStore(hybrid.slotDB, hybrid.routeSlot)
			o.AdminSearchStore = NewSlotAdminSearchStore(hybrid.slotDB, hybrid.slotCount, hybrid.routeSlot)
			return
		}
		o.MessageEventStore = db
		o.UserDeviceStore = db
		o.ConversationStore = db
		o.ChannelStateStore = db
		o.ChannelClusterConfigStore = db
		o.AdminSearchStore = NewLegacyAdminSearchStore(db)
	}
}

func WithHybridRuntime(runtime *HybridRuntime) Option {
	return func(o *Options) {
		if runtime == nil {
			return
		}
		o.Lifecycle = runtime.Lifecycle
		o.SlotSnapshotBackend = runtime.SlotSnapshotBackend
		o.PrimaryKeyAllocator = runtime.PrimaryKeyAllocator
		o.UserDeviceStore = runtime.UserDeviceStore
		o.ConversationStore = runtime.ConversationStore
		o.ChannelStateStore = runtime.ChannelStateStore
		o.ChannelClusterConfigStore = runtime.ChannelClusterConfigStore
		o.MessageQueryStore = runtime.MessageQueryStore
		o.MessageIndexStore = runtime.MessageIndexStore
		o.MessageSearchStore = runtime.MessageSearchStore
		o.NotifyQueueStore = runtime.NotifyQueueStore
		o.MessageEventStore = runtime.MessageEventStore
		o.MetaStore = runtime.MetaStore
		o.AdminSearchStore = runtime.AdminSearchStore
	}
}

func WithStorageLifecycle(lifecycle StorageLifecycle) Option {
	return func(o *Options) {
		o.Lifecycle = lifecycle
	}
}

func WithPrimaryKeyAllocator(primaryKeyAllocator PrimaryKeyAllocator) Option {
	return func(o *Options) {
		o.PrimaryKeyAllocator = primaryKeyAllocator
	}
}

func WithUserDeviceStore(userDeviceStore UserDeviceStore) Option {
	return func(o *Options) {
		o.UserDeviceStore = userDeviceStore
	}
}

func WithConversationStore(conversationStore ConversationStore) Option {
	return func(o *Options) {
		o.ConversationStore = conversationStore
	}
}

func WithChannelStateStore(channelStateStore ChannelStateStore) Option {
	return func(o *Options) {
		o.ChannelStateStore = channelStateStore
	}
}

func WithChannelClusterConfigStore(channelClusterConfigStore ChannelClusterConfigStore) Option {
	return func(o *Options) {
		o.ChannelClusterConfigStore = channelClusterConfigStore
	}
}

func WithMessageQueryStore(messageQueryStore MessageQueryStore) Option {
	return func(o *Options) {
		o.MessageQueryStore = messageQueryStore
	}
}

func WithMessageIndexStore(messageIndexStore MessageIndexStore) Option {
	return func(o *Options) {
		o.MessageIndexStore = messageIndexStore
	}
}

func WithMessageSearchStore(messageSearchStore MessageSearchStore) Option {
	return func(o *Options) {
		o.MessageSearchStore = messageSearchStore
	}
}

func WithNotifyQueueStore(notifyQueueStore NotifyQueueStore) Option {
	return func(o *Options) {
		o.NotifyQueueStore = notifyQueueStore
	}
}

func WithMessageEventStore(messageEventStore MessageEventStore) Option {
	return func(o *Options) {
		o.MessageEventStore = messageEventStore
	}
}

func WithMetaStore(metaStore MetaStore) Option {
	return func(o *Options) {
		o.MetaStore = metaStore
	}
}

func WithMetaCommandProposer(metaCommandProposer MetaCommandProposer) Option {
	return func(o *Options) {
		o.MetaCommandProposer = metaCommandProposer
	}
}

func WithAdminSearchStore(adminSearchStore AdminSearchStore) Option {
	return func(o *Options) {
		o.AdminSearchStore = adminSearchStore
	}
}

func WithSlotSnapshotBackend(backend SlotSnapshotBackend) Option {
	return func(o *Options) {
		o.SlotSnapshotBackend = backend
	}
}

func WithIsCmdChannel(isCmdChannel func(channel string) bool) Option {
	return func(o *Options) {
		o.IsCmdChannel = isCmdChannel
	}
}
