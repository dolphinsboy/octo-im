package store

import (
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/stretchr/testify/require"
)

type stubAdminSearchStore struct {
	searchConversationsFn func(req wkdb.ConversationSearchReq) ([]wkdb.Conversation, error)
}

var _ AdminSearchStore = (*stubAdminSearchStore)(nil)

func (s *stubAdminSearchStore) SearchUsers(req wkdb.UserSearchReq) ([]wkdb.User, error) {
	return nil, nil
}

func (s *stubAdminSearchStore) CountUsers() (int, error) {
	return 0, nil
}

func (s *stubAdminSearchStore) SearchDevices(req wkdb.DeviceSearchReq) ([]wkdb.Device, error) {
	return nil, nil
}

func (s *stubAdminSearchStore) CountDevices() (int, error) {
	return 0, nil
}

func (s *stubAdminSearchStore) SearchChannels(req wkdb.ChannelSearchReq) ([]wkdb.ChannelInfo, error) {
	return nil, nil
}

func (s *stubAdminSearchStore) SearchConversations(req wkdb.ConversationSearchReq) ([]wkdb.Conversation, error) {
	if s.searchConversationsFn != nil {
		return s.searchConversationsFn(req)
	}
	return nil, nil
}

func (s *stubAdminSearchStore) CountConversations() (int, error) {
	return 0, nil
}

func (s *stubAdminSearchStore) SearchChannelClusterConfigs(req wkdb.ChannelClusterConfigSearchReq, filter ...func(cfg wkdb.ChannelClusterConfig) bool) ([]wkdb.ChannelClusterConfig, error) {
	return nil, nil
}

func TestStoreDerivesAdminSearchStoreFromHybridDB(t *testing.T) {
	hybrid, _ := newHybridMetaLocalTestDB(t)
	s := New(NewOptions(WithCompatDBRuntime(hybrid)))
	require.NotNil(t, s.adminSearchStore)
}

func TestStoreDerivesLegacyAdminSearchStoreFromLegacyDB(t *testing.T) {
	_, legacy := newHybridMetaLocalTestDB(t)
	now := time.Unix(1710000000, 0)

	require.NoError(t, legacy.AddUser(wkdb.User{
		Id:        1,
		Uid:       "user-in-legacy",
		CreatedAt: &now,
		UpdatedAt: &now,
	}))

	s := New(NewOptions(WithCompatDBRuntime(legacy)))
	require.IsType(t, &LegacyAdminSearchStore{}, s.adminSearchStore)

	users, err := s.SearchUsers(wkdb.UserSearchReq{Limit: 10})
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "user-in-legacy", users[0].Uid)
}

func TestStoreSearchUsersUsesDerivedAdminSearchStore(t *testing.T) {
	hybrid, legacy := newHybridMetaLocalTestDB(t)
	now := time.Unix(1710000000, 0)

	require.NoError(t, hybrid.slotDB.Slots().Scope(0).Users().Put(wkdb.User{
		Id:        1,
		Uid:       "user-in-v3",
		CreatedAt: &now,
		UpdatedAt: &now,
	}))

	legacyUsers, err := legacy.SearchUser(wkdb.UserSearchReq{Limit: 0})
	require.NoError(t, err)
	require.Empty(t, legacyUsers)

	s := New(NewOptions(WithCompatDBRuntime(hybrid)))

	users, err := s.SearchUsers(wkdb.UserSearchReq{Limit: 10})
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "user-in-v3", users[0].Uid)

	count, err := s.CountUsers()
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestStoreGetChannelConversationLocalUsersUsesAdminSearchStore(t *testing.T) {
	s := &Store{
		adminSearchStore: &stubAdminSearchStore{
			searchConversationsFn: func(req wkdb.ConversationSearchReq) ([]wkdb.Conversation, error) {
				require.Equal(t, 0, req.Limit)
				return []wkdb.Conversation{
					{Uid: "user-2", ChannelId: "channel-1", ChannelType: 2},
					{Uid: "user-1", ChannelId: "channel-1", ChannelType: 2},
					{Uid: "user-1", ChannelId: "channel-1", ChannelType: 2},
					{Uid: "user-3", ChannelId: "channel-2", ChannelType: 2},
				}, nil
			},
		},
	}

	uids, err := s.GetChannelConversationLocalUsers("channel-1", 2)
	require.NoError(t, err)
	require.Equal(t, []string{"user-1", "user-2"}, uids)
}
