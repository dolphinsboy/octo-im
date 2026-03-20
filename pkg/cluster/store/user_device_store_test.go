package store

import (
	"context"
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/raft/types"
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

type captureDeviceProposalSlot struct {
	slotByUID               map[string]uint32
	getSlotIdCalls          []string
	proposedUntilAppliedIDs []uint32
	proposedCmds            [][]byte
}

func (s *captureDeviceProposalSlot) SlotLeaderId(slotId uint32) uint64 {
	return 0
}

func (s *captureDeviceProposalSlot) GetSlotId(v string) uint32 {
	s.getSlotIdCalls = append(s.getSlotIdCalls, v)
	return s.slotByUID[v]
}

func (s *captureDeviceProposalSlot) Propose(slotId uint32, data []byte) (*types.ProposeResp, error) {
	return &types.ProposeResp{}, nil
}

func (s *captureDeviceProposalSlot) ProposeUntilApplied(slotId uint32, data []byte) (*types.ProposeResp, error) {
	s.proposedUntilAppliedIDs = append(s.proposedUntilAppliedIDs, slotId)
	s.proposedCmds = append(s.proposedCmds, append([]byte(nil), data...))
	return &types.ProposeResp{}, nil
}

func (s *captureDeviceProposalSlot) ProposeUntilAppliedTimeout(ctx context.Context, slotId uint32, data []byte) (*types.ProposeResp, error) {
	return &types.ProposeResp{}, nil
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

	s := New(NewOptions(WithCompatDBRuntime(hybrid)))
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

func TestStoreAddDeviceAssignsPrimaryKeyWhenMissing(t *testing.T) {
	slot := &captureDeviceProposalSlot{
		slotByUID: map[string]uint32{"user-1": 11},
	}
	s := New(NewOptions(
		WithSlot(slot),
		WithPrimaryKeyAllocator(&stubPrimaryKeyAllocator{nextPrimaryKey: 99}),
	))

	now := time.Unix(1710000000, 0)
	require.NoError(t, s.AddDevice(wkdb.Device{
		Uid:         "user-1",
		DeviceFlag:  1,
		DeviceLevel: 2,
		Token:       "token-1",
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}))

	require.Equal(t, []string{"user-1"}, slot.getSlotIdCalls)
	require.Equal(t, []uint32{11}, slot.proposedUntilAppliedIDs)
	require.Len(t, slot.proposedCmds, 1)

	var cmd CMD
	require.NoError(t, cmd.Unmarshal(slot.proposedCmds[0]))
	device, err := cmd.DecodeCMDDevice()
	require.NoError(t, err)
	require.Equal(t, uint64(99), device.Id)
	require.Equal(t, "user-1", device.Uid)
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
