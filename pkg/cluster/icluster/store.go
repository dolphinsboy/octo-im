package icluster

import "github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"

type IStore interface {
	// SaveChannelClusterConfig 保存频道分布式配置
	SaveChannelClusterConfig(cfg wkdb.ChannelClusterConfig) (version uint64, err error)
}
