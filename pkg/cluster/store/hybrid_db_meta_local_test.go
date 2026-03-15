package store

import (
	"context"
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/trace"
	wkdb "github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
	wkproto "github.com/WuKongIM/WuKongIMGoProto"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/stretchr/testify/require"
)

func TestHybridDBRoutesMetaAndLocalStoresToV3(t *testing.T) {
	hybrid, legacy := newHybridMetaLocalTestDB(t)

	require.NoError(t, hybrid.AddSystemUids([]string{"system-2", "system-1"}))
	uids, err := hybrid.GetSystemUids()
	require.NoError(t, err)
	require.Equal(t, []string{"system-1", "system-2"}, uids)
	legacyUids, err := legacy.GetSystemUids()
	require.NoError(t, err)
	require.Nil(t, legacyUids)

	require.NoError(t, hybrid.AddOrUpdateTester(wkdb.Tester{
		No:   "tester-1",
		Addr: "http://127.0.0.1:9466",
	}))
	tester, err := hybrid.GetTester("tester-1")
	require.NoError(t, err)
	require.Equal(t, "tester-1", tester.No)
	legacyTester, err := legacy.GetTester("tester-1")
	require.NoError(t, err)
	require.Empty(t, legacyTester.No)

	now := time.Unix(1710000000, 0)
	require.NoError(t, hybrid.AddOrUpdatePlugin(wkdb.Plugin{
		No:        "plugin-a",
		Name:      "Plugin A",
		Priority:  1,
		CreatedAt: &now,
		UpdatedAt: &now,
	}))
	require.NoError(t, hybrid.AddOrUpdatePlugin(wkdb.Plugin{
		No:        "plugin-b",
		Name:      "Plugin B",
		Priority:  9,
		CreatedAt: &now,
		UpdatedAt: &now,
	}))
	plugins, err := hybrid.GetPlugins()
	require.NoError(t, err)
	require.Len(t, plugins, 2)
	legacyPlugins, err := legacy.GetPlugins()
	require.NoError(t, err)
	require.Empty(t, legacyPlugins)

	require.NoError(t, hybrid.AddOrUpdatePluginUsers([]wkdb.PluginUser{
		{PluginNo: "plugin-a", Uid: "user-1"},
		{PluginNo: "plugin-b", Uid: "user-1"},
		{PluginNo: "plugin-a", Uid: "user-2"},
	}))
	pluginUsers, err := hybrid.GetPluginUsers("plugin-a")
	require.NoError(t, err)
	require.Len(t, pluginUsers, 2)
	pluginUsers, err = hybrid.SearchPluginUsers(wkdb.SearchPluginUserReq{Uid: "user-1"})
	require.NoError(t, err)
	require.Len(t, pluginUsers, 2)
	highestPriorityPlugin, err := hybrid.GetHighestPriorityPluginByUid("user-1")
	require.NoError(t, err)
	require.Equal(t, "plugin-b", highestPriorityPlugin)
	exists, err := hybrid.ExistPluginByUid("user-1")
	require.NoError(t, err)
	require.True(t, exists)

	require.NoError(t, hybrid.UpdatePluginConfig("plugin-a", map[string]interface{}{"api_key": "secret"}))
	config, err := hybrid.GetPluginConfig("plugin-a")
	require.NoError(t, err)
	require.Equal(t, "secret", config["api_key"])

	require.NoError(t, hybrid.AppendMessageOfNotifyQueue([]wkdb.Message{
		{
			RecvPacket: wkproto.RecvPacket{
				MessageID:   1,
				ChannelID:   "channel-1",
				ChannelType: 1,
				FromUID:     "user-1",
				ClientMsgNo: "client-1",
				Timestamp:   100,
				Payload:     []byte("hello"),
			},
		},
	}))
	messages, err := hybrid.GetMessagesOfNotifyQueue(10)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	legacyMessages, err := legacy.GetMessagesOfNotifyQueue(10)
	require.NoError(t, err)
	require.Empty(t, legacyMessages)

	require.NoError(t, hybrid.RemoveMessagesOfNotifyQueueCount(1))
	messages, err = hybrid.GetMessagesOfNotifyQueue(10)
	require.NoError(t, err)
	require.Nil(t, messages)

	require.NoError(t, hybrid.RemovePluginUser("plugin-a", "user-2"))
	pluginUsers, err = hybrid.GetPluginUsers("plugin-a")
	require.NoError(t, err)
	require.Len(t, pluginUsers, 1)

	require.NoError(t, hybrid.RemoveTester("tester-1"))
	tester, err = hybrid.GetTester("tester-1")
	require.NoError(t, err)
	require.Empty(t, tester.No)
}

func TestStorePluginWrappersUseHybridDB(t *testing.T) {
	hybrid, _ := newHybridMetaLocalTestDB(t)
	s := New(NewOptions(WithDB(hybrid)))

	now := time.Unix(1710000000, 0)
	require.NoError(t, s.AddOrUpdatePlugin(wkdb.Plugin{
		No:        "plugin-a",
		Name:      "Plugin A",
		Priority:  1,
		CreatedAt: &now,
		UpdatedAt: &now,
	}))
	require.NoError(t, s.AddOrUpdatePlugin(wkdb.Plugin{
		No:        "plugin-b",
		Name:      "Plugin B",
		Priority:  8,
		CreatedAt: &now,
		UpdatedAt: &now,
	}))
	require.NoError(t, s.AddOrUpdatePluginUsers([]wkdb.PluginUser{
		{PluginNo: "plugin-a", Uid: "user-1"},
		{PluginNo: "plugin-b", Uid: "user-1"},
	}))

	plugins, err := s.GetPlugins()
	require.NoError(t, err)
	require.Len(t, plugins, 2)

	plugin, err := s.GetPlugin("plugin-a")
	require.NoError(t, err)
	require.Equal(t, "Plugin A", plugin.Name)

	exists, err := s.ExistPluginByUid("user-1")
	require.NoError(t, err)
	require.True(t, exists)

	pluginNo, err := s.GetHighestPriorityPluginByUid("user-1")
	require.NoError(t, err)
	require.Equal(t, "plugin-b", pluginNo)

	pluginUsers, err := s.SearchPluginUsers(wkdb.SearchPluginUserReq{Uid: "user-1"})
	require.NoError(t, err)
	require.Len(t, pluginUsers, 2)

	require.NoError(t, s.UpdatePluginConfig("plugin-a", map[string]interface{}{"api_key": "secret"}))
	config, err := s.GetPluginConfig("plugin-a")
	require.NoError(t, err)
	require.Equal(t, "secret", config["api_key"])

	require.NoError(t, s.DeletePlugin("plugin-b"))
	plugins, err = s.GetPlugins()
	require.NoError(t, err)
	require.Len(t, plugins, 1)
}

func newHybridMetaLocalTestDB(t *testing.T) (*HybridDB, wkdb.DB) {
	t.Helper()

	traceObj := trace.New(
		context.Background(),
		trace.NewOptions(
			trace.WithServiceName("test"),
			trace.WithServiceHostName("host"),
		),
	)
	trace.SetGlobalTrace(traceObj)

	legacy := wkdb.NewWukongDB(wkdb.NewOptions(
		wkdb.WithDir(t.TempDir()),
		wkdb.WithShardNum(1),
	))

	router, err := wkdbv3.NewStaticBucketRouter(2)
	require.NoError(t, err)

	slotDB, err := wkdbv3.NewPebbleDB(wkdbv3.PebbleDBOptions{
		DataDir:      "/wkdb-v3",
		Router:       router,
		FS:           vfs.NewMem(),
		WriteOptions: pebble.NoSync,
		PebbleOptions: &pebble.Options{
			FormatMajorVersion: pebble.FormatNewest,
		},
		SnapshotNow: func() int64 { return 1710000000 },
		Now:         func() time.Time { return time.Unix(1710000000, 0) },
	})
	require.NoError(t, err)

	hybrid := NewHybridDB(legacy, slotDB, 1, func(string) uint32 { return 0 })
	require.NoError(t, hybrid.Open())
	t.Cleanup(func() {
		require.NoError(t, hybrid.Close())
	})
	return hybrid, legacy
}
