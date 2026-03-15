package service

import (
	"errors"
	"testing"

	clusterstore "github.com/WuKongIM/WuKongIM/pkg/cluster/store"
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkproto "github.com/WuKongIM/WuKongIMGoProto"
	"github.com/stretchr/testify/require"
)

type stubChannelStateStore struct {
	channelInfo wkdb.ChannelInfo
	getErr      error
}

var _ clusterstore.ChannelStateStore = (*stubChannelStateStore)(nil)

func (s *stubChannelStateStore) AddSubscribers(channelId string, channelType uint8, members []wkdb.Member) error {
	return nil
}

func (s *stubChannelStateStore) RemoveSubscribers(channelId string, channelType uint8, uids []string) error {
	return nil
}

func (s *stubChannelStateStore) ExistSubscriber(channelId string, channelType uint8, uid string) (bool, error) {
	return false, nil
}

func (s *stubChannelStateStore) RemoveAllSubscriber(channelId string, channelType uint8) error {
	return nil
}

func (s *stubChannelStateStore) GetSubscribers(channelId string, channelType uint8) ([]wkdb.Member, error) {
	return nil, nil
}

func (s *stubChannelStateStore) AddChannel(channelInfo wkdb.ChannelInfo) (uint64, error) {
	return channelInfo.Id, nil
}

func (s *stubChannelStateStore) GetChannel(channelId string, channelType uint8) (wkdb.ChannelInfo, error) {
	return s.channelInfo, s.getErr
}

func (s *stubChannelStateStore) UpdateChannel(channelInfo wkdb.ChannelInfo) error {
	return nil
}

func (s *stubChannelStateStore) ExistChannel(channelId string, channelType uint8) (bool, error) {
	return false, nil
}

func (s *stubChannelStateStore) DeleteChannel(channelId string, channelType uint8) error {
	return nil
}

func (s *stubChannelStateStore) AddDenylist(channelId string, channelType uint8, members []wkdb.Member) error {
	return nil
}

func (s *stubChannelStateStore) GetDenylist(channelId string, channelType uint8) ([]wkdb.Member, error) {
	return nil, nil
}

func (s *stubChannelStateStore) RemoveDenylist(channelId string, channelType uint8, uids []string) error {
	return nil
}

func (s *stubChannelStateStore) RemoveAllDenylist(channelId string, channelType uint8) error {
	return nil
}

func (s *stubChannelStateStore) ExistDenylist(channelId string, channelType uint8, uid string) (bool, error) {
	return false, nil
}

func (s *stubChannelStateStore) AddAllowlist(channelId string, channelType uint8, members []wkdb.Member) error {
	return nil
}

func (s *stubChannelStateStore) GetAllowlist(channelId string, channelType uint8) ([]wkdb.Member, error) {
	return nil, nil
}

func (s *stubChannelStateStore) RemoveAllowlist(channelId string, channelType uint8, uids []string) error {
	return nil
}

func (s *stubChannelStateStore) RemoveAllAllowlist(channelId string, channelType uint8) error {
	return nil
}

func (s *stubChannelStateStore) ExistAllowlist(channelId string, channelType uint8, uid string) (bool, error) {
	return false, nil
}

func (s *stubChannelStateStore) HasAllowlist(channelId string, channelType uint8) (bool, error) {
	return false, nil
}

func TestLoadChannelInfoOrEmptyReturnsEmptyOnNotFound(t *testing.T) {
	previousStore := Store
	Store = clusterstore.New(clusterstore.NewOptions(
		clusterstore.WithChannelStateStore(&stubChannelStateStore{getErr: wkdb.ErrNotFound}),
	))
	t.Cleanup(func() {
		Store = previousStore
	})

	channelInfo, err := LoadChannelInfoOrEmpty("user-1", wkproto.ChannelTypePerson)
	require.NoError(t, err)
	require.True(t, wkdb.IsEmptyChannelInfo(channelInfo))
}

func TestPermissionServiceHasPermissionForChannelAllowsMissingChannel(t *testing.T) {
	previousStore := Store
	Store = clusterstore.New(clusterstore.NewOptions(
		clusterstore.WithChannelStateStore(&stubChannelStateStore{getErr: wkdb.ErrNotFound}),
	))
	t.Cleanup(func() {
		Store = previousStore
	})

	permission := NewPermissionService(nil)
	reasonCode, err := permission.HasPermissionForChannel("user-1", wkproto.ChannelTypePerson)
	require.NoError(t, err)
	require.Equal(t, wkproto.ReasonSuccess, reasonCode)
}

func TestPermissionServiceHasPermissionForChannelRejectsBanAndDisband(t *testing.T) {
	tests := []struct {
		name       string
		channel    wkdb.ChannelInfo
		reasonCode wkproto.ReasonCode
	}{
		{
			name:       "ban",
			channel:    wkdb.ChannelInfo{ChannelId: "user-1", ChannelType: wkproto.ChannelTypePerson, Ban: true},
			reasonCode: wkproto.ReasonBan,
		},
		{
			name:       "disband",
			channel:    wkdb.ChannelInfo{ChannelId: "group-1", ChannelType: wkproto.ChannelTypeGroup, Disband: true},
			reasonCode: wkproto.ReasonDisband,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previousStore := Store
			Store = clusterstore.New(clusterstore.NewOptions(
				clusterstore.WithChannelStateStore(&stubChannelStateStore{channelInfo: tt.channel}),
			))
			t.Cleanup(func() {
				Store = previousStore
			})

			permission := NewPermissionService(nil)
			reasonCode, err := permission.HasPermissionForChannel(tt.channel.ChannelId, tt.channel.ChannelType)
			require.NoError(t, err)
			require.Equal(t, tt.reasonCode, reasonCode)
		})
	}
}

func TestPermissionServiceHasPermissionForChannelReturnsStoreErrors(t *testing.T) {
	previousStore := Store
	Store = clusterstore.New(clusterstore.NewOptions(
		clusterstore.WithChannelStateStore(&stubChannelStateStore{getErr: errors.New("boom")}),
	))
	t.Cleanup(func() {
		Store = previousStore
	})

	permission := NewPermissionService(nil)
	reasonCode, err := permission.HasPermissionForChannel("user-1", wkproto.ChannelTypePerson)
	require.EqualError(t, err, "boom")
	require.Equal(t, wkproto.ReasonSystemError, reasonCode)
}
