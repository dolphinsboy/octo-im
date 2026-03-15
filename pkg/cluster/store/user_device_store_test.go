package store

import (
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkproto "github.com/WuKongIM/WuKongIMGoProto"
	"github.com/stretchr/testify/require"
)

type stubUserDeviceStore struct {
	user           wkdb.User
	device         wkdb.Device
	lastUID        string
	lastDeviceFlag uint64
	lastOp         string
	addedUser      wkdb.User
	updatedUser    wkdb.User
	addedDevice    wkdb.Device
	updatedDevice  wkdb.Device
}

var _ UserDeviceStore = (*stubUserDeviceStore)(nil)

func (s *stubUserDeviceStore) AddUser(u wkdb.User) error {
	s.lastOp = "add_user"
	s.addedUser = u
	s.user = u
	return nil
}

func (s *stubUserDeviceStore) UpdateUser(u wkdb.User) error {
	s.lastOp = "update_user"
	s.updatedUser = u
	s.user = u
	return nil
}

func (s *stubUserDeviceStore) GetUser(uid string) (wkdb.User, error) {
	s.lastOp = "get_user"
	s.lastUID = uid
	return s.user, nil
}

func (s *stubUserDeviceStore) AddDevice(d wkdb.Device) error {
	s.lastOp = "add_device"
	s.addedDevice = d
	s.device = d
	return nil
}

func (s *stubUserDeviceStore) UpdateDevice(d wkdb.Device) error {
	s.lastOp = "update_device"
	s.updatedDevice = d
	s.device = d
	return nil
}

func (s *stubUserDeviceStore) GetDevice(uid string, deviceFlag uint64) (wkdb.Device, error) {
	s.lastOp = "get_device"
	s.lastUID = uid
	s.lastDeviceFlag = deviceFlag
	return s.device, nil
}

func TestStoreDerivesUserDeviceStoreFromHybridDB(t *testing.T) {
	hybrid, _ := newHybridMetaLocalTestDB(t)
	now := time.Unix(1710000000, 0)

	require.NoError(t, hybrid.slotDB.Slots().Scope(0).Users().Put(wkdb.User{
		Id:        1,
		Uid:       "user-in-v3",
		CreatedAt: &now,
		UpdatedAt: &now,
	}))
	require.NoError(t, hybrid.slotDB.Slots().Scope(0).Devices().Put(wkdb.Device{
		Id:         2,
		Uid:        "user-in-v3",
		DeviceFlag: 1,
		Token:      "token-in-v3",
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}))

	s := New(NewOptions(WithDB(hybrid)))
	require.NotNil(t, s.userDeviceStore)

	user, err := s.GetUser("user-in-v3")
	require.NoError(t, err)
	require.Equal(t, "user-in-v3", user.Uid)

	device, err := s.GetDevice("user-in-v3", wkproto.DeviceFlag(1))
	require.NoError(t, err)
	require.Equal(t, "token-in-v3", device.Token)
}

func TestStoreUserAndDeviceReadsUseUserDeviceStore(t *testing.T) {
	now := time.Unix(1710000000, 0)
	stub := &stubUserDeviceStore{
		user: wkdb.User{
			Id:        1,
			Uid:       "user-1",
			CreatedAt: &now,
			UpdatedAt: &now,
		},
		device: wkdb.Device{
			Id:         2,
			Uid:        "user-1",
			DeviceFlag: 1,
			Token:      "token-1",
			CreatedAt:  &now,
			UpdatedAt:  &now,
		},
	}
	s := &Store{userDeviceStore: stub}

	user, err := s.GetUser("user-1")
	require.NoError(t, err)
	require.Equal(t, "user-1", user.Uid)
	require.Equal(t, "get_user", stub.lastOp)
	require.Equal(t, "user-1", stub.lastUID)

	device, err := s.GetDevice("user-1", wkproto.DeviceFlag(1))
	require.NoError(t, err)
	require.Equal(t, "token-1", device.Token)
	require.Equal(t, "get_device", stub.lastOp)
	require.Equal(t, "user-1", stub.lastUID)
	require.Equal(t, uint64(1), stub.lastDeviceFlag)
}

func TestStoreApplyUserAndDeviceCommandsUseUserDeviceStore(t *testing.T) {
	now := time.Unix(1710000000, 0)
	user := wkdb.User{
		Id:        1,
		Uid:       "user-1",
		CreatedAt: &now,
		UpdatedAt: &now,
	}
	device := wkdb.Device{
		Id:         2,
		Uid:        "user-1",
		DeviceFlag: 1,
		Token:      "token-1",
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}

	stub := &stubUserDeviceStore{}
	s := &Store{userDeviceStore: stub}

	require.NoError(t, s.handleAddUser(NewCMD(CMDAddUser, EncodeCMDUser(user))))
	require.Equal(t, "add_user", stub.lastOp)
	require.Equal(t, user.Uid, stub.addedUser.Uid)

	updatedAt := now.Add(time.Minute)
	user.UpdatedAt = &updatedAt
	require.NoError(t, s.handleUpdateUser(NewCMD(CMDUpdateUser, EncodeCMDUser(user))))
	require.Equal(t, "update_user", stub.lastOp)
	require.NotNil(t, stub.updatedUser.UpdatedAt)
	require.True(t, stub.updatedUser.UpdatedAt.Equal(updatedAt))

	require.NoError(t, s.handleAddDevice(NewCMD(CMDAddDevice, EncodeCMDDevice(device))))
	require.Equal(t, "add_device", stub.lastOp)
	require.Equal(t, "token-1", stub.addedDevice.Token)

	device.Token = "token-2"
	require.NoError(t, s.handleUpdateDevice(NewCMD(CMDUpdateDevice, EncodeCMDDevice(device))))
	require.Equal(t, "update_device", stub.lastOp)
	require.Equal(t, "token-2", stub.updatedDevice.Token)
}
