package v3

import (
	"context"
	"io"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
)

type DB interface {
	Open() error
	Close() error

	Slots() SlotDB
	Meta() MetaDB
	Local() LocalDB

	Snapshotter() SlotSnapshotter
	Maintenance() Maintenance
}

type SlotDB interface {
	Scope(slotID uint32) SlotScope
}

type SlotScope interface {
	SlotID() uint32

	Users() UserStore
	Devices() DeviceStore
	Conversations() ConversationStore

	Channels() ChannelStore
	Subscribers() SubscriberStore
	Allowlists() AllowlistStore
	Denylists() DenylistStore
	ChannelClusterConfigs() ChannelClusterConfigStore
	MessageEvents() MessageEventStore

	Raw() SlotRawKV
}

type MetaDB interface {
	SystemUIDs() SystemUIDStore
	Testers() TesterStore
	Plugins() PluginStore
	PluginUsers() PluginUserStore
}

type LocalDB interface {
	NotifyQueue() NotifyQueueStore
}

type SlotSnapshotMeta struct {
	SlotID      uint32
	Format      uint16
	ExportedAt  int64
	RecordCount uint64
	ByteSize    uint64
	Checksum    []byte
}

type SlotSnapshotter interface {
	ExportSlotKV(ctx context.Context, slotID uint32, w io.Writer) (SlotSnapshotMeta, error)
	ImportSlotKV(ctx context.Context, slotID uint32, r io.Reader, meta SlotSnapshotMeta) error
	DeleteSlotKV(ctx context.Context, slotID uint32) error
	VerifySlotKV(ctx context.Context, slotID uint32, r io.Reader, meta SlotSnapshotMeta) error
}

type Maintenance interface {
	BucketCount() uint32
	BucketForSlot(slotID uint32) uint32
	ClearSlotCaches(slotID uint32)
	RebuildDerivedState(ctx context.Context, slotID uint32) error
}

type SlotRawKV interface {
	Get(key []byte) ([]byte, error)
	Range(ctx context.Context, fn func(key, value []byte) error) error
}

type UserStore interface {
	Get(uid string) (wkdb.User, error)
	Exists(uid string) (bool, error)
	Put(u wkdb.User) error
	Delete(uid string) error
	Search(req wkdb.UserSearchReq) ([]wkdb.User, error)
}

type DeviceStore interface {
	Get(uid string, deviceFlag uint64) (wkdb.Device, error)
	ListByUID(uid string) ([]wkdb.Device, error)
	Put(d wkdb.Device) error
	Delete(uid string, deviceFlag uint64) error
	Search(req wkdb.DeviceSearchReq) ([]wkdb.Device, error)
}

type ConversationStore interface {
	Get(uid, channelID string, channelType uint8) (wkdb.Conversation, error)
	ListByUID(uid string) ([]wkdb.Conversation, error)
	Put(uid string, cs []wkdb.Conversation) error
	Delete(uid, channelID string, channelType uint8) error
	DeleteBatch(uid string, channels []wkdb.Channel) error
	UpdateIfSeqGreater(uid, channelID string, channelType uint8, seq uint64) error
	UpdateDeletedAt(uid, channelID string, channelType uint8, seq uint64) error
	Search(req wkdb.ConversationSearchReq) ([]wkdb.Conversation, error)
}

type ChannelStore interface {
	Get(channelID string, channelType uint8) (wkdb.ChannelInfo, error)
	Exists(channelID string, channelType uint8) (bool, error)
	Put(info wkdb.ChannelInfo) error
	Delete(channelID string, channelType uint8) error
	Search(req wkdb.ChannelSearchReq) ([]wkdb.ChannelInfo, error)
}

type SubscriberStore interface {
	List(channelID string, channelType uint8) ([]wkdb.Member, error)
	Put(channelID string, channelType uint8, members []wkdb.Member) error
	Delete(channelID string, channelType uint8, uids []string) error
	DeleteAll(channelID string, channelType uint8) error
}

type AllowlistStore interface {
	List(channelID string, channelType uint8) ([]wkdb.Member, error)
	Put(channelID string, channelType uint8, members []wkdb.Member) error
	Delete(channelID string, channelType uint8, uids []string) error
	DeleteAll(channelID string, channelType uint8) error
}

type DenylistStore interface {
	List(channelID string, channelType uint8) ([]wkdb.Member, error)
	Put(channelID string, channelType uint8, members []wkdb.Member) error
	Delete(channelID string, channelType uint8, uids []string) error
	DeleteAll(channelID string, channelType uint8) error
}

type ChannelClusterConfigStore interface {
	Get(channelID string, channelType uint8) (wkdb.ChannelClusterConfig, error)
	Put(cfg wkdb.ChannelClusterConfig) error
	Search(req wkdb.ChannelClusterConfigSearchReq) ([]wkdb.ChannelClusterConfig, error)
}

type MessageEventStore interface {
	GetState(channelID string, channelType uint8, clientMsgNo, eventKey string) (*wkdb.MessageEventState, error)
	GetStates(channelID string, channelType uint8, clientMsgNo string) ([]wkdb.MessageEventState, error)
	PutState(state wkdb.MessageEventState) error
	DeleteStates(channelID string, channelType uint8, clientMsgNo string) error
	GetSeq(channelID string, channelType uint8, clientMsgNo string) (uint64, error)
	SetSeq(channelID string, channelType uint8, clientMsgNo string, seq uint64) error
}

type SystemUIDStore interface {
	GetAll() ([]string, error)
	PutAll(uids []string) error
	Delete(uids []string) error
}

type TesterStore interface {
	Get(no string) (wkdb.Tester, error)
	List() ([]wkdb.Tester, error)
	Put(tester wkdb.Tester) error
	Delete(no string) error
}

type PluginStore interface {
	Get(no string) (wkdb.Plugin, error)
	List() ([]wkdb.Plugin, error)
	Put(plugin wkdb.Plugin) error
	Delete(no string) error
	UpdateConfig(no string, config map[string]interface{}) error
}

type PluginUserStore interface {
	ListByPlugin(pluginNo string) ([]wkdb.PluginUser, error)
	Search(req wkdb.SearchPluginUserReq) ([]wkdb.PluginUser, error)
	Put(pluginUsers []wkdb.PluginUser) error
	Delete(pluginNo, uid string) error
}

type NotifyQueueStore interface {
	Put(messages []wkdb.Message) error
	List(count int) ([]wkdb.Message, error)
	Delete(messageIDs []int64) error
	DeleteCount(count int) error
}
