package store

import "github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"

func (s *Store) SearchUsers(req wkdb.UserSearchReq) ([]wkdb.User, error) {
	if s.adminSearchStore != nil {
		return s.adminSearchStore.SearchUsers(req)
	}
	return s.wdb.SearchUser(req)
}

func (s *Store) CountUsers() (int, error) {
	if s.adminSearchStore != nil {
		return s.adminSearchStore.CountUsers()
	}
	return s.wdb.GetTotalUserCount()
}

func (s *Store) SearchDevices(req wkdb.DeviceSearchReq) ([]wkdb.Device, error) {
	if s.adminSearchStore != nil {
		return s.adminSearchStore.SearchDevices(req)
	}
	return s.wdb.SearchDevice(req)
}

func (s *Store) CountDevices() (int, error) {
	if s.adminSearchStore != nil {
		return s.adminSearchStore.CountDevices()
	}
	return s.wdb.GetTotalDeviceCount()
}

func (s *Store) SearchChannels(req wkdb.ChannelSearchReq) ([]wkdb.ChannelInfo, error) {
	if s.adminSearchStore != nil {
		return s.adminSearchStore.SearchChannels(req)
	}
	return s.wdb.SearchChannels(req)
}

func (s *Store) SearchConversations(req wkdb.ConversationSearchReq) ([]wkdb.Conversation, error) {
	if s.adminSearchStore != nil {
		return s.adminSearchStore.SearchConversations(req)
	}
	return s.wdb.SearchConversation(req)
}

func (s *Store) CountConversations() (int, error) {
	if s.adminSearchStore != nil {
		return s.adminSearchStore.CountConversations()
	}
	return s.wdb.GetTotalSessionCount()
}

func (s *Store) SearchChannelClusterConfigs(req wkdb.ChannelClusterConfigSearchReq, filter ...func(cfg wkdb.ChannelClusterConfig) bool) ([]wkdb.ChannelClusterConfig, error) {
	if s.adminSearchStore != nil {
		return s.adminSearchStore.SearchChannelClusterConfigs(req, filter...)
	}
	return s.wdb.SearchChannelClusterConfig(req, filter...)
}
