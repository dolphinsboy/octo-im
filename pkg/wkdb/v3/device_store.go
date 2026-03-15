package v3

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	wkproto "github.com/WuKongIM/WuKongIMGoProto"
	"github.com/cockroachdb/pebble"
)

type pebbleDeviceStore struct {
	slotID  uint32
	buckets *PebbleBucketManager
	now     func() time.Time
}

var _ DeviceStore = (*pebbleDeviceStore)(nil)

func (p *pebbleDeviceStore) Get(uid string, deviceFlag uint64) (wkdb.Device, error) {
	bucket, err := p.bucket()
	if err != nil {
		return wkdb.Device{}, err
	}
	value, closer, err := bucket.DB().Get(v3key.EncodeDeviceRowKey(p.slotID, uid, deviceFlag))
	if err != nil {
		if err == pebble.ErrNotFound {
			return wkdb.Device{}, wkdb.ErrNotFound
		}
		return wkdb.Device{}, err
	}
	defer closer.Close()
	return decodeDeviceValue(value)
}

func (p *pebbleDeviceStore) ListByUID(uid string) ([]wkdb.Device, error) {
	bucket, err := p.bucket()
	if err != nil {
		return nil, err
	}
	lower, upper := v3key.DeviceUIDRange(p.slotID, uid)
	iter := bucket.DB().NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	devices := make([]wkdb.Device, 0)
	for iter.First(); iter.Valid(); iter.Next() {
		device, err := decodeDeviceValue(iter.Value())
		if err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	sortDevices(devices)
	return devices, nil
}

func (p *pebbleDeviceStore) Put(device wkdb.Device) error {
	if device.Uid == "" {
		return wkdb.ErrInvalidUserId
	}
	if device.DeviceFlag == 0 {
		return wkdb.ErrInvalidDeviceId
	}
	bucket, err := p.bucket()
	if err != nil {
		return err
	}
	oldDevice, err := p.Get(device.Uid, device.DeviceFlag)
	if err != nil && err != wkdb.ErrNotFound {
		return err
	}
	existed := err == nil
	now := p.currentTime()
	if existed && oldDevice.CreatedAt != nil {
		device.CreatedAt = cloneTimePtr(oldDevice.CreatedAt)
	} else if device.CreatedAt == nil {
		device.CreatedAt = cloneTimePtr(&now)
	}
	if device.UpdatedAt == nil {
		device.UpdatedAt = cloneTimePtr(&now)
	}

	batch := bucket.DB().NewBatch()
	defer batch.Close()
	if existed && oldDevice.CreatedAt != nil {
		if err := batch.Delete(v3key.EncodeDeviceCreatedAtSecondIndexKey(p.slotID, uint64(oldDevice.CreatedAt.UnixNano()), oldDevice.Uid, oldDevice.DeviceFlag), p.writeOptions()); err != nil {
			return err
		}
	}
	value, err := encodeDeviceValue(device)
	if err != nil {
		return err
	}
	if err := batch.Set(v3key.EncodeDeviceRowKey(p.slotID, device.Uid, device.DeviceFlag), value, p.writeOptions()); err != nil {
		return err
	}
	if err := batch.Set(v3key.EncodeDeviceCreatedAtSecondIndexKey(p.slotID, uint64(device.CreatedAt.UnixNano()), device.Uid, device.DeviceFlag), []byte(device.Uid), p.writeOptions()); err != nil {
		return err
	}
	return batch.Commit(p.writeOptions())
}

func (p *pebbleDeviceStore) Delete(uid string, deviceFlag uint64) error {
	device, err := p.Get(uid, deviceFlag)
	if err != nil {
		if err == wkdb.ErrNotFound {
			return nil
		}
		return err
	}
	bucket, err := p.bucket()
	if err != nil {
		return err
	}
	batch := bucket.DB().NewBatch()
	defer batch.Close()
	if err := batch.Delete(v3key.EncodeDeviceRowKey(p.slotID, uid, deviceFlag), p.writeOptions()); err != nil {
		return err
	}
	if device.CreatedAt != nil {
		if err := batch.Delete(v3key.EncodeDeviceCreatedAtSecondIndexKey(p.slotID, uint64(device.CreatedAt.UnixNano()), uid, deviceFlag), p.writeOptions()); err != nil {
			return err
		}
	}
	return batch.Commit(p.writeOptions())
}

func (p *pebbleDeviceStore) Search(req wkdb.DeviceSearchReq) ([]wkdb.Device, error) {
	if req.Uid != "" && req.DeviceFlag != 0 {
		device, err := p.Get(req.Uid, req.DeviceFlag)
		if err != nil {
			return nil, err
		}
		if !matchDeviceSearch(device, req) {
			return nil, nil
		}
		return limitDevices([]wkdb.Device{device}, req.Limit, req.Pre), nil
	}
	if req.Uid != "" {
		devices, err := p.ListByUID(req.Uid)
		if err != nil {
			return nil, err
		}
		filtered := make([]wkdb.Device, 0, len(devices))
		for _, device := range devices {
			if req.DeviceFlag != 0 && device.DeviceFlag != req.DeviceFlag {
				continue
			}
			if !matchDeviceSearch(device, req) {
				continue
			}
			filtered = append(filtered, device)
		}
		sortDevices(filtered)
		return limitDevices(filtered, req.Limit, req.Pre), nil
	}

	bucket, err := p.bucket()
	if err != nil {
		return nil, err
	}
	lower, upper := v3key.SlotTableRange(p.slotID, v3key.ScopeSlotPrimary, v3key.TableDevice)
	iter := bucket.DB().NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	devices := make([]wkdb.Device, 0)
	for iter.First(); iter.Valid(); iter.Next() {
		device, err := decodeDeviceValue(iter.Value())
		if err != nil {
			return nil, err
		}
		if req.DeviceFlag != 0 && device.DeviceFlag != req.DeviceFlag {
			continue
		}
		if !matchDeviceSearch(device, req) {
			continue
		}
		devices = append(devices, device)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	sortDevices(devices)
	return limitDevices(devices, req.Limit, req.Pre), nil
}

func (p *pebbleDeviceStore) bucket() (*PebbleBucket, error) {
	return p.buckets.Bucket(p.slotID)
}

func (p *pebbleDeviceStore) currentTime() time.Time {
	if p.now != nil {
		return p.now()
	}
	return time.Now()
}

func (p *pebbleDeviceStore) writeOptions() *pebble.WriteOptions {
	bucket, _ := p.bucket()
	if bucket == nil {
		return pebble.Sync
	}
	return bucket.SnapshotStore().writeOptions
}

func encodeDeviceValue(device wkdb.Device) ([]byte, error) {
	if device.Uid == "" {
		return nil, wkdb.ErrInvalidUserId
	}
	if device.DeviceFlag == 0 {
		return nil, wkdb.ErrInvalidDeviceId
	}
	if device.CreatedAt == nil || device.UpdatedAt == nil {
		return nil, fmt.Errorf("device timestamps are required")
	}
	enc := wkproto.NewEncoder()
	defer enc.End()
	enc.WriteUint64(device.Id)
	enc.WriteString(device.Uid)
	enc.WriteString(device.Token)
	enc.WriteUint64(device.DeviceFlag)
	enc.WriteUint8(device.DeviceLevel)
	enc.WriteUint32(device.ConnCount)
	enc.WriteUint64(device.SendMsgCount)
	enc.WriteUint64(device.RecvMsgCount)
	enc.WriteUint64(device.SendMsgBytes)
	enc.WriteUint64(device.RecvMsgBytes)
	enc.WriteUint64(uint64(device.CreatedAt.UnixNano()))
	enc.WriteUint64(uint64(device.UpdatedAt.UnixNano()))
	return enc.Bytes(), nil
}

func decodeDeviceValue(data []byte) (wkdb.Device, error) {
	dec := wkproto.NewDecoder(data)
	var (
		device wkdb.Device
		err    error
	)
	if device.Id, err = dec.Uint64(); err != nil {
		return wkdb.Device{}, err
	}
	if device.Uid, err = dec.String(); err != nil {
		return wkdb.Device{}, err
	}
	if device.Token, err = dec.String(); err != nil {
		return wkdb.Device{}, err
	}
	if device.DeviceFlag, err = dec.Uint64(); err != nil {
		return wkdb.Device{}, err
	}
	if device.DeviceLevel, err = dec.Uint8(); err != nil {
		return wkdb.Device{}, err
	}
	if device.ConnCount, err = dec.Uint32(); err != nil {
		return wkdb.Device{}, err
	}
	if device.SendMsgCount, err = dec.Uint64(); err != nil {
		return wkdb.Device{}, err
	}
	if device.RecvMsgCount, err = dec.Uint64(); err != nil {
		return wkdb.Device{}, err
	}
	if device.SendMsgBytes, err = dec.Uint64(); err != nil {
		return wkdb.Device{}, err
	}
	if device.RecvMsgBytes, err = dec.Uint64(); err != nil {
		return wkdb.Device{}, err
	}
	createdAt, err := dec.Uint64()
	if err != nil {
		return wkdb.Device{}, err
	}
	updatedAt, err := dec.Uint64()
	if err != nil {
		return wkdb.Device{}, err
	}
	device.CreatedAt = unixNanoPtr(createdAt)
	device.UpdatedAt = unixNanoPtr(updatedAt)
	return device, nil
}

func matchDeviceSearch(device wkdb.Device, req wkdb.DeviceSearchReq) bool {
	ts := deviceSortTime(device)
	if req.OffsetCreatedAt <= 0 {
		return true
	}
	if req.Pre {
		return ts > req.OffsetCreatedAt
	}
	return ts < req.OffsetCreatedAt
}

func deviceSortTime(device wkdb.Device) int64 {
	if device.CreatedAt != nil {
		return device.CreatedAt.UnixNano()
	}
	return 0
}

func sortDevices(devices []wkdb.Device) {
	sort.SliceStable(devices, func(i, j int) bool {
		left := deviceSortTime(devices[i])
		right := deviceSortTime(devices[j])
		if left == right {
			if devices[i].Uid == devices[j].Uid {
				return devices[i].DeviceFlag < devices[j].DeviceFlag
			}
			return strings.Compare(devices[i].Uid, devices[j].Uid) < 0
		}
		return left > right
	})
}

func limitDevices(devices []wkdb.Device, limit int, pre bool) []wkdb.Device {
	if len(devices) == 0 {
		return nil
	}
	if limit <= 0 {
		out := make([]wkdb.Device, 0, len(devices))
		for _, device := range devices {
			out = append(out, cloneDevice(device))
		}
		return out
	}
	if len(devices) > limit {
		if pre {
			devices = devices[len(devices)-limit:]
		} else {
			devices = devices[:limit]
		}
	}

	out := make([]wkdb.Device, 0, len(devices))
	for _, device := range devices {
		out = append(out, cloneDevice(device))
	}
	return out
}

func cloneDevice(device wkdb.Device) wkdb.Device {
	device.CreatedAt = cloneTimePtr(device.CreatedAt)
	device.UpdatedAt = cloneTimePtr(device.UpdatedAt)
	return device
}
