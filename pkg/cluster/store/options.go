package store

import (
	"github.com/WuKongIM/WuKongIM/pkg/cluster/icluster"
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
)

type Options struct {
	NodeId uint64 // 节点ID

	Slot icluster.Slot

	DB wkdb.DB

	UserDeviceStore     UserDeviceStore
	ConversationStore   ConversationStore
	ChannelStateStore   ChannelStateStore
	MessageStore        MessageStore
	MessageEventStore   MessageEventStore
	MetaStore           MetaStore
	MetaCommandProposer MetaCommandProposer
	AdminSearchStore    AdminSearchStore

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

func WithDB(db wkdb.DB) Option {
	return func(o *Options) {
		o.DB = db
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

func WithMessageStore(messageStore MessageStore) Option {
	return func(o *Options) {
		o.MessageStore = messageStore
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
