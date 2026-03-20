package store

import "github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"

type LegacyAdminSearchStore struct {
	db wkdb.DB
}

var _ AdminSearchStore = (*LegacyAdminSearchStore)(nil)

func NewLegacyAdminSearchStore(db wkdb.DB) *LegacyAdminSearchStore {
	if db == nil {
		return nil
	}
	return &LegacyAdminSearchStore{db: db}
}

func (s *LegacyAdminSearchStore) SearchUsers(req wkdb.UserSearchReq) ([]wkdb.User, error) {
	return s.db.SearchUser(req)
}

func (s *LegacyAdminSearchStore) CountUsers() (int, error) {
	return s.db.GetTotalUserCount()
}

func (s *LegacyAdminSearchStore) SearchDevices(req wkdb.DeviceSearchReq) ([]wkdb.Device, error) {
	return s.db.SearchDevice(req)
}

func (s *LegacyAdminSearchStore) CountDevices() (int, error) {
	return s.db.GetTotalDeviceCount()
}

func (s *LegacyAdminSearchStore) SearchChannels(req wkdb.ChannelSearchReq) ([]wkdb.ChannelInfo, error) {
	return s.db.SearchChannels(req)
}

func (s *LegacyAdminSearchStore) SearchConversations(req wkdb.ConversationSearchReq) ([]wkdb.Conversation, error) {
	return s.db.SearchConversation(req)
}

func (s *LegacyAdminSearchStore) CountConversations() (int, error) {
	return s.db.GetTotalSessionCount()
}

func (s *LegacyAdminSearchStore) SearchChannelClusterConfigs(req wkdb.ChannelClusterConfigSearchReq, filter ...func(cfg wkdb.ChannelClusterConfig) bool) ([]wkdb.ChannelClusterConfig, error) {
	return s.db.SearchChannelClusterConfig(req, filter...)
}
