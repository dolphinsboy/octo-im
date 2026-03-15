package store

import (
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
)

type UserDeviceStore interface {
	AddUser(u wkdb.User) error
	UpdateUser(u wkdb.User) error
	GetUser(uid string) (wkdb.User, error)
	AddDevice(d wkdb.Device) error
	UpdateDevice(d wkdb.Device) error
	GetDevice(uid string, deviceFlag uint64) (wkdb.Device, error)
}

type SlotUserDeviceStore struct {
	db        wkdbv3.DB
	routeSlot func(key string) uint32
}

var _ UserDeviceStore = (*SlotUserDeviceStore)(nil)

func NewSlotUserDeviceStore(db wkdbv3.DB, routeSlot func(key string) uint32) *SlotUserDeviceStore {
	if db == nil || routeSlot == nil {
		return nil
	}
	return &SlotUserDeviceStore{
		db:        db,
		routeSlot: routeSlot,
	}
}

func (s *SlotUserDeviceStore) slotScopeByUID(uid string) wkdbv3.SlotScope {
	return s.db.Slots().Scope(s.routeSlot(uid))
}

func (s *SlotUserDeviceStore) AddUser(u wkdb.User) error {
	return s.slotScopeByUID(u.Uid).Users().Put(u)
}

func (s *SlotUserDeviceStore) UpdateUser(u wkdb.User) error {
	return s.slotScopeByUID(u.Uid).Users().Put(u)
}

func (s *SlotUserDeviceStore) GetUser(uid string) (wkdb.User, error) {
	return s.slotScopeByUID(uid).Users().Get(uid)
}

func (s *SlotUserDeviceStore) AddDevice(d wkdb.Device) error {
	return s.slotScopeByUID(d.Uid).Devices().Put(d)
}

func (s *SlotUserDeviceStore) UpdateDevice(d wkdb.Device) error {
	return s.slotScopeByUID(d.Uid).Devices().Put(d)
}

func (s *SlotUserDeviceStore) GetDevice(uid string, deviceFlag uint64) (wkdb.Device, error) {
	return s.slotScopeByUID(uid).Devices().Get(uid, deviceFlag)
}
