package store

import (
	"fmt"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
)

func (s *Store) requireAdminSearchStore() (AdminSearchStore, error) {
	if s.adminSearchStore == nil {
		return nil, fmt.Errorf("admin search store is not configured")
	}
	return s.adminSearchStore, nil
}

func (s *Store) SearchUsers(req wkdb.UserSearchReq) ([]wkdb.User, error) {
	adminSearchStore, err := s.requireAdminSearchStore()
	if err != nil {
		return nil, err
	}
	return adminSearchStore.SearchUsers(req)
}

func (s *Store) CountUsers() (int, error) {
	adminSearchStore, err := s.requireAdminSearchStore()
	if err != nil {
		return 0, err
	}
	return adminSearchStore.CountUsers()
}

func (s *Store) SearchDevices(req wkdb.DeviceSearchReq) ([]wkdb.Device, error) {
	adminSearchStore, err := s.requireAdminSearchStore()
	if err != nil {
		return nil, err
	}
	return adminSearchStore.SearchDevices(req)
}

func (s *Store) CountDevices() (int, error) {
	adminSearchStore, err := s.requireAdminSearchStore()
	if err != nil {
		return 0, err
	}
	return adminSearchStore.CountDevices()
}

func (s *Store) SearchChannels(req wkdb.ChannelSearchReq) ([]wkdb.ChannelInfo, error) {
	adminSearchStore, err := s.requireAdminSearchStore()
	if err != nil {
		return nil, err
	}
	return adminSearchStore.SearchChannels(req)
}

func (s *Store) SearchConversations(req wkdb.ConversationSearchReq) ([]wkdb.Conversation, error) {
	adminSearchStore, err := s.requireAdminSearchStore()
	if err != nil {
		return nil, err
	}
	return adminSearchStore.SearchConversations(req)
}

func (s *Store) CountConversations() (int, error) {
	adminSearchStore, err := s.requireAdminSearchStore()
	if err != nil {
		return 0, err
	}
	return adminSearchStore.CountConversations()
}

func (s *Store) SearchChannelClusterConfigs(req wkdb.ChannelClusterConfigSearchReq, filter ...func(cfg wkdb.ChannelClusterConfig) bool) ([]wkdb.ChannelClusterConfig, error) {
	adminSearchStore, err := s.requireAdminSearchStore()
	if err != nil {
		return nil, err
	}
	return adminSearchStore.SearchChannelClusterConfigs(req, filter...)
}
