package store

import (
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/stretchr/testify/require"
)

type stubMetaStore struct {
	systemUIDs               []string
	tester                   wkdb.Tester
	testers                  []wkdb.Tester
	plugin                   wkdb.Plugin
	plugins                  []wkdb.Plugin
	pluginUsers              []wkdb.PluginUser
	pluginConfig             map[string]interface{}
	highestPriorityPluginNo  string
	existPluginByUID         bool
	lastAddedSystemUIDs      []string
	lastRemovedSystemUIDs    []string
	lastPutTester            wkdb.Tester
	lastRemovedTesterNo      string
	lastPutPlugin            wkdb.Plugin
	lastDeletedPluginNo      string
	lastPluginConfigNo       string
	lastUpdatedPluginConfig  map[string]interface{}
	lastPluginUserSearchReq  wkdb.SearchPluginUserReq
	lastPluginUsersPut       []wkdb.PluginUser
	lastRemovedPluginNo      string
	lastRemovedPluginUserUID string
}

var _ MetaStore = (*stubMetaStore)(nil)

type stubMetaCommandProposer struct {
	lastData [][]byte
}

var _ MetaCommandProposer = (*stubMetaCommandProposer)(nil)

func (s *stubMetaCommandProposer) ProposeMetaCommandUntilApplied(data []byte) error {
	s.lastData = append(s.lastData, append([]byte(nil), data...))
	return nil
}

func (s *stubMetaStore) AddSystemUids(uids []string) error {
	s.lastAddedSystemUIDs = append([]string(nil), uids...)
	return nil
}

func (s *stubMetaStore) RemoveSystemUids(uids []string) error {
	s.lastRemovedSystemUIDs = append([]string(nil), uids...)
	return nil
}

func (s *stubMetaStore) GetSystemUids() ([]string, error) {
	return append([]string(nil), s.systemUIDs...), nil
}

func (s *stubMetaStore) AddOrUpdateTester(tester wkdb.Tester) error {
	s.lastPutTester = tester
	return nil
}

func (s *stubMetaStore) GetTester(no string) (wkdb.Tester, error) {
	return s.tester, nil
}

func (s *stubMetaStore) GetTesters() ([]wkdb.Tester, error) {
	return append([]wkdb.Tester(nil), s.testers...), nil
}

func (s *stubMetaStore) RemoveTester(no string) error {
	s.lastRemovedTesterNo = no
	return nil
}

func (s *stubMetaStore) AddOrUpdatePlugin(plugin wkdb.Plugin) error {
	s.lastPutPlugin = plugin
	return nil
}

func (s *stubMetaStore) DeletePlugin(no string) error {
	s.lastDeletedPluginNo = no
	return nil
}

func (s *stubMetaStore) GetPlugins() ([]wkdb.Plugin, error) {
	return append([]wkdb.Plugin(nil), s.plugins...), nil
}

func (s *stubMetaStore) GetPlugin(no string) (wkdb.Plugin, error) {
	return s.plugin, nil
}

func (s *stubMetaStore) AddOrUpdatePluginUsers(pluginUsers []wkdb.PluginUser) error {
	s.lastPluginUsersPut = append([]wkdb.PluginUser(nil), pluginUsers...)
	return nil
}

func (s *stubMetaStore) RemovePluginUser(pluginNo string, uid string) error {
	s.lastRemovedPluginNo = pluginNo
	s.lastRemovedPluginUserUID = uid
	return nil
}

func (s *stubMetaStore) GetPluginUsers(pluginNo string) ([]wkdb.PluginUser, error) {
	return append([]wkdb.PluginUser(nil), s.pluginUsers...), nil
}

func (s *stubMetaStore) GetHighestPriorityPluginByUid(uid string) (string, error) {
	return s.highestPriorityPluginNo, nil
}

func (s *stubMetaStore) ExistPluginByUid(uid string) (bool, error) {
	return s.existPluginByUID, nil
}

func (s *stubMetaStore) UpdatePluginConfig(no string, config map[string]interface{}) error {
	s.lastPluginConfigNo = no
	s.lastUpdatedPluginConfig = clonePluginConfig(config)
	return nil
}

func (s *stubMetaStore) GetPluginConfig(no string) (map[string]interface{}, error) {
	return clonePluginConfig(s.pluginConfig), nil
}

func (s *stubMetaStore) SearchPluginUsers(req wkdb.SearchPluginUserReq) ([]wkdb.PluginUser, error) {
	s.lastPluginUserSearchReq = req
	return append([]wkdb.PluginUser(nil), s.pluginUsers...), nil
}

func TestStoreMetaQueriesUseDedicatedMetaStore(t *testing.T) {
	meta := &stubMetaStore{
		systemUIDs:              []string{"system-1", "system-2"},
		tester:                  wkdb.Tester{No: "tester-1", Addr: "http://127.0.0.1:9466"},
		testers:                 []wkdb.Tester{{No: "tester-1", Addr: "http://127.0.0.1:9466"}},
		plugin:                  wkdb.Plugin{No: "plugin-a", Name: "Plugin A"},
		plugins:                 []wkdb.Plugin{{No: "plugin-a", Name: "Plugin A"}},
		pluginUsers:             []wkdb.PluginUser{{PluginNo: "plugin-a", Uid: "user-1"}},
		pluginConfig:            map[string]interface{}{"api_key": "secret"},
		highestPriorityPluginNo: "plugin-a",
		existPluginByUID:        true,
	}
	s := &Store{metaStore: meta}

	uids, err := s.GetSystemUids()
	require.NoError(t, err)
	require.Equal(t, []string{"system-1", "system-2"}, uids)

	tester, err := s.GetTester("tester-1")
	require.NoError(t, err)
	require.Equal(t, wkdb.Tester{No: "tester-1", Addr: "http://127.0.0.1:9466"}, tester)

	testers, err := s.GetTesters()
	require.NoError(t, err)
	require.Equal(t, []wkdb.Tester{{No: "tester-1", Addr: "http://127.0.0.1:9466"}}, testers)

	plugin, err := s.GetPlugin("plugin-a")
	require.NoError(t, err)
	require.Equal(t, wkdb.Plugin{No: "plugin-a", Name: "Plugin A"}, plugin)

	plugins, err := s.GetPlugins()
	require.NoError(t, err)
	require.Equal(t, []wkdb.Plugin{{No: "plugin-a", Name: "Plugin A"}}, plugins)

	pluginUsers, err := s.GetPluginUsers("plugin-a")
	require.NoError(t, err)
	require.Equal(t, []wkdb.PluginUser{{PluginNo: "plugin-a", Uid: "user-1"}}, pluginUsers)

	pluginUsers, err = s.SearchPluginUsers(wkdb.SearchPluginUserReq{Uid: "user-1"})
	require.NoError(t, err)
	require.Equal(t, wkdb.SearchPluginUserReq{Uid: "user-1"}, meta.lastPluginUserSearchReq)
	require.Equal(t, []wkdb.PluginUser{{PluginNo: "plugin-a", Uid: "user-1"}}, pluginUsers)

	exists, err := s.ExistPluginByUid("user-1")
	require.NoError(t, err)
	require.True(t, exists)

	pluginNo, err := s.GetHighestPriorityPluginByUid("user-1")
	require.NoError(t, err)
	require.Equal(t, "plugin-a", pluginNo)

	config, err := s.GetPluginConfig("plugin-a")
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{"api_key": "secret"}, config)

	require.NoError(t, s.AddOrUpdatePlugin(wkdb.Plugin{No: "plugin-b", Name: "Plugin B"}))
	require.Equal(t, wkdb.Plugin{No: "plugin-b", Name: "Plugin B"}, meta.lastPutPlugin)

	require.NoError(t, s.DeletePlugin("plugin-a"))
	require.Equal(t, "plugin-a", meta.lastDeletedPluginNo)

	require.NoError(t, s.UpdatePluginConfig("plugin-a", map[string]interface{}{"region": "ap-south"}))
	require.Equal(t, "plugin-a", meta.lastPluginConfigNo)
	require.Equal(t, map[string]interface{}{"region": "ap-south"}, meta.lastUpdatedPluginConfig)

	require.NoError(t, s.AddOrUpdatePluginUsers([]wkdb.PluginUser{{PluginNo: "plugin-a", Uid: "user-2"}}))
	require.Equal(t, []wkdb.PluginUser{{PluginNo: "plugin-a", Uid: "user-2"}}, meta.lastPluginUsersPut)
}

func TestStoreMetaApplyHandlersUseDedicatedMetaStore(t *testing.T) {
	meta := &stubMetaStore{}
	s := &Store{metaStore: meta}
	now := time.Unix(1710000000, 0)

	cmd := NewCMD(CMDSystemUIDsAdd, EncodeCMDSystemUIDs([]string{"system-1", "system-2"}))
	require.NoError(t, s.handleSystemUIDsAdd(cmd))
	require.Equal(t, []string{"system-1", "system-2"}, meta.lastAddedSystemUIDs)

	cmd = NewCMD(CMDSystemUIDsRemove, EncodeCMDSystemUIDs([]string{"system-2"}))
	require.NoError(t, s.handleSystemUIDsRemove(cmd))
	require.Equal(t, []string{"system-2"}, meta.lastRemovedSystemUIDs)

	cmd = NewCMD(CMDAddOrUpdateTester, EncodeCMDAddOrUpdateTester(wkdb.Tester{No: "tester-1", Addr: "addr", CreatedAt: &now, UpdatedAt: &now}))
	require.NoError(t, s.handleAddOrUpdateTester(cmd))
	require.Equal(t, "tester-1", meta.lastPutTester.No)
	require.Equal(t, "addr", meta.lastPutTester.Addr)
	require.NotNil(t, meta.lastPutTester.CreatedAt)
	require.NotNil(t, meta.lastPutTester.UpdatedAt)

	cmd = NewCMD(CMDRemoveTester, EncodeCMDRemoveTester("tester-1"))
	require.NoError(t, s.handleRemoveTester(cmd))
	require.Equal(t, "tester-1", meta.lastRemovedTesterNo)

	cmd = NewCMD(CMDUpdateUserPluginNo, EncodeCMDUserPluginNo(wkdb.PluginUser{PluginNo: "plugin-a", Uid: "user-1"}))
	require.NoError(t, s.handleUpdateUserPluginNo(cmd))
	require.Equal(t, []wkdb.PluginUser{{PluginNo: "plugin-a", Uid: "user-1"}}, meta.lastPluginUsersPut)

	cmd = NewCMD(CMDRemovePluginUser, EncodeCMDPluginUser("plugin-a", "user-1"))
	require.NoError(t, s.handleRemovePluginUser(cmd))
	require.Equal(t, "plugin-a", meta.lastRemovedPluginNo)
	require.Equal(t, "user-1", meta.lastRemovedPluginUserUID)
}

func TestStoreMetaMutationsUseDedicatedMetaCommandProposer(t *testing.T) {
	now := time.Unix(1710000000, 0)
	proposer := &stubMetaCommandProposer{}
	s := &Store{metaCommandProposer: proposer}

	require.NoError(t, s.AddSystemUids([]string{"system-1", "system-2"}))
	require.NoError(t, s.RemoveSystemUids([]string{"system-2"}))
	require.NoError(t, s.AddOrUpdateTester(wkdb.Tester{No: "tester-1", Addr: "addr", CreatedAt: &now, UpdatedAt: &now}))
	require.NoError(t, s.RemoveTester("tester-1"))
	require.NoError(t, s.UpdateUserPluginNo("user-1", "plugin-a"))
	require.NoError(t, s.RemovePluginUser("user-1", "plugin-a"))

	require.Len(t, proposer.lastData, 6)

	var cmd CMD
	require.NoError(t, cmd.Unmarshal(proposer.lastData[0]))
	require.Equal(t, CMDSystemUIDsAdd, cmd.CmdType)

	require.NoError(t, cmd.Unmarshal(proposer.lastData[1]))
	require.Equal(t, CMDSystemUIDsRemove, cmd.CmdType)

	require.NoError(t, cmd.Unmarshal(proposer.lastData[2]))
	require.Equal(t, CMDAddOrUpdateTester, cmd.CmdType)

	require.NoError(t, cmd.Unmarshal(proposer.lastData[3]))
	require.Equal(t, CMDRemoveTester, cmd.CmdType)

	require.NoError(t, cmd.Unmarshal(proposer.lastData[4]))
	require.Equal(t, CMDUpdateUserPluginNo, cmd.CmdType)

	require.NoError(t, cmd.Unmarshal(proposer.lastData[5]))
	require.Equal(t, CMDRemovePluginUser, cmd.CmdType)
}
