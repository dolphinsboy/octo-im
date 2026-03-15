package slot

import (
	"github.com/WuKongIM/WuKongIM/pkg/cluster/icluster"
	"github.com/WuKongIM/WuKongIM/pkg/raft/raftgroup"
	"github.com/WuKongIM/WuKongIM/pkg/raft/types"
)

type Options struct {
	// 节点Id
	NodeId uint64
	// 数据目录
	DataDir string
	// 槽位数据库分片数量
	SlotDbShardNum int
	// 节点接口
	Node icluster.Node
	// 传输层
	Transport raftgroup.ITransport
	// 槽数量
	SlotCount uint32
	// api接口
	RPC icluster.RPC
	// OnApply 应用日志回调
	OnApply func(slotId uint32, logs []types.Log) error

	// OnSaveConfig 保存槽配置
	OnSaveConfig func(slotId uint32, cfg types.Config) error

	// OnCreateSnapshot 创建状态机快照回调（序列化当前状态到 []byte）
	OnCreateSnapshot func(slotId uint32) ([]byte, error)
	// OnApplySnapshot 恢复状态机快照回调（从 []byte 恢复状态）
	OnApplySnapshot func(slotId uint32, data []byte) error

	// CompactionEnabled 是否启用 slot raft 日志压缩
	CompactionEnabled bool
	// CompactionIntervalTick 压缩检查间隔（tick 次数）
	CompactionIntervalTick int
	// CompactionMinLogCount 触发压缩的最小日志数
	CompactionMinLogCount uint64
	// CompactionRetainCount 压缩后保留的最近日志数
	CompactionRetainCount uint64
}

func NewOptions(opt ...Option) *Options {
	defaultOpts := &Options{
		DataDir:                "clusterdata",
		SlotDbShardNum:         8,
		CompactionEnabled:      false,
		CompactionIntervalTick: 2000,
		CompactionMinLogCount:  10000,
		CompactionRetainCount:  1000,
	}
	for _, o := range opt {
		o(defaultOpts)
	}

	return defaultOpts
}

type Option func(*Options)

func WithNodeId(nodeId uint64) Option {
	return func(o *Options) {
		o.NodeId = nodeId
	}
}

func WithDataDir(dataDir string) Option {
	return func(o *Options) {
		o.DataDir = dataDir
	}
}

func WithSlotDbShardNum(slotDbShardNum int) Option {
	return func(o *Options) {
		o.SlotDbShardNum = slotDbShardNum
	}
}

func WithTransport(transport raftgroup.ITransport) Option {
	return func(o *Options) {
		o.Transport = transport
	}
}

func WithNode(node icluster.Node) Option {
	return func(o *Options) {
		o.Node = node
	}
}

func WithSlotCount(slotCount uint32) Option {
	return func(o *Options) {
		o.SlotCount = slotCount
	}
}

func WithOnApply(onApply func(slotId uint32, logs []types.Log) error) Option {
	return func(o *Options) {
		o.OnApply = onApply
	}
}

func WithOnSaveConfig(onSaveConfig func(slotId uint32, cfg types.Config) error) Option {
	return func(o *Options) {
		o.OnSaveConfig = onSaveConfig
	}
}

func WithOnCreateSnapshot(onCreateSnapshot func(slotId uint32) ([]byte, error)) Option {
	return func(o *Options) {
		o.OnCreateSnapshot = onCreateSnapshot
	}
}

func WithOnApplySnapshot(onApplySnapshot func(slotId uint32, data []byte) error) Option {
	return func(o *Options) {
		o.OnApplySnapshot = onApplySnapshot
	}
}

func WithCompactionEnabled(enabled bool) Option {
	return func(o *Options) {
		o.CompactionEnabled = enabled
	}
}

func WithCompactionIntervalTick(interval int) Option {
	return func(o *Options) {
		o.CompactionIntervalTick = interval
	}
}

func WithCompactionMinLogCount(count uint64) Option {
	return func(o *Options) {
		o.CompactionMinLogCount = count
	}
}

func WithCompactionRetainCount(count uint64) Option {
	return func(o *Options) {
		o.CompactionRetainCount = count
	}
}

func WithRPC(rpc icluster.RPC) Option {
	return func(o *Options) {
		o.RPC = rpc
	}
}
