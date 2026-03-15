package store

import (
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/WuKongIM/WuKongIM/pkg/wkutil"
)

func (s *Store) GetPlugin(no string) (wkdb.Plugin, error) {
	return s.metaStore.GetPlugin(no)
}

func (s *Store) GetPlugins() ([]wkdb.Plugin, error) {
	return s.metaStore.GetPlugins()
}

func (s *Store) AddOrUpdatePlugin(plugin wkdb.Plugin) error {
	return s.metaStore.AddOrUpdatePlugin(plugin)
}

func (s *Store) DeletePlugin(no string) error {
	return s.metaStore.DeletePlugin(no)
}

func (s *Store) UpdatePluginConfig(pluginNo string, config map[string]interface{}) error {
	return s.metaStore.UpdatePluginConfig(pluginNo, config)
}

func (s *Store) GetPluginConfig(no string) (map[string]interface{}, error) {
	return s.metaStore.GetPluginConfig(no)
}

func (s *Store) SearchPluginUsers(req wkdb.SearchPluginUserReq) ([]wkdb.PluginUser, error) {
	return s.metaStore.SearchPluginUsers(req)
}

func (s *Store) GetPluginUsers(pluginNo string) ([]wkdb.PluginUser, error) {
	return s.metaStore.GetPluginUsers(pluginNo)
}

func (s *Store) ExistPluginByUid(uid string) (bool, error) {
	return s.metaStore.ExistPluginByUid(uid)
}

func (s *Store) GetHighestPriorityPluginByUid(uid string) (string, error) {
	return s.metaStore.GetHighestPriorityPluginByUid(uid)
}

func (s *Store) AddOrUpdatePluginUsers(pluginUsers []wkdb.PluginUser) error {
	return s.metaStore.AddOrUpdatePluginUsers(pluginUsers)
}

// UpdateUserPluginNo 更新用户插件编号
func (s *Store) UpdateUserPluginNo(uid string, pluginNo string) error {
	data := EncodeCMDUserPluginNo(wkdb.PluginUser{
		Uid:       uid,
		PluginNo:  pluginNo,
		CreatedAt: wkutil.TimePtr(time.Now()),
		UpdatedAt: wkutil.TimePtr(time.Now()),
	})
	cmd := NewCMD(CMDUpdateUserPluginNo, data)
	cmdData, err := cmd.Marshal()
	if err != nil {
		return err
	}
	return s.metaCommandProposer.ProposeMetaCommandUntilApplied(cmdData)
}

func (s *Store) RemovePluginUser(uid string, pluginNo string) error {
	data := EncodeCMDPluginUser(pluginNo, uid)
	cmd := NewCMD(CMDRemovePluginUser, data)
	cmdData, err := cmd.Marshal()
	if err != nil {
		return err
	}
	return s.metaCommandProposer.ProposeMetaCommandUntilApplied(cmdData)
}
