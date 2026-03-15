package v3

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	"github.com/cockroachdb/pebble"
)

type channelMemberCounter uint8

const (
	channelMemberCounterSubscriber channelMemberCounter = iota + 1
	channelMemberCounterAllowlist
	channelMemberCounterDenylist
)

type pebbleChannelMemberStore struct {
	slotID  uint32
	buckets *PebbleBucketManager
	now     func() time.Time
	table   v3key.TableID
	counter channelMemberCounter
}

var _ SubscriberStore = (*pebbleChannelMemberStore)(nil)
var _ AllowlistStore = (*pebbleChannelMemberStore)(nil)
var _ DenylistStore = (*pebbleChannelMemberStore)(nil)

func (p *pebbleChannelMemberStore) List(channelID string, channelType uint8) ([]wkdb.Member, error) {
	bucket, err := p.bucket()
	if err != nil {
		return nil, err
	}
	lower, upper := channelMemberRangeForTable(p.table, p.slotID, channelID, channelType)
	iter := bucket.DB().NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	members := make([]wkdb.Member, 0)
	for iter.First(); iter.Valid(); iter.Next() {
		member, err := decodeMemberValue(iter.Value())
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	sort.SliceStable(members, func(i, j int) bool {
		return strings.Compare(members[i].Uid, members[j].Uid) < 0
	})
	return cloneMembers(members), nil
}

func (p *pebbleChannelMemberStore) Put(channelID string, channelType uint8, members []wkdb.Member) error {
	if strings.TrimSpace(channelID) == "" {
		return fmt.Errorf("channel id is required")
	}
	if len(members) == 0 {
		return nil
	}
	bucket, err := p.bucket()
	if err != nil {
		return err
	}
	channel, err := p.loadChannel(channelID, channelType)
	if err != nil {
		return err
	}
	normalized := dedupeMembers(members)
	batch := bucket.DB().NewBatch()
	defer batch.Close()

	var newCount int
	for _, member := range normalized {
		if strings.TrimSpace(member.Uid) == "" {
			return wkdb.ErrInvalidUserId
		}
		oldMember, err := p.getMember(bucket, channelID, channelType, member.Uid)
		if err != nil && err != wkdb.ErrNotFound {
			return err
		}
		existed := err == nil
		now := p.currentTime()
		member.Id = hashString(member.Uid)
		if existed && oldMember.CreatedAt != nil {
			member.CreatedAt = cloneTimePtr(oldMember.CreatedAt)
		} else if member.CreatedAt == nil {
			member.CreatedAt = cloneTimePtr(&now)
		}
		if member.UpdatedAt == nil {
			member.UpdatedAt = cloneTimePtr(&now)
		}
		value, err := encodeMemberValue(member)
		if err != nil {
			return err
		}
		if err := batch.Set(channelMemberRowKeyForTable(p.table, p.slotID, channelID, channelType, member.Uid), value, p.writeOptions()); err != nil {
			return err
		}
		if !existed {
			newCount++
		}
	}
	if newCount > 0 {
		p.adjustCount(&channel, newCount)
		if err := p.writeChannel(channel, batch); err != nil {
			return err
		}
	}
	return batch.Commit(p.writeOptions())
}

func (p *pebbleChannelMemberStore) Delete(channelID string, channelType uint8, uids []string) error {
	if len(uids) == 0 {
		return nil
	}
	bucket, err := p.bucket()
	if err != nil {
		return err
	}
	channel, err := p.loadChannel(channelID, channelType)
	if err != nil {
		if err == wkdb.ErrNotFound {
			return nil
		}
		return err
	}
	normalized := dedupeStrings(uids)
	batch := bucket.DB().NewBatch()
	defer batch.Close()

	var removed int
	for _, uid := range normalized {
		if strings.TrimSpace(uid) == "" {
			continue
		}
		_, err := p.getMember(bucket, channelID, channelType, uid)
		if err != nil {
			if err == wkdb.ErrNotFound {
				continue
			}
			return err
		}
		if err := batch.Delete(channelMemberRowKeyForTable(p.table, p.slotID, channelID, channelType, uid), p.writeOptions()); err != nil {
			return err
		}
		removed++
	}
	if removed > 0 {
		p.adjustCount(&channel, -removed)
		if err := p.writeChannel(channel, batch); err != nil {
			return err
		}
	}
	return batch.Commit(p.writeOptions())
}

func (p *pebbleChannelMemberStore) DeleteAll(channelID string, channelType uint8) error {
	bucket, err := p.bucket()
	if err != nil {
		return err
	}
	channel, err := p.loadChannel(channelID, channelType)
	if err != nil {
		if err == wkdb.ErrNotFound {
			return nil
		}
		return err
	}
	lower, upper := channelMemberRangeForTable(p.table, p.slotID, channelID, channelType)
	iter := bucket.DB().NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	var count int
	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}
	if err := iter.Error(); err != nil {
		return err
	}
	if count == 0 {
		return nil
	}

	batch := bucket.DB().NewBatch()
	defer batch.Close()
	if err := batch.DeleteRange(lower, upper, p.writeOptions()); err != nil {
		return err
	}
	p.adjustCount(&channel, -count)
	if err := p.writeChannel(channel, batch); err != nil {
		return err
	}
	return batch.Commit(p.writeOptions())
}

func (p *pebbleChannelMemberStore) bucket() (*PebbleBucket, error) {
	return p.buckets.Bucket(p.slotID)
}

func (p *pebbleChannelMemberStore) currentTime() time.Time {
	if p.now != nil {
		return p.now()
	}
	return time.Now()
}

func (p *pebbleChannelMemberStore) writeOptions() *pebble.WriteOptions {
	bucket, _ := p.bucket()
	if bucket == nil {
		return pebble.Sync
	}
	return bucket.SnapshotStore().writeOptions
}

func (p *pebbleChannelMemberStore) loadChannel(channelID string, channelType uint8) (wkdb.ChannelInfo, error) {
	channelStore := &pebbleChannelStore{slotID: p.slotID, buckets: p.buckets, now: p.now}
	return channelStore.Get(channelID, channelType)
}

func (p *pebbleChannelMemberStore) writeChannel(channel wkdb.ChannelInfo, writer pebble.Writer) error {
	channelStore := &pebbleChannelStore{slotID: p.slotID, buckets: p.buckets, now: p.now}
	return channelStore.writeChannelInfo(writer, channel)
}

func (p *pebbleChannelMemberStore) adjustCount(channel *wkdb.ChannelInfo, delta int) {
	switch p.counter {
	case channelMemberCounterSubscriber:
		channel.SubscriberCount = clampCount(channel.SubscriberCount + delta)
	case channelMemberCounterAllowlist:
		channel.AllowlistCount = clampCount(channel.AllowlistCount + delta)
	case channelMemberCounterDenylist:
		channel.DenylistCount = clampCount(channel.DenylistCount + delta)
	}
}

func (p *pebbleChannelMemberStore) getMember(bucket *PebbleBucket, channelID string, channelType uint8, uid string) (wkdb.Member, error) {
	value, closer, err := bucket.DB().Get(channelMemberRowKeyForTable(p.table, p.slotID, channelID, channelType, uid))
	if err != nil {
		if err == pebble.ErrNotFound {
			return wkdb.Member{}, wkdb.ErrNotFound
		}
		return wkdb.Member{}, err
	}
	defer closer.Close()
	return decodeMemberValue(value)
}

func channelMemberRowKeyForTable(table v3key.TableID, slotID uint32, channelID string, channelType uint8, uid string) []byte {
	switch table {
	case v3key.TableSubscriber:
		return v3key.EncodeSubscriberRowKey(slotID, channelID, channelType, uid)
	case v3key.TableAllowlist:
		return v3key.EncodeAllowlistRowKey(slotID, channelID, channelType, uid)
	case v3key.TableDenylist:
		return v3key.EncodeDenylistRowKey(slotID, channelID, channelType, uid)
	default:
		panic(fmt.Sprintf("unsupported channel member table: 0x%04x", uint16(table)))
	}
}

func channelMemberRangeForTable(table v3key.TableID, slotID uint32, channelID string, channelType uint8) (lower, upper []byte) {
	switch table {
	case v3key.TableSubscriber:
		return v3key.SubscriberChannelRange(slotID, channelID, channelType)
	case v3key.TableAllowlist:
		return v3key.AllowlistChannelRange(slotID, channelID, channelType)
	case v3key.TableDenylist:
		return v3key.DenylistChannelRange(slotID, channelID, channelType)
	default:
		panic(fmt.Sprintf("unsupported channel member table: 0x%04x", uint16(table)))
	}
}

func encodeMemberValue(member wkdb.Member) ([]byte, error) {
	if strings.TrimSpace(member.Uid) == "" {
		return nil, wkdb.ErrInvalidUserId
	}
	if member.CreatedAt == nil || member.UpdatedAt == nil {
		return nil, fmt.Errorf("member timestamps are required")
	}
	return member.Marshal()
}

func decodeMemberValue(data []byte) (wkdb.Member, error) {
	var member wkdb.Member
	if err := member.Unmarshal(data); err != nil {
		return wkdb.Member{}, err
	}
	return member, nil
}

func dedupeMembers(members []wkdb.Member) []wkdb.Member {
	if len(members) == 0 {
		return nil
	}
	indexByUID := make(map[string]int, len(members))
	out := make([]wkdb.Member, 0, len(members))
	for _, member := range members {
		if pos, ok := indexByUID[member.Uid]; ok {
			out[pos] = member
			continue
		}
		indexByUID[member.Uid] = len(out)
		out = append(out, member)
	}
	return out
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func clampCount(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func cloneMembers(members []wkdb.Member) []wkdb.Member {
	if len(members) == 0 {
		return nil
	}
	out := make([]wkdb.Member, 0, len(members))
	for _, member := range members {
		member.CreatedAt = cloneTimePtr(member.CreatedAt)
		member.UpdatedAt = cloneTimePtr(member.UpdatedAt)
		out = append(out, member)
	}
	return out
}

func hashString(value string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(value))
	return h.Sum64()
}
