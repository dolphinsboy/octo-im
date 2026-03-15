package v3

import (
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/stretchr/testify/require"
)

func TestPebbleSystemUIDStorePutGetDelete(t *testing.T) {
	db := newTestPebbleDBWithNow(t, time.Now)
	defer db.Close()

	store := db.Meta().SystemUIDs()
	require.NoError(t, store.PutAll([]string{"system-2", "system-1"}))

	uids, err := store.GetAll()
	require.NoError(t, err)
	require.Equal(t, []string{"system-1", "system-2"}, uids)

	require.NoError(t, store.Delete([]string{"system-1"}))
	uids, err = store.GetAll()
	require.NoError(t, err)
	require.Equal(t, []string{"system-2"}, uids)
}

func TestPebbleTesterStorePutListGetDelete(t *testing.T) {
	now := time.Unix(1710000000, 0)
	db := newTestPebbleDBWithNow(t, func() time.Time { return now })
	defer db.Close()

	store := db.Meta().Testers()
	require.NoError(t, store.Put(wkdb.Tester{
		No:   "tester-1",
		Addr: "http://127.0.0.1:9466",
	}))

	tester, err := store.Get("tester-1")
	require.NoError(t, err)
	require.Equal(t, "tester-1", tester.No)
	require.Equal(t, "http://127.0.0.1:9466", tester.Addr)
	require.NotZero(t, tester.Id)
	require.NotNil(t, tester.CreatedAt)
	require.NotNil(t, tester.UpdatedAt)
	require.Equal(t, now.UnixNano(), tester.CreatedAt.UnixNano())
	require.Equal(t, now.UnixNano(), tester.UpdatedAt.UnixNano())

	testers, err := store.List()
	require.NoError(t, err)
	require.Len(t, testers, 1)
	require.Equal(t, "tester-1", testers[0].No)

	require.NoError(t, store.Delete("tester-1"))
	_, err = store.Get("tester-1")
	require.ErrorIs(t, err, wkdb.ErrNotFound)
}

func TestPebblePluginAndPluginUserStores(t *testing.T) {
	now := time.Unix(1710000000, 0)
	db := newTestPebbleDBWithNow(t, func() time.Time { return now })
	defer db.Close()

	pluginStore := db.Meta().Plugins()
	pluginUserStore := db.Meta().PluginUsers()

	require.NoError(t, pluginStore.Put(wkdb.Plugin{
		No:             "plugin-a",
		Name:           "Plugin A",
		ConfigTemplate: []byte(`{"api_key":""}`),
		Config:         map[string]interface{}{"api_key": "secret"},
		Status:         wkdb.PluginStatusEnable,
		Version:        "1.0.0",
		Methods:        []string{"receive", "reply"},
		Priority:       9,
	}))
	require.NoError(t, pluginStore.Put(wkdb.Plugin{
		No:       "plugin-b",
		Name:     "Plugin B",
		Version:  "1.0.1",
		Priority: 3,
	}))

	plugin, err := pluginStore.Get("plugin-a")
	require.NoError(t, err)
	require.Equal(t, "Plugin A", plugin.Name)
	require.Equal(t, "secret", plugin.Config["api_key"])
	require.Equal(t, []string{"receive", "reply"}, plugin.Methods)
	require.NotNil(t, plugin.CreatedAt)
	require.NotNil(t, plugin.UpdatedAt)

	require.NoError(t, pluginStore.UpdateConfig("plugin-a", map[string]interface{}{"api_key": "updated"}))
	plugin, err = pluginStore.Get("plugin-a")
	require.NoError(t, err)
	require.Equal(t, "updated", plugin.Config["api_key"])

	plugins, err := pluginStore.List()
	require.NoError(t, err)
	require.Len(t, plugins, 2)
	require.Equal(t, "plugin-a", plugins[0].No)
	require.Equal(t, "plugin-b", plugins[1].No)

	require.NoError(t, pluginUserStore.Put([]wkdb.PluginUser{
		{PluginNo: "plugin-a", Uid: "user-1"},
		{PluginNo: "plugin-a", Uid: "user-2"},
		{PluginNo: "plugin-b", Uid: "user-1"},
	}))

	pluginUsers, err := pluginUserStore.ListByPlugin("plugin-a")
	require.NoError(t, err)
	require.Len(t, pluginUsers, 2)

	pluginUsers, err = pluginUserStore.Search(wkdb.SearchPluginUserReq{Uid: "user-1"})
	require.NoError(t, err)
	require.Len(t, pluginUsers, 2)

	pluginUsers, err = pluginUserStore.Search(wkdb.SearchPluginUserReq{PluginNo: "plugin-a", Uid: "user-2"})
	require.NoError(t, err)
	require.Len(t, pluginUsers, 1)
	require.Equal(t, "user-2", pluginUsers[0].Uid)

	require.NoError(t, pluginUserStore.Delete("plugin-a", "user-2"))
	pluginUsers, err = pluginUserStore.ListByPlugin("plugin-a")
	require.NoError(t, err)
	require.Len(t, pluginUsers, 1)

	require.NoError(t, pluginStore.Delete("plugin-b"))
	plugins, err = pluginStore.List()
	require.NoError(t, err)
	require.Len(t, plugins, 1)
	require.Equal(t, "plugin-a", plugins[0].No)
}
