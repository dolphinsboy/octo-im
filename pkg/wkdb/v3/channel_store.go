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

type pebbleChannelStore struct {
	slotID  uint32
	buckets *PebbleBucketManager
	now     func() time.Time
}

var _ ChannelStore = (*pebbleChannelStore)(nil)

func (p *pebbleChannelStore) Get(channelID string, channelType uint8) (wkdb.ChannelInfo, error) {
	bucket, err := p.bucket()
	if err != nil {
		return wkdb.ChannelInfo{}, err
	}
	value, closer, err := bucket.DB().Get(v3key.EncodeChannelInfoRowKey(p.slotID, channelID, channelType))
	if err != nil {
		if err == pebble.ErrNotFound {
			return wkdb.ChannelInfo{}, wkdb.ErrNotFound
		}
		return wkdb.ChannelInfo{}, err
	}
	defer closer.Close()
	return decodeChannelInfoValue(value)
}

func (p *pebbleChannelStore) Exists(channelID string, channelType uint8) (bool, error) {
	_, err := p.Get(channelID, channelType)
	if err == nil {
		return true, nil
	}
	if err == wkdb.ErrNotFound {
		return false, nil
	}
	return false, err
}

func (p *pebbleChannelStore) Put(info wkdb.ChannelInfo) error {
	if strings.TrimSpace(info.ChannelId) == "" {
		return fmt.Errorf("channel id is required")
	}
	bucket, err := p.bucket()
	if err != nil {
		return err
	}
	oldInfo, err := p.Get(info.ChannelId, info.ChannelType)
	if err != nil && err != wkdb.ErrNotFound {
		return err
	}
	existed := err == nil
	now := p.currentTime()
	if existed && oldInfo.CreatedAt != nil {
		info.CreatedAt = cloneTimePtr(oldInfo.CreatedAt)
	} else if info.CreatedAt == nil {
		info.CreatedAt = cloneTimePtr(&now)
	}
	if info.UpdatedAt == nil {
		info.UpdatedAt = cloneTimePtr(&now)
	}
	if existed {
		info.SubscriberCount = oldInfo.SubscriberCount
		info.AllowlistCount = oldInfo.AllowlistCount
		info.DenylistCount = oldInfo.DenylistCount
	}

	batch := bucket.DB().NewBatch()
	defer batch.Close()
	if existed && oldInfo.CreatedAt != nil {
		if err := batch.Delete(v3key.EncodeChannelInfoCreatedAtSecondIndexKey(p.slotID, uint64(oldInfo.CreatedAt.UnixNano()), oldInfo.ChannelId, oldInfo.ChannelType), p.writeOptions()); err != nil {
			return err
		}
	}
	if err := p.writeChannelInfo(batch, info); err != nil {
		return err
	}
	return batch.Commit(p.writeOptions())
}

func (p *pebbleChannelStore) Delete(channelID string, channelType uint8) error {
	info, err := p.Get(channelID, channelType)
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

	if err := batch.Delete(v3key.EncodeChannelInfoRowKey(p.slotID, channelID, channelType), p.writeOptions()); err != nil {
		return err
	}
	if info.CreatedAt != nil {
		if err := batch.Delete(v3key.EncodeChannelInfoCreatedAtSecondIndexKey(p.slotID, uint64(info.CreatedAt.UnixNano()), channelID, channelType), p.writeOptions()); err != nil {
			return err
		}
	}
	for _, table := range []v3key.TableID{v3key.TableSubscriber, v3key.TableAllowlist, v3key.TableDenylist} {
		lower, upper := channelMemberRangeForTable(table, p.slotID, channelID, channelType)
		if err := batch.DeleteRange(lower, upper, p.writeOptions()); err != nil {
			return err
		}
	}
	return batch.Commit(p.writeOptions())
}

func (p *pebbleChannelStore) Search(req wkdb.ChannelSearchReq) ([]wkdb.ChannelInfo, error) {
	if req.ChannelId != "" && req.ChannelType != 0 {
		info, err := p.Get(req.ChannelId, req.ChannelType)
		if err != nil {
			return nil, err
		}
		if !matchChannelSearch(info, req) {
			return nil, nil
		}
		return limitChannelInfos([]wkdb.ChannelInfo{info}, req.Limit, req.Pre), nil
	}
	bucket, err := p.bucket()
	if err != nil {
		return nil, err
	}
	lower, upper := v3key.SlotTableRange(p.slotID, v3key.ScopeSlotPrimary, v3key.TableChannelInfo)
	iter := bucket.DB().NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	channels := make([]wkdb.ChannelInfo, 0)
	for iter.First(); iter.Valid(); iter.Next() {
		info, err := decodeChannelInfoValue(iter.Value())
		if err != nil {
			return nil, err
		}
		if !matchChannelSearch(info, req) {
			continue
		}
		channels = append(channels, info)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	sort.SliceStable(channels, func(i, j int) bool {
		left := channelSortTime(channels[i])
		right := channelSortTime(channels[j])
		if left == right {
			return strings.Compare(wkdb.ChannelToKey(channels[i].ChannelId, channels[i].ChannelType), wkdb.ChannelToKey(channels[j].ChannelId, channels[j].ChannelType)) < 0
		}
		return left > right
	})
	return limitChannelInfos(channels, req.Limit, req.Pre), nil
}

func (p *pebbleChannelStore) bucket() (*PebbleBucket, error) {
	return p.buckets.Bucket(p.slotID)
}

func (p *pebbleChannelStore) currentTime() time.Time {
	if p.now != nil {
		return p.now()
	}
	return time.Now()
}

func (p *pebbleChannelStore) writeOptions() *pebble.WriteOptions {
	bucket, _ := p.bucket()
	if bucket == nil {
		return pebble.Sync
	}
	return bucket.SnapshotStore().writeOptions
}

func (p *pebbleChannelStore) writeChannelInfo(writer pebble.Writer, info wkdb.ChannelInfo) error {
	value, err := encodeChannelInfoValue(info)
	if err != nil {
		return err
	}
	if err := writer.Set(v3key.EncodeChannelInfoRowKey(p.slotID, info.ChannelId, info.ChannelType), value, p.writeOptions()); err != nil {
		return err
	}
	if info.CreatedAt != nil {
		if err := writer.Set(v3key.EncodeChannelInfoCreatedAtSecondIndexKey(p.slotID, uint64(info.CreatedAt.UnixNano()), info.ChannelId, info.ChannelType), nil, p.writeOptions()); err != nil {
			return err
		}
	}
	return nil
}

func encodeChannelInfoValue(info wkdb.ChannelInfo) ([]byte, error) {
	if strings.TrimSpace(info.ChannelId) == "" {
		return nil, fmt.Errorf("channel id is required")
	}
	if info.CreatedAt == nil || info.UpdatedAt == nil {
		return nil, fmt.Errorf("channel timestamps are required")
	}
	enc := wkproto.NewEncoder()
	defer enc.End()
	enc.WriteUint64(info.Id)
	enc.WriteString(info.ChannelId)
	enc.WriteUint8(info.ChannelType)
	enc.WriteUint8(boolToByte(info.Ban))
	enc.WriteUint8(boolToByte(info.Large))
	enc.WriteUint8(boolToByte(info.Disband))
	enc.WriteUint32(uint32(info.SubscriberCount))
	enc.WriteUint32(uint32(info.DenylistCount))
	enc.WriteUint32(uint32(info.AllowlistCount))
	enc.WriteUint64(info.LastMsgSeq)
	enc.WriteUint64(info.LastMsgTime)
	enc.WriteString(info.Webhook)
	enc.WriteUint8(boolToByte(info.SendBan))
	enc.WriteUint8(boolToByte(info.AllowStranger))
	enc.WriteUint64(uint64(info.CreatedAt.UnixNano()))
	enc.WriteUint64(uint64(info.UpdatedAt.UnixNano()))
	return enc.Bytes(), nil
}

func decodeChannelInfoValue(data []byte) (wkdb.ChannelInfo, error) {
	dec := wkproto.NewDecoder(data)
	var (
		info wkdb.ChannelInfo
		err  error
	)
	if info.Id, err = dec.Uint64(); err != nil {
		return wkdb.ChannelInfo{}, err
	}
	if info.ChannelId, err = dec.String(); err != nil {
		return wkdb.ChannelInfo{}, err
	}
	if info.ChannelType, err = dec.Uint8(); err != nil {
		return wkdb.ChannelInfo{}, err
	}
	ban, err := dec.Uint8()
	if err != nil {
		return wkdb.ChannelInfo{}, err
	}
	info.Ban = byteToBool(ban)
	large, err := dec.Uint8()
	if err != nil {
		return wkdb.ChannelInfo{}, err
	}
	info.Large = byteToBool(large)
	disband, err := dec.Uint8()
	if err != nil {
		return wkdb.ChannelInfo{}, err
	}
	info.Disband = byteToBool(disband)
	subscriberCount, err := dec.Uint32()
	if err != nil {
		return wkdb.ChannelInfo{}, err
	}
	info.SubscriberCount = int(subscriberCount)
	denylistCount, err := dec.Uint32()
	if err != nil {
		return wkdb.ChannelInfo{}, err
	}
	info.DenylistCount = int(denylistCount)
	allowlistCount, err := dec.Uint32()
	if err != nil {
		return wkdb.ChannelInfo{}, err
	}
	info.AllowlistCount = int(allowlistCount)
	if info.LastMsgSeq, err = dec.Uint64(); err != nil {
		return wkdb.ChannelInfo{}, err
	}
	if info.LastMsgTime, err = dec.Uint64(); err != nil {
		return wkdb.ChannelInfo{}, err
	}
	if info.Webhook, err = dec.String(); err != nil {
		return wkdb.ChannelInfo{}, err
	}
	sendBan, err := dec.Uint8()
	if err != nil {
		return wkdb.ChannelInfo{}, err
	}
	info.SendBan = byteToBool(sendBan)
	allowStranger, err := dec.Uint8()
	if err != nil {
		return wkdb.ChannelInfo{}, err
	}
	info.AllowStranger = byteToBool(allowStranger)
	createdAt, err := dec.Uint64()
	if err != nil {
		return wkdb.ChannelInfo{}, err
	}
	updatedAt, err := dec.Uint64()
	if err != nil {
		return wkdb.ChannelInfo{}, err
	}
	info.CreatedAt = unixNanoPtr(createdAt)
	info.UpdatedAt = unixNanoPtr(updatedAt)
	return info, nil
}

func matchChannelSearch(info wkdb.ChannelInfo, req wkdb.ChannelSearchReq) bool {
	if req.ChannelId != "" && req.ChannelId != info.ChannelId {
		return false
	}
	if req.ChannelType != 0 && req.ChannelType != info.ChannelType {
		return false
	}
	if req.Ban != nil && *req.Ban != info.Ban {
		return false
	}
	if req.Disband != nil && *req.Disband != info.Disband {
		return false
	}
	if req.SubscriberCountGte != nil && info.SubscriberCount < *req.SubscriberCountGte {
		return false
	}
	if req.SubscriberCountLte != nil && info.SubscriberCount > *req.SubscriberCountLte {
		return false
	}
	if req.DenylistCountGte != nil && info.DenylistCount < *req.DenylistCountGte {
		return false
	}
	if req.DenylistCountLte != nil && info.DenylistCount > *req.DenylistCountLte {
		return false
	}
	if req.AllowlistCountGte != nil && info.AllowlistCount < *req.AllowlistCountGte {
		return false
	}
	if req.AllowlistCountLte != nil && info.AllowlistCount > *req.AllowlistCountLte {
		return false
	}
	ts := channelSortTime(info)
	if req.OffsetCreatedAt <= 0 {
		return true
	}
	if req.Pre {
		return ts > req.OffsetCreatedAt
	}
	return ts < req.OffsetCreatedAt
}

func channelSortTime(info wkdb.ChannelInfo) int64 {
	if info.CreatedAt != nil {
		return info.CreatedAt.UnixNano()
	}
	return 0
}

func limitChannelInfos(channels []wkdb.ChannelInfo, limit int, pre bool) []wkdb.ChannelInfo {
	if len(channels) == 0 {
		return nil
	}
	if limit > 0 && len(channels) > limit {
		if pre {
			channels = channels[len(channels)-limit:]
		} else {
			channels = channels[:limit]
		}
	}
	out := make([]wkdb.ChannelInfo, 0, len(channels))
	for _, info := range channels {
		out = append(out, cloneChannelInfo(info))
	}
	return out
}

func cloneChannelInfo(info wkdb.ChannelInfo) wkdb.ChannelInfo {
	info.CreatedAt = cloneTimePtr(info.CreatedAt)
	info.UpdatedAt = cloneTimePtr(info.UpdatedAt)
	return info
}

func boolToByte(v bool) uint8 {
	if v {
		return 1
	}
	return 0
}

func byteToBool(v uint8) bool {
	return v != 0
}
