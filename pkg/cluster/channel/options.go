package channel

import (
	"github.com/WuKongIM/WuKongIM/pkg/cluster/icluster"
	"github.com/WuKongIM/WuKongIM/pkg/raft/raftgroup"
	"github.com/WuKongIM/WuKongIM/pkg/raft/types"
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
)

type ChannelLogStore interface {
	GetLastMsg(channelId string, channelType uint8) (wkdb.Message, error)
	AppendMessages(channelId string, channelType uint8, msgs []wkdb.Message) error
	LoadNextRangeMsgsForSize(channelId string, channelType uint8, startMessageSeq, endMessageSeq, limitSize uint64) ([]wkdb.Message, error)
	TruncateLogTo(channelId string, channelType uint8, messageSeq uint64) error
	GetChannelLastMessageSeq(channelId string, channelType uint8) (seq uint64, lastTime uint64, err error)

	SetLeaderTermStartIndex(shardNo string, term uint32, index uint64) error
	LeaderTermStartIndex(shardNo string, term uint32) (uint64, error)
	LeaderLastTerm(shardNo string) (uint32, error)
	LeaderLastTermGreaterEqThan(shardNo string, term uint32) (uint32, error)
	DeleteLeaderTermStartIndexGreaterThanTerm(shardNo string, term uint32) error
}

type Options struct {
	// 节点ID
	NodeId uint64
	// slot的接口
	Slot icluster.Slot
	// 节点接口
	Node icluster.Node
	// 频道日志和相关 raft 元数据存储
	LogDB ChannelLogStore
	// 分布式接口
	Cluster icluster.ICluster
	// api接口
	RPC icluster.RPC

	// raft group 的数量
	GroupCount int
	// 传输层
	Transport raftgroup.ITransport

	//频道最大副本数量
	ChannelMaxReplicaCount uint32

	// OnSaveConfig 保存频道配置
	OnSaveConfig func(channelId string, channelType uint8, cfg types.Config) error

	//DestoryAfterIdleTick 频道空闲多久后销毁（如果TickInterval是100ms, 那么10 * 60 * 30这个值是30分钟，具体时间根据TickInterval来定）
	DestoryAfterIdleTick int
}

func NewOptions(opt ...Option) *Options {
	opts := &Options{
		GroupCount:             100,
		ChannelMaxReplicaCount: 3,
		DestoryAfterIdleTick:   10 * 60 * 30, // 大约30分钟，如果raft的TickInterval是100ms
	}
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

func WithGroupCount(groupCount int) Option {
	return func(o *Options) {
		o.GroupCount = groupCount
	}
}

func WithTransport(transport raftgroup.ITransport) Option {
	return func(o *Options) {
		o.Transport = transport
	}
}

func WithLogDB(db ChannelLogStore) Option {
	return func(o *Options) {
		o.LogDB = db
	}
}

func WithChannelMaxReplicaCount(count uint32) Option {
	return func(o *Options) {
		o.ChannelMaxReplicaCount = count
	}
}

func WithNode(node icluster.Node) Option {
	return func(o *Options) {
		o.Node = node
	}
}

func WithCluster(cluster icluster.ICluster) Option {
	return func(o *Options) {
		o.Cluster = cluster
	}
}

func WithRPC(rpc icluster.RPC) Option {
	return func(o *Options) {
		o.RPC = rpc
	}
}

func WithOnSaveConfig(fn func(channelId string, channelType uint8, cfg types.Config) error) Option {
	return func(o *Options) {
		o.OnSaveConfig = fn
	}
}

func WithDestoryAfterIdleTick(tick int) Option {
	return func(o *Options) {
		o.DestoryAfterIdleTick = tick
	}
}
