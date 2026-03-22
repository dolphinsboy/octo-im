package v3

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"math"
	"sort"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	"github.com/WuKongIM/WuKongIM/pkg/wkutil"
	"github.com/cockroachdb/pebble"
)

const (
	channelLogKeyMessageRow  byte = 0x01
	channelLogKeyLastSeq     byte = 0x02
	channelLogKeyMessageID   byte = 0x03
	channelLogKeyFromUID     byte = 0x04
	channelLogKeyClientMsgNo byte = 0x05
	channelLogKeyLeaderTerm  byte = 0x06
)

type pebbleChannelLogStore struct {
	owner   *PebbleDB
	buckets *PebbleBucketManager
}

var _ ChannelLogStore = (*pebbleChannelLogStore)(nil)

func (p *pebbleChannelLogStore) GetMessage(messageID uint64) (wkdb.Message, error) {
	if messageID == 0 {
		return wkdb.EmptyMessage, wkdb.ErrNotFound
	}
	rowKey, bucket, err := p.lookupMessageRowKeyByID(messageID)
	if err != nil {
		return wkdb.EmptyMessage, err
	}
	return p.loadMessageByRowKey(bucket.DB(), rowKey)
}

func (p *pebbleChannelLogStore) AppendMessages(channelID string, channelType uint8, msgs []wkdb.Message) error {
	if len(msgs) == 0 {
		return nil
	}
	bucket, err := p.bucketForChannel(channelID, channelType)
	if err != nil {
		return err
	}
	batch := bucket.DB().NewBatch()
	defer batch.Close()

	for _, msg := range msgs {
		msg.ChannelID = channelID
		msg.ChannelType = channelType
		rowKey, value, err := encodeStoredChannelMessage(channelID, channelType, msg)
		if err != nil {
			return err
		}
		if err := batch.Set(rowKey, value, p.writeOptions()); err != nil {
			return err
		}
		if msg.MessageID > 0 {
			if err := batch.Set(encodeChannelMessageIDIndexKey(uint64(msg.MessageID)), rowKey, p.writeOptions()); err != nil {
				return err
			}
		}
		if msg.FromUID != "" {
			if err := batch.Set(encodeChannelMessageFromUIDIndexKey(msg.FromUID, channelID, channelType, uint64(msg.MessageSeq)), rowKey, p.writeOptions()); err != nil {
				return err
			}
		}
		if msg.ClientMsgNo != "" {
			if err := batch.Set(encodeChannelMessageClientMsgNoIndexKey(msg.ClientMsgNo, channelID, channelType, uint64(msg.MessageSeq)), rowKey, p.writeOptions()); err != nil {
				return err
			}
		}
	}
	lastMsg := msgs[len(msgs)-1]
	if err := batch.Set(encodeChannelLastSeqKey(channelID, channelType), encodeLastSeqValue(uint64(lastMsg.MessageSeq), uint64(p.owner.currentTime().UnixNano())), p.writeOptions()); err != nil {
		return err
	}
	return batch.Commit(p.writeOptions())
}

func (p *pebbleChannelLogStore) LoadPrevRangeMsgs(channelID string, channelType uint8, startMessageSeq, endMessageSeq uint64, limit int) ([]wkdb.Message, error) {
	if startMessageSeq == 0 {
		return nil, fmt.Errorf("start messageSeq[%d] must be greater than 0", startMessageSeq)
	}
	if endMessageSeq != 0 && endMessageSeq > startMessageSeq {
		return nil, fmt.Errorf("end messageSeq[%d] must be less than start messageSeq[%d]", endMessageSeq, startMessageSeq)
	}

	var minSeq uint64
	maxSeq := startMessageSeq + 1
	if endMessageSeq == 0 {
		if startMessageSeq < uint64(limit) || limit <= 0 {
			minSeq = 1
		} else {
			minSeq = startMessageSeq - uint64(limit) + 1
		}
	} else if limit > 0 && startMessageSeq-endMessageSeq > uint64(limit) {
		minSeq = startMessageSeq - uint64(limit) + 1
	} else {
		minSeq = endMessageSeq + 1
	}

	lastSeq, _, err := p.GetChannelLastMessageSeq(channelID, channelType)
	if err != nil {
		return nil, err
	}
	if maxSeq > lastSeq+1 {
		maxSeq = lastSeq + 1
	}
	return p.loadChannelMessages(channelID, channelType, minSeq, maxSeq, limit, false)
}

func (p *pebbleChannelLogStore) LoadNextRangeMsgs(channelID string, channelType uint8, startMessageSeq, endMessageSeq uint64, limit int) ([]wkdb.Message, error) {
	maxSeq := endMessageSeq
	if endMessageSeq == 0 {
		maxSeq = math.MaxUint64
	}
	lastSeq, _, err := p.GetChannelLastMessageSeq(channelID, channelType)
	if err != nil {
		return nil, err
	}
	if maxSeq > lastSeq+1 {
		maxSeq = lastSeq + 1
	}
	return p.loadChannelMessages(channelID, channelType, startMessageSeq, maxSeq, limit, false)
}

func (p *pebbleChannelLogStore) LoadNextRangeMsgsForSize(channelID string, channelType uint8, startMessageSeq, endMessageSeq, limitSize uint64) ([]wkdb.Message, error) {
	return p.loadChannelMessagesForSize(channelID, channelType, startMessageSeq, endMessageSeq, limitSize)
}

func (p *pebbleChannelLogStore) LoadMsg(channelID string, channelType uint8, seq uint64) (wkdb.Message, error) {
	bucket, err := p.bucketForChannel(channelID, channelType)
	if err != nil {
		return wkdb.EmptyMessage, err
	}
	value, closer, err := bucket.DB().Get(encodeChannelMessageRowKey(channelID, channelType, seq))
	if err != nil {
		if err == pebble.ErrNotFound {
			return wkdb.EmptyMessage, wkdb.ErrNotFound
		}
		return wkdb.EmptyMessage, err
	}
	defer closer.Close()
	return decodeStoredChannelMessage(value)
}

func (p *pebbleChannelLogStore) TruncateLogTo(channelID string, channelType uint8, messageSeq uint64) error {
	if messageSeq == 0 {
		return fmt.Errorf("messageSeq[%d] must be greater than 0", messageSeq)
	}
	lastSeq, _, err := p.GetChannelLastMessageSeq(channelID, channelType)
	if err != nil {
		return err
	}
	if messageSeq >= lastSeq {
		return nil
	}
	bucket, err := p.bucketForChannel(channelID, channelType)
	if err != nil {
		return err
	}
	lower, upper := channelMessageRowRange(channelID, channelType, messageSeq+1, 0)
	iter := bucket.DB().NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	defer iter.Close()

	batch := bucket.DB().NewBatch()
	defer batch.Close()
	for iter.First(); iter.Valid(); iter.Next() {
		var msg wkdb.Message
		if err := msg.Unmarshal(iter.Value()); err != nil {
			return err
		}
		rowKey := append([]byte(nil), iter.Key()...)
		if msg.MessageID > 0 {
			if err := batch.Delete(encodeChannelMessageIDIndexKey(uint64(msg.MessageID)), p.writeOptions()); err != nil {
				return err
			}
		}
		if msg.FromUID != "" {
			if err := batch.Delete(encodeChannelMessageFromUIDIndexKey(msg.FromUID, channelID, channelType, uint64(msg.MessageSeq)), p.writeOptions()); err != nil {
				return err
			}
		}
		if msg.ClientMsgNo != "" {
			if err := batch.Delete(encodeChannelMessageClientMsgNoIndexKey(msg.ClientMsgNo, channelID, channelType, uint64(msg.MessageSeq)), p.writeOptions()); err != nil {
				return err
			}
		}
		if err := batch.Delete(rowKey, p.writeOptions()); err != nil {
			return err
		}
	}
	if err := batch.Set(encodeChannelLastSeqKey(channelID, channelType), encodeLastSeqValue(messageSeq, uint64(p.owner.currentTime().UnixNano())), p.writeOptions()); err != nil {
		return err
	}
	return batch.Commit(p.writeOptions())
}

func (p *pebbleChannelLogStore) LoadLastMsgsWithEnd(channelID string, channelType uint8, endMessageSeq uint64, limit int) ([]wkdb.Message, error) {
	lastSeq, _, err := p.GetChannelLastMessageSeq(channelID, channelType)
	if err != nil {
		return nil, err
	}
	if lastSeq == 0 || endMessageSeq > lastSeq {
		return nil, nil
	}
	return p.LoadPrevRangeMsgs(channelID, channelType, lastSeq, endMessageSeq, limit)
}

func (p *pebbleChannelLogStore) LoadLastMsgs(channelID string, channelType uint8, limit int) ([]wkdb.Message, error) {
	lastSeq, _, err := p.GetChannelLastMessageSeq(channelID, channelType)
	if err != nil {
		return nil, err
	}
	if lastSeq == 0 {
		return nil, nil
	}
	return p.LoadPrevRangeMsgs(channelID, channelType, lastSeq, 0, limit)
}

func (p *pebbleChannelLogStore) GetChannelLastMessageSeq(channelID string, channelType uint8) (uint64, uint64, error) {
	bucket, err := p.bucketForChannel(channelID, channelType)
	if err != nil {
		return 0, 0, err
	}
	value, closer, err := bucket.DB().Get(encodeChannelLastSeqKey(channelID, channelType))
	if err != nil {
		if err == pebble.ErrNotFound {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	defer closer.Close()
	return decodeLastSeqValue(value)
}

func (p *pebbleChannelLogStore) SetChannelLastMessageSeq(channelID string, channelType uint8, seq uint64) error {
	bucket, err := p.bucketForChannel(channelID, channelType)
	if err != nil {
		return err
	}
	return bucket.DB().Set(encodeChannelLastSeqKey(channelID, channelType), encodeLastSeqValue(seq, uint64(p.owner.currentTime().UnixNano())), p.writeOptions())
}

func (p *pebbleChannelLogStore) SearchMessages(req wkdb.MessageSearchReq) ([]wkdb.Message, error) {
	if req.MessageId > 0 {
		msg, err := p.GetMessage(uint64(req.MessageId))
		if err != nil {
			if err == wkdb.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}
		if !matchMessageSearchReq(msg, req) {
			return nil, nil
		}
		return []wkdb.Message{msg}, nil
	}

	if req.ChannelId != "" && req.ChannelType != 0 {
		return p.searchChannelMessages(req)
	}

	var messages []wkdb.Message
	switch {
	case req.FromUid != "":
		if err := p.collectMessagesByRowKeyIndex(func(bucket *PebbleBucket) ([]wkdb.Message, error) {
			lower, upper := channelMessageFromUIDIndexRange(req.FromUid)
			return p.scanMessagesByIndex(bucket, lower, upper)
		}, &messages); err != nil {
			return nil, err
		}
	case req.ClientMsgNo != "":
		if err := p.collectMessagesByRowKeyIndex(func(bucket *PebbleBucket) ([]wkdb.Message, error) {
			lower, upper := channelMessageClientMsgNoIndexRange(req.ClientMsgNo)
			return p.scanMessagesByIndex(bucket, lower, upper)
		}, &messages); err != nil {
			return nil, err
		}
	default:
		if err := p.eachBucket(func(bucket *PebbleBucket) error {
			rows, err := p.scanAllMessages(bucket)
			if err != nil {
				return err
			}
			messages = append(messages, rows...)
			return nil
		}); err != nil {
			return nil, err
		}
	}

	filtered := make([]wkdb.Message, 0, len(messages))
	for _, msg := range messages {
		if !matchMessageSearchReq(msg, req) {
			continue
		}
		filtered = append(filtered, msg)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].MessageID > filtered[j].MessageID
	})
	if req.OffsetMessageId > 0 {
		next := filtered[:0]
		for _, msg := range filtered {
			if req.Pre {
				if msg.MessageID > req.OffsetMessageId {
					next = append(next, msg)
				}
				continue
			}
			if msg.MessageID < req.OffsetMessageId {
				next = append(next, msg)
			}
		}
		filtered = next
	}
	if req.Limit > 0 && len(filtered) > req.Limit {
		if req.Pre {
			filtered = filtered[len(filtered)-req.Limit:]
		} else {
			filtered = filtered[:req.Limit]
		}
	}
	if len(filtered) == 0 {
		return nil, nil
	}
	return filtered, nil
}

func (p *pebbleChannelLogStore) CountMessages() (int, error) {
	count := 0
	err := p.eachBucket(func(bucket *PebbleBucket) error {
		iter := bucket.DB().NewIter(&pebble.IterOptions{
			LowerBound: []byte{channelLogKeyMessageRow},
			UpperBound: []byte{channelLogKeyMessageRow + 1},
		})
		defer iter.Close()
		for iter.First(); iter.Valid(); iter.Next() {
			count++
		}
		return nil
	})
	return count, err
}

func (p *pebbleChannelLogStore) GetLastMsg(channelID string, channelType uint8) (wkdb.Message, error) {
	lastSeq, _, err := p.GetChannelLastMessageSeq(channelID, channelType)
	if err != nil {
		return wkdb.EmptyMessage, err
	}
	if lastSeq == 0 {
		return wkdb.EmptyMessage, nil
	}
	return p.LoadMsg(channelID, channelType, lastSeq)
}

func (p *pebbleChannelLogStore) LoadMsgByClientMsgNo(channelID string, channelType uint8, clientMsgNo string) (wkdb.Message, error) {
	if clientMsgNo == "" {
		return wkdb.EmptyMessage, fmt.Errorf("clientMsgNo is empty")
	}
	bucket, err := p.bucketForChannel(channelID, channelType)
	if err != nil {
		return wkdb.EmptyMessage, err
	}
	lower, upper := channelMessageClientMsgNoIndexRangeForChannel(clientMsgNo, channelID, channelType)
	iter := bucket.DB().NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	defer iter.Close()
	if !iter.First() {
		return wkdb.EmptyMessage, wkdb.ErrNotFound
	}
	return p.loadMessageByRowKey(bucket.DB(), iter.Value())
}

func (p *pebbleChannelLogStore) GetUserLastMsgSeq(fromUID string, channelID string, channelType uint8) (uint64, error) {
	if fromUID == "" {
		return 0, nil
	}
	bucket, err := p.bucketForChannel(channelID, channelType)
	if err != nil {
		return 0, err
	}
	lower, upper := channelMessageFromUIDIndexRangeForChannel(fromUID, channelID, channelType)
	iter := bucket.DB().NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	defer iter.Close()
	if !iter.Last() {
		return 0, nil
	}
	_, _, seq, err := parseChannelMessageRowKey(iter.Value())
	if err != nil {
		return 0, err
	}
	return seq, nil
}

func (p *pebbleChannelLogStore) LoadMsgsBatch(requests []wkdb.BatchMsgRequest) ([]wkdb.BatchMsgResponse, error) {
	if len(requests) == 0 {
		return nil, nil
	}
	results := make([]wkdb.BatchMsgResponse, 0, len(requests))
	for _, req := range requests {
		var (
			msgs []wkdb.Message
			err  error
		)
		if req.OrderByLast {
			msgs, err = p.LoadLastMsgsWithEnd(req.ChannelId, req.ChannelType, req.MsgSeq, req.Limit)
		} else {
			msgs, err = p.LoadNextRangeMsgs(req.ChannelId, req.ChannelType, req.MsgSeq, 0, req.Limit)
		}
		if err != nil {
			return nil, err
		}
		results = append(results, wkdb.BatchMsgResponse{
			ChannelId:   req.ChannelId,
			ChannelType: req.ChannelType,
			Messages:    msgs,
		})
	}
	return results, nil
}

func (p *pebbleChannelLogStore) GetUserLastMsgSeqBatch(fromUID string, channels []wkdb.Channel) (map[string]uint64, error) {
	if len(channels) == 0 {
		return nil, nil
	}
	result := make(map[string]uint64, len(channels))
	for _, ch := range channels {
		seq, err := p.GetUserLastMsgSeq(fromUID, ch.ChannelId, ch.ChannelType)
		if err != nil {
			return nil, err
		}
		result[ch.ChannelId+":"+string(rune(ch.ChannelType))] = seq
	}
	return result, nil
}

func (p *pebbleChannelLogStore) SetLeaderTermStartIndex(shardNo string, term uint32, index uint64) error {
	bucket, err := p.bucketForShard(shardNo)
	if err != nil {
		return err
	}
	value := make([]byte, 8)
	binary.BigEndian.PutUint64(value, index)
	return bucket.DB().Set(encodeChannelLeaderTermKey(shardNo, term), value, p.writeOptions())
}

func (p *pebbleChannelLogStore) LeaderTermStartIndex(shardNo string, term uint32) (uint64, error) {
	bucket, err := p.bucketForShard(shardNo)
	if err != nil {
		return 0, err
	}
	value, closer, err := bucket.DB().Get(encodeChannelLeaderTermKey(shardNo, term))
	if err != nil {
		if err == pebble.ErrNotFound {
			return 0, nil
		}
		return 0, err
	}
	defer closer.Close()
	if len(value) != 8 {
		return 0, fmt.Errorf("leader term start index value size mismatch: %d", len(value))
	}
	return binary.BigEndian.Uint64(value), nil
}

func (p *pebbleChannelLogStore) LeaderLastTerm(shardNo string) (uint32, error) {
	bucket, err := p.bucketForShard(shardNo)
	if err != nil {
		return 0, err
	}
	lower, upper := channelLeaderTermRange(shardNo)
	iter := bucket.DB().NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	defer iter.Close()
	if !iter.Last() {
		return 0, nil
	}
	term, err := parseChannelLeaderTermKey(iter.Key())
	if err != nil {
		return 0, err
	}
	return term, nil
}

func (p *pebbleChannelLogStore) LeaderLastTermGreaterEqThan(shardNo string, term uint32) (uint32, error) {
	bucket, err := p.bucketForShard(shardNo)
	if err != nil {
		return 0, err
	}
	lower := encodeChannelLeaderTermKey(shardNo, term)
	_, upper := channelLeaderTermRange(shardNo)
	iter := bucket.DB().NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	defer iter.Close()
	if !iter.First() {
		return term, nil
	}
	return parseChannelLeaderTermKey(iter.Key())
}

func (p *pebbleChannelLogStore) DeleteLeaderTermStartIndexGreaterThanTerm(shardNo string, term uint32) error {
	bucket, err := p.bucketForShard(shardNo)
	if err != nil {
		return err
	}
	lower := encodeChannelLeaderTermKey(shardNo, term+1)
	_, upper := channelLeaderTermRange(shardNo)
	return bucket.DB().DeleteRange(lower, upper, p.writeOptions())
}

func (p *pebbleChannelLogStore) bucketForChannel(channelID string, channelType uint8) (*PebbleBucket, error) {
	return p.bucketForChannelKey(wkutil.ChannelToKey(channelID, channelType))
}

func (p *pebbleChannelLogStore) bucketForShard(shardNo string) (*PebbleBucket, error) {
	return p.bucketForChannelKey(shardNo)
}

func (p *pebbleChannelLogStore) bucketForChannelKey(channelKey string) (*PebbleBucket, error) {
	hash := crc32.ChecksumIEEE([]byte(channelKey))
	return p.buckets.Bucket(hash)
}

func (p *pebbleChannelLogStore) eachBucket(fn func(bucket *PebbleBucket) error) error {
	if p.buckets == nil {
		return fmt.Errorf("channel bucket manager is not initialized")
	}
	for bucketID := uint32(0); bucketID < p.buckets.BucketCount(); bucketID++ {
		bucket, err := p.buckets.BucketByID(bucketID)
		if err != nil {
			return err
		}
		if err := fn(bucket); err != nil {
			return err
		}
	}
	return nil
}

func (p *pebbleChannelLogStore) writeOptions() *pebble.WriteOptions {
	return p.owner.standaloneWriteOptions()
}

func (p *pebbleChannelLogStore) loadMessageByRowKey(db *pebble.DB, rowKey []byte) (wkdb.Message, error) {
	value, closer, err := db.Get(rowKey)
	if err != nil {
		if err == pebble.ErrNotFound {
			return wkdb.EmptyMessage, wkdb.ErrNotFound
		}
		return wkdb.EmptyMessage, err
	}
	defer closer.Close()
	return decodeStoredChannelMessage(value)
}

func (p *pebbleChannelLogStore) lookupMessageRowKeyByID(messageID uint64) ([]byte, *PebbleBucket, error) {
	indexKey := encodeChannelMessageIDIndexKey(messageID)
	var (
		foundRowKey []byte
		foundBucket *PebbleBucket
	)
	err := p.eachBucket(func(bucket *PebbleBucket) error {
		rowKey, closer, err := bucket.DB().Get(indexKey)
		if err != nil {
			if err == pebble.ErrNotFound {
				return nil
			}
			return err
		}
		defer closer.Close()
		foundRowKey = append([]byte(nil), rowKey...)
		foundBucket = bucket
		return stopIterating
	})
	if err != nil && err != stopIterating {
		return nil, nil, err
	}
	if foundBucket == nil {
		return nil, nil, wkdb.ErrNotFound
	}
	return foundRowKey, foundBucket, nil
}

func (p *pebbleChannelLogStore) loadChannelMessages(channelID string, channelType uint8, startMessageSeq, endMessageSeq uint64, limit int, reverse bool) ([]wkdb.Message, error) {
	bucket, err := p.bucketForChannel(channelID, channelType)
	if err != nil {
		return nil, err
	}
	lower, upper := channelMessageRowRange(channelID, channelType, startMessageSeq, endMessageSeq)
	iter := bucket.DB().NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	defer iter.Close()

	msgs := make([]wkdb.Message, 0)
	if reverse {
		for ok := iter.Last(); ok; ok = iter.Prev() {
			msg, err := decodeStoredChannelMessage(iter.Value())
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, msg)
			if limit > 0 && len(msgs) >= limit {
				break
			}
		}
		return msgs, nil
	}
	for ok := iter.First(); ok; ok = iter.Next() {
		msg, err := decodeStoredChannelMessage(iter.Value())
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
		if limit > 0 && len(msgs) >= limit {
			break
		}
	}
	return msgs, nil
}

func (p *pebbleChannelLogStore) loadChannelMessagesForSize(channelID string, channelType uint8, startMessageSeq, endMessageSeq, limitSize uint64) ([]wkdb.Message, error) {
	bucket, err := p.bucketForChannel(channelID, channelType)
	if err != nil {
		return nil, err
	}
	lower, upper := channelMessageRowRange(channelID, channelType, startMessageSeq, endMessageSeq)
	iter := bucket.DB().NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	defer iter.Close()

	msgs := make([]wkdb.Message, 0)
	var size uint64
	for ok := iter.First(); ok; ok = iter.Next() {
		msg, err := decodeStoredChannelMessage(iter.Value())
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
		size += uint64(msg.Size())
		if limitSize > 0 && size >= limitSize {
			break
		}
	}
	if len(msgs) == 0 {
		return nil, nil
	}
	return msgs, nil
}

func (p *pebbleChannelLogStore) searchChannelMessages(req wkdb.MessageSearchReq) ([]wkdb.Message, error) {
	var (
		startSeq = req.OffsetMessageSeq
		endSeq   uint64
		reverse  = !req.Pre
	)
	if req.Pre {
		if req.OffsetMessageSeq > 0 {
			startSeq = req.OffsetMessageSeq + 1
		}
		endSeq = 0
	} else if req.OffsetMessageSeq > 0 {
		startSeq = 0
		endSeq = req.OffsetMessageSeq
	}
	msgs, err := p.loadChannelMessages(req.ChannelId, req.ChannelType, startSeq, endSeq, 0, reverse)
	if err != nil {
		return nil, err
	}
	filtered := make([]wkdb.Message, 0, len(msgs))
	for _, msg := range msgs {
		if req.Pre {
			if req.OffsetMessageId > 0 && msg.MessageID <= req.OffsetMessageId {
				break
			}
		} else if req.OffsetMessageId > 0 && msg.MessageID >= req.OffsetMessageId {
			break
		}
		if !matchMessageSearchReq(msg, req) {
			continue
		}
		filtered = append(filtered, msg)
		if req.Limit > 0 && len(filtered) >= req.Limit {
			break
		}
	}
	if len(filtered) == 0 {
		return nil, nil
	}
	return filtered, nil
}

func (p *pebbleChannelLogStore) collectMessagesByRowKeyIndex(fetch func(bucket *PebbleBucket) ([]wkdb.Message, error), out *[]wkdb.Message) error {
	return p.eachBucket(func(bucket *PebbleBucket) error {
		msgs, err := fetch(bucket)
		if err != nil {
			return err
		}
		*out = append(*out, msgs...)
		return nil
	})
}

func (p *pebbleChannelLogStore) scanMessagesByIndex(bucket *PebbleBucket, lower, upper []byte) ([]wkdb.Message, error) {
	iter := bucket.DB().NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	defer iter.Close()
	msgs := make([]wkdb.Message, 0)
	for ok := iter.First(); ok; ok = iter.Next() {
		msg, err := p.loadMessageByRowKey(bucket.DB(), iter.Value())
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

func (p *pebbleChannelLogStore) scanAllMessages(bucket *PebbleBucket) ([]wkdb.Message, error) {
	iter := bucket.DB().NewIter(&pebble.IterOptions{
		LowerBound: []byte{channelLogKeyMessageRow},
		UpperBound: []byte{channelLogKeyMessageRow + 1},
	})
	defer iter.Close()
	msgs := make([]wkdb.Message, 0)
	for ok := iter.First(); ok; ok = iter.Next() {
		msg, err := decodeStoredChannelMessage(iter.Value())
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

func encodeStoredChannelMessage(channelID string, channelType uint8, msg wkdb.Message) ([]byte, []byte, error) {
	rowKey := encodeChannelMessageRowKey(channelID, channelType, uint64(msg.MessageSeq))
	data, err := msg.Marshal()
	if err != nil {
		return nil, nil, err
	}
	return rowKey, data, nil
}

func decodeStoredChannelMessage(data []byte) (wkdb.Message, error) {
	var msg wkdb.Message
	if err := msg.Unmarshal(data); err != nil {
		return wkdb.EmptyMessage, err
	}
	return msg, nil
}

func encodeLastSeqValue(seq uint64, lastTime uint64) []byte {
	value := make([]byte, 16)
	binary.BigEndian.PutUint64(value[:8], seq)
	binary.BigEndian.PutUint64(value[8:], lastTime)
	return value
}

func decodeLastSeqValue(value []byte) (uint64, uint64, error) {
	if len(value) == 0 {
		return 0, 0, nil
	}
	if len(value) < 16 {
		return 0, 0, fmt.Errorf("channel last seq value size mismatch: %d", len(value))
	}
	return binary.BigEndian.Uint64(value[:8]), binary.BigEndian.Uint64(value[8:16]), nil
}

func matchMessageSearchReq(msg wkdb.Message, req wkdb.MessageSearchReq) bool {
	if req.ChannelId != "" && msg.ChannelID != req.ChannelId {
		return false
	}
	if req.ChannelType != 0 && msg.ChannelType != req.ChannelType {
		return false
	}
	if req.FromUid != "" && msg.FromUID != req.FromUid {
		return false
	}
	if req.ClientMsgNo != "" && msg.ClientMsgNo != req.ClientMsgNo {
		return false
	}
	if req.MessageId > 0 && msg.MessageID != req.MessageId {
		return false
	}
	if len(req.Payload) > 0 {
		return bytes.Contains(msg.Payload, req.Payload)
	}
	return true
}

func encodeChannelMessageRowKey(channelID string, channelType uint8, seq uint64) []byte {
	key := channelMessageRowPrefix(channelID, channelType)
	return appendUint64(key, seq)
}

func parseChannelMessageRowKey(key []byte) (string, uint8, uint64, error) {
	if len(key) < 2 || key[0] != channelLogKeyMessageRow {
		return "", 0, 0, fmt.Errorf("invalid channel message row key")
	}
	channelType := key[1]
	channelID, off, err := readString(key, 2)
	if err != nil {
		return "", 0, 0, err
	}
	seq, _, err := readUint64(key, off)
	if err != nil {
		return "", 0, 0, err
	}
	return channelID, channelType, seq, nil
}

func channelMessageRowPrefix(channelID string, channelType uint8) []byte {
	key := []byte{channelLogKeyMessageRow, channelType}
	return appendString(key, channelID)
}

func channelMessageRowRange(channelID string, channelType uint8, startSeq, endSeq uint64) ([]byte, []byte) {
	prefix := channelMessageRowPrefix(channelID, channelType)
	lower := encodeChannelMessageRowKey(channelID, channelType, startSeq)
	if startSeq == 0 {
		lower = prefix
	}
	if endSeq == 0 || endSeq == math.MaxUint64 {
		return lower, v3key.PrefixUpperBound(prefix)
	}
	return lower, encodeChannelMessageRowKey(channelID, channelType, endSeq)
}

func encodeChannelLastSeqKey(channelID string, channelType uint8) []byte {
	key := []byte{channelLogKeyLastSeq, channelType}
	return appendString(key, channelID)
}

func encodeChannelMessageIDIndexKey(messageID uint64) []byte {
	return appendUint64([]byte{channelLogKeyMessageID}, messageID)
}

func encodeChannelMessageFromUIDIndexKey(fromUID, channelID string, channelType uint8, seq uint64) []byte {
	key := []byte{channelLogKeyFromUID}
	key = appendString(key, fromUID)
	key = append(key, channelType)
	key = appendString(key, channelID)
	return appendUint64(key, seq)
}

func channelMessageFromUIDIndexRange(fromUID string) ([]byte, []byte) {
	lower := []byte{channelLogKeyFromUID}
	lower = appendString(lower, fromUID)
	return lower, v3key.PrefixUpperBound(lower)
}

func channelMessageFromUIDIndexRangeForChannel(fromUID, channelID string, channelType uint8) ([]byte, []byte) {
	lower := []byte{channelLogKeyFromUID}
	lower = appendString(lower, fromUID)
	lower = append(lower, channelType)
	lower = appendString(lower, channelID)
	return lower, v3key.PrefixUpperBound(lower)
}

func encodeChannelMessageClientMsgNoIndexKey(clientMsgNo, channelID string, channelType uint8, seq uint64) []byte {
	key := []byte{channelLogKeyClientMsgNo}
	key = appendString(key, clientMsgNo)
	key = append(key, channelType)
	key = appendString(key, channelID)
	return appendUint64(key, seq)
}

func channelMessageClientMsgNoIndexRange(clientMsgNo string) ([]byte, []byte) {
	lower := []byte{channelLogKeyClientMsgNo}
	lower = appendString(lower, clientMsgNo)
	return lower, v3key.PrefixUpperBound(lower)
}

func channelMessageClientMsgNoIndexRangeForChannel(clientMsgNo, channelID string, channelType uint8) ([]byte, []byte) {
	lower := []byte{channelLogKeyClientMsgNo}
	lower = appendString(lower, clientMsgNo)
	lower = append(lower, channelType)
	lower = appendString(lower, channelID)
	return lower, v3key.PrefixUpperBound(lower)
}

func encodeChannelLeaderTermKey(shardNo string, term uint32) []byte {
	key := []byte{channelLogKeyLeaderTerm}
	key = appendString(key, shardNo)
	return appendUint32(key, term)
}

func parseChannelLeaderTermKey(key []byte) (uint32, error) {
	if len(key) == 0 || key[0] != channelLogKeyLeaderTerm {
		return 0, fmt.Errorf("invalid channel leader term key")
	}
	_, off, err := readString(key, 1)
	if err != nil {
		return 0, err
	}
	term, _, err := readUint32(key, off)
	return term, err
}

func channelLeaderTermRange(shardNo string) ([]byte, []byte) {
	lower := []byte{channelLogKeyLeaderTerm}
	lower = appendString(lower, shardNo)
	return lower, v3key.PrefixUpperBound(lower)
}

func appendUint32(dst []byte, v uint32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, v)
	return append(dst, buf...)
}

func appendUint64(dst []byte, v uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, v)
	return append(dst, buf...)
}

func appendString(dst []byte, v string) []byte {
	dst = appendUint32(dst, uint32(len(v)))
	return append(dst, v...)
}

func readUint32(src []byte, off int) (uint32, int, error) {
	if len(src) < off+4 {
		return 0, off, fmt.Errorf("read uint32: buffer too short")
	}
	return binary.BigEndian.Uint32(src[off : off+4]), off + 4, nil
}

func readUint64(src []byte, off int) (uint64, int, error) {
	if len(src) < off+8 {
		return 0, off, fmt.Errorf("read uint64: buffer too short")
	}
	return binary.BigEndian.Uint64(src[off : off+8]), off + 8, nil
}

func readString(src []byte, off int) (string, int, error) {
	size, next, err := readUint32(src, off)
	if err != nil {
		return "", off, err
	}
	end := next + int(size)
	if len(src) < end {
		return "", off, fmt.Errorf("read string: buffer too short")
	}
	return string(src[next:end]), end, nil
}

var stopIterating = fmt.Errorf("stop iterating")
