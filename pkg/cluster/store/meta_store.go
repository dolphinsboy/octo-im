package store

import "github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"

type MetaStore interface {
	wkdb.SystemUidDB
	wkdb.TesterDB
	wkdb.PluginDB
}
