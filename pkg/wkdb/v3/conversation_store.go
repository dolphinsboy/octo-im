package v3

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	wkproto "github.com/WuKongIM/WuKongIMGoProto"
	"github.com/cockroachdb/pebble"
)

type pebbleConversationStore struct {
	slotID  uint32
	buckets *PebbleBucketManager
	now     func() time.Time
}

var _ ConversationStore = (*pebbleConversationStore)(nil)

func (p *pebbleConversationStore) Get(uid, channelID string, channelType uint8) (wkdb.Conversation, error) {
	rowKey := v3key.EncodeConversationRowKey(p.slotID, uid, channelID, channelType)
	return p.getByRowKey(rowKey)
}

func (p *pebbleConversationStore) ListByUID(uid string) ([]wkdb.Conversation, error) {
	bucket, err := p.bucket()
	if err != nil {
		return nil, err
	}
	lower, upper := v3key.ConversationUIDRange(p.slotID, uid)
	iter := bucket.DB().NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	conversations := make([]wkdb.Conversation, 0)
	for iter.First(); iter.Valid(); iter.Next() {
		conversation, err := decodeConversationValue(iter.Value())
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, conversation)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	sortConversations(conversations)
	return conversations, nil
}

func (p *pebbleConversationStore) Put(uid string, cs []wkdb.Conversation) error {
	if len(cs) == 0 {
		return nil
	}
	bucket, err := p.bucket()
	if err != nil {
		return err
	}
	batch := bucket.DB().NewBatch()
	defer batch.Close()

	now := p.currentTime()
	for _, conversation := range cs {
		normalized, oldConversation, existed, err := p.prepareConversation(uid, conversation, now)
		if err != nil {
			return err
		}
		if existed {
			if err := p.deleteConversationIndex(batch, oldConversation); err != nil {
				return err
			}
		}
		if err := p.writeConversation(batch, normalized); err != nil {
			return err
		}
	}
	return batch.Commit(p.writeOptions())
}

func (p *pebbleConversationStore) Delete(uid, channelID string, channelType uint8) error {
	bucket, err := p.bucket()
	if err != nil {
		return err
	}

	conversation, err := p.Get(uid, channelID, channelType)
	if err != nil {
		if err == wkdb.ErrNotFound {
			return nil
		}
		return err
	}

	batch := bucket.DB().NewBatch()
	defer batch.Close()
	if err := p.deleteConversation(batch, conversation); err != nil {
		return err
	}
	return batch.Commit(p.writeOptions())
}

func (p *pebbleConversationStore) DeleteBatch(uid string, channels []wkdb.Channel) error {
	if len(channels) == 0 {
		return nil
	}
	bucket, err := p.bucket()
	if err != nil {
		return err
	}

	batch := bucket.DB().NewBatch()
	defer batch.Close()
	for _, channel := range channels {
		conversation, err := p.Get(uid, channel.ChannelId, channel.ChannelType)
		if err != nil {
			if err == wkdb.ErrNotFound {
				continue
			}
			return err
		}
		if err := p.deleteConversation(batch, conversation); err != nil {
			return err
		}
	}
	return batch.Commit(p.writeOptions())
}

func (p *pebbleConversationStore) UpdateIfSeqGreater(uid, channelID string, channelType uint8, seq uint64) error {
	conversation, err := p.Get(uid, channelID, channelType)
	if err != nil {
		if err == wkdb.ErrNotFound {
			return nil
		}
		return err
	}
	if conversation.ReadToMsgSeq >= seq {
		return nil
	}
	conversation.ReadToMsgSeq = seq
	now := p.currentTime()
	conversation.UpdatedAt = &now
	return p.Put(uid, []wkdb.Conversation{conversation})
}

func (p *pebbleConversationStore) UpdateDeletedAt(uid, channelID string, channelType uint8, seq uint64) error {
	conversation, err := p.Get(uid, channelID, channelType)
	if err != nil {
		if err == wkdb.ErrNotFound {
			return nil
		}
		return err
	}
	conversation.DeletedAtMsgSeq = seq
	now := p.currentTime()
	conversation.UpdatedAt = &now
	return p.Put(uid, []wkdb.Conversation{conversation})
}

func (p *pebbleConversationStore) Search(req wkdb.ConversationSearchReq) ([]wkdb.Conversation, error) {
	var (
		conversations []wkdb.Conversation
		err           error
	)
	if req.Uid != "" {
		conversations, err = p.ListByUID(req.Uid)
	} else {
		conversations, err = p.listAll()
	}
	if err != nil {
		return nil, err
	}
	return paginateConversations(conversations, req.CurrentPage, req.Limit), nil
}

func (p *pebbleConversationStore) prepareConversation(uid string, conversation wkdb.Conversation, now time.Time) (wkdb.Conversation, wkdb.Conversation, bool, error) {
	if uid == "" {
		return wkdb.Conversation{}, wkdb.Conversation{}, false, fmt.Errorf("conversation uid is required")
	}
	if conversation.ChannelId == "" {
		return wkdb.Conversation{}, wkdb.Conversation{}, false, fmt.Errorf("conversation channel id is required")
	}
	if conversation.Uid == "" {
		conversation.Uid = uid
	}
	if conversation.Uid != uid {
		return wkdb.Conversation{}, wkdb.Conversation{}, false, fmt.Errorf("conversation uid mismatch: got %s want %s", conversation.Uid, uid)
	}
	oldConversation, err := p.Get(uid, conversation.ChannelId, conversation.ChannelType)
	if err != nil && err != wkdb.ErrNotFound {
		return wkdb.Conversation{}, wkdb.Conversation{}, false, err
	}
	existed := err == nil
	if conversation.CreatedAt == nil {
		if existed && oldConversation.CreatedAt != nil {
			conversation.CreatedAt = cloneTimePtr(oldConversation.CreatedAt)
		} else {
			conversation.CreatedAt = cloneTimePtr(&now)
		}
	}
	if conversation.UpdatedAt == nil {
		conversation.UpdatedAt = cloneTimePtr(&now)
	}
	if existed && conversation.Id == 0 {
		conversation.Id = oldConversation.Id
	}
	return conversation, oldConversation, existed, nil
}

func (p *pebbleConversationStore) writeConversation(batch *pebble.Batch, conversation wkdb.Conversation) error {
	value, err := encodeConversationValue(conversation)
	if err != nil {
		return err
	}
	rowKey := v3key.EncodeConversationRowKey(p.slotID, conversation.Uid, conversation.ChannelId, conversation.ChannelType)
	if err := batch.Set(rowKey, value, p.writeOptions()); err != nil {
		return err
	}
	indexKey := v3key.EncodeConversationUpdatedAtSecondIndexKey(p.slotID, conversation.Uid, uint64(conversation.UpdatedAt.UnixNano()), conversation.ChannelId, conversation.ChannelType)
	return batch.Set(indexKey, rowKey, p.writeOptions())
}

func (p *pebbleConversationStore) deleteConversation(batch *pebble.Batch, conversation wkdb.Conversation) error {
	rowKey := v3key.EncodeConversationRowKey(p.slotID, conversation.Uid, conversation.ChannelId, conversation.ChannelType)
	if err := batch.Delete(rowKey, p.writeOptions()); err != nil {
		return err
	}
	return p.deleteConversationIndex(batch, conversation)
}

func (p *pebbleConversationStore) deleteConversationIndex(batch *pebble.Batch, conversation wkdb.Conversation) error {
	if conversation.UpdatedAt == nil {
		return nil
	}
	indexKey := v3key.EncodeConversationUpdatedAtSecondIndexKey(p.slotID, conversation.Uid, uint64(conversation.UpdatedAt.UnixNano()), conversation.ChannelId, conversation.ChannelType)
	return batch.Delete(indexKey, p.writeOptions())
}

func (p *pebbleConversationStore) getByRowKey(rowKey []byte) (wkdb.Conversation, error) {
	bucket, err := p.bucket()
	if err != nil {
		return wkdb.Conversation{}, err
	}
	value, closer, err := bucket.DB().Get(rowKey)
	if err != nil {
		if err == pebble.ErrNotFound {
			return wkdb.Conversation{}, wkdb.ErrNotFound
		}
		return wkdb.Conversation{}, err
	}
	defer closer.Close()
	return decodeConversationValue(value)
}

func (p *pebbleConversationStore) listAll() ([]wkdb.Conversation, error) {
	bucket, err := p.bucket()
	if err != nil {
		return nil, err
	}
	lower, upper := v3key.SlotTableRange(p.slotID, v3key.ScopeSlotPrimary, v3key.TableConversation)
	iter := bucket.DB().NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	conversations := make([]wkdb.Conversation, 0)
	for iter.First(); iter.Valid(); iter.Next() {
		conversation, err := decodeConversationValue(iter.Value())
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, conversation)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	sortConversations(conversations)
	return conversations, nil
}

func (p *pebbleConversationStore) bucket() (*PebbleBucket, error) {
	return p.buckets.Bucket(p.slotID)
}

func (p *pebbleConversationStore) currentTime() time.Time {
	if p.now != nil {
		return p.now()
	}
	return time.Now()
}

func (p *pebbleConversationStore) writeOptions() *pebble.WriteOptions {
	bucket, _ := p.bucket()
	if bucket == nil {
		return pebble.Sync
	}
	return bucket.SnapshotStore().writeOptions
}

func encodeConversationValue(conversation wkdb.Conversation) ([]byte, error) {
	if conversation.Uid == "" {
		return nil, fmt.Errorf("conversation uid is required")
	}
	if conversation.ChannelId == "" {
		return nil, fmt.Errorf("conversation channel id is required")
	}
	if conversation.CreatedAt == nil || conversation.UpdatedAt == nil {
		return nil, fmt.Errorf("conversation timestamps are required")
	}
	enc := wkproto.NewEncoder()
	defer enc.End()
	enc.WriteUint64(conversation.Id)
	enc.WriteString(conversation.Uid)
	enc.WriteUint8(uint8(conversation.Type))
	enc.WriteString(conversation.ChannelId)
	enc.WriteUint8(conversation.ChannelType)
	enc.WriteUint32(conversation.UnreadCount)
	enc.WriteUint64(conversation.ReadToMsgSeq)
	enc.WriteUint64(conversation.DeletedAtMsgSeq)
	enc.WriteUint64(uint64(conversation.CreatedAt.UnixNano()))
	enc.WriteUint64(uint64(conversation.UpdatedAt.UnixNano()))
	return enc.Bytes(), nil
}

func decodeConversationValue(data []byte) (wkdb.Conversation, error) {
	dec := wkproto.NewDecoder(data)
	var (
		conversation wkdb.Conversation
		err          error
	)
	if conversation.Id, err = dec.Uint64(); err != nil {
		return wkdb.Conversation{}, err
	}
	if conversation.Uid, err = dec.String(); err != nil {
		return wkdb.Conversation{}, err
	}
	tp, err := dec.Uint8()
	if err != nil {
		return wkdb.Conversation{}, err
	}
	conversation.Type = wkdb.ConversationType(tp)
	if conversation.ChannelId, err = dec.String(); err != nil {
		return wkdb.Conversation{}, err
	}
	if conversation.ChannelType, err = dec.Uint8(); err != nil {
		return wkdb.Conversation{}, err
	}
	if conversation.UnreadCount, err = dec.Uint32(); err != nil {
		return wkdb.Conversation{}, err
	}
	if conversation.ReadToMsgSeq, err = dec.Uint64(); err != nil {
		return wkdb.Conversation{}, err
	}
	if conversation.DeletedAtMsgSeq, err = dec.Uint64(); err != nil {
		return wkdb.Conversation{}, err
	}
	createdAt, err := dec.Uint64()
	if err != nil {
		return wkdb.Conversation{}, err
	}
	updatedAt, err := dec.Uint64()
	if err != nil {
		return wkdb.Conversation{}, err
	}
	conversation.CreatedAt = unixNanoPtr(createdAt)
	conversation.UpdatedAt = unixNanoPtr(updatedAt)
	return conversation, nil
}

func paginateConversations(conversations []wkdb.Conversation, currentPage, limit int) []wkdb.Conversation {
	if len(conversations) == 0 {
		return nil
	}
	if limit <= 0 {
		out := make([]wkdb.Conversation, 0, len(conversations))
		for _, conversation := range conversations {
			out = append(out, cloneConversation(conversation))
		}
		return out
	}
	if currentPage <= 0 {
		currentPage = 1
	}
	start := (currentPage - 1) * limit
	if start >= len(conversations) {
		return nil
	}
	end := start + limit
	if end > len(conversations) {
		end = len(conversations)
	}
	out := make([]wkdb.Conversation, 0, end-start)
	for _, conversation := range conversations[start:end] {
		out = append(out, cloneConversation(conversation))
	}
	return out
}

func sortConversations(conversations []wkdb.Conversation) {
	sort.SliceStable(conversations, func(i, j int) bool {
		left := conversationSortTime(conversations[i])
		right := conversationSortTime(conversations[j])
		if left == right {
			return bytes.Compare(
				v3key.EncodeConversationRowKey(0, conversations[i].Uid, conversations[i].ChannelId, conversations[i].ChannelType),
				v3key.EncodeConversationRowKey(0, conversations[j].Uid, conversations[j].ChannelId, conversations[j].ChannelType),
			) < 0
		}
		return left > right
	})
}

func conversationSortTime(conversation wkdb.Conversation) int64 {
	if conversation.UpdatedAt != nil {
		return conversation.UpdatedAt.UnixNano()
	}
	if conversation.CreatedAt != nil {
		return conversation.CreatedAt.UnixNano()
	}
	return 0
}

func cloneConversation(conversation wkdb.Conversation) wkdb.Conversation {
	conversation.CreatedAt = cloneTimePtr(conversation.CreatedAt)
	conversation.UpdatedAt = cloneTimePtr(conversation.UpdatedAt)
	return conversation
}

func cloneTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	cp := *t
	return &cp
}

func unixNanoPtr(v uint64) *time.Time {
	if v == 0 {
		return nil
	}
	t := time.Unix(0, int64(v))
	return &t
}
