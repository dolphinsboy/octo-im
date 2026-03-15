package v3

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	"github.com/cockroachdb/pebble"
)

type pebbleMessageEventStore struct {
	slotID  uint32
	buckets *PebbleBucketManager
}

var _ MessageEventStore = (*pebbleMessageEventStore)(nil)

func (p *pebbleMessageEventStore) GetState(channelID string, channelType uint8, clientMsgNo, eventKey string) (*wkdb.MessageEventState, error) {
	clientMsgNo = strings.TrimSpace(clientMsgNo)
	if clientMsgNo == "" {
		return nil, nil
	}
	eventKey = normalizeMessageEventKey(eventKey)

	bucket, err := p.bucket()
	if err != nil {
		return nil, err
	}
	value, closer, err := bucket.DB().Get(v3key.EncodeMessageEventStateRowKey(p.slotID, channelID, channelType, clientMsgNo, eventKey))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	state, err := decodeMessageEventState(value)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (p *pebbleMessageEventStore) GetStates(channelID string, channelType uint8, clientMsgNo string) ([]wkdb.MessageEventState, error) {
	clientMsgNo = strings.TrimSpace(clientMsgNo)
	if clientMsgNo == "" {
		return nil, nil
	}
	bucket, err := p.bucket()
	if err != nil {
		return nil, err
	}
	lower, upper := v3key.MessageEventStateRange(p.slotID, channelID, channelType, clientMsgNo)
	iter := bucket.DB().NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	states := make([]wkdb.MessageEventState, 0)
	for iter.First(); iter.Valid(); iter.Next() {
		state, err := decodeMessageEventState(iter.Value())
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	return cloneMessageEventStates(states), nil
}

func (p *pebbleMessageEventStore) PutState(state wkdb.MessageEventState) error {
	if strings.TrimSpace(state.ChannelId) == "" {
		return fmt.Errorf("channel id is required")
	}
	if strings.TrimSpace(state.ClientMsgNo) == "" {
		return fmt.Errorf("client msg no is required")
	}
	state.ClientMsgNo = strings.TrimSpace(state.ClientMsgNo)
	state.EventKey = normalizeMessageEventKey(state.EventKey)
	value, err := encodeMessageEventState(state)
	if err != nil {
		return err
	}
	bucket, err := p.bucket()
	if err != nil {
		return err
	}
	return bucket.DB().Set(v3key.EncodeMessageEventStateRowKey(p.slotID, state.ChannelId, state.ChannelType, state.ClientMsgNo, state.EventKey), value, p.writeOptions())
}

func (p *pebbleMessageEventStore) DeleteStates(channelID string, channelType uint8, clientMsgNo string) error {
	clientMsgNo = strings.TrimSpace(clientMsgNo)
	if clientMsgNo == "" {
		return nil
	}
	bucket, err := p.bucket()
	if err != nil {
		return err
	}
	lower, upper := v3key.MessageEventStateRange(p.slotID, channelID, channelType, clientMsgNo)
	batch := bucket.DB().NewBatch()
	defer batch.Close()
	if err := batch.DeleteRange(lower, upper, p.writeOptions()); err != nil {
		return err
	}
	if err := batch.Delete(v3key.EncodeMessageEventSeqAuxKey(p.slotID, channelID, channelType, clientMsgNo), p.writeOptions()); err != nil && err != pebble.ErrNotFound {
		return err
	}
	return batch.Commit(p.writeOptions())
}

func (p *pebbleMessageEventStore) GetSeq(channelID string, channelType uint8, clientMsgNo string) (uint64, error) {
	clientMsgNo = strings.TrimSpace(clientMsgNo)
	if clientMsgNo == "" {
		return 0, nil
	}
	bucket, err := p.bucket()
	if err != nil {
		return 0, err
	}
	value, closer, err := bucket.DB().Get(v3key.EncodeMessageEventSeqAuxKey(p.slotID, channelID, channelType, clientMsgNo))
	if err != nil {
		if err == pebble.ErrNotFound {
			return 0, nil
		}
		return 0, err
	}
	defer closer.Close()
	if len(value) != 8 {
		return 0, fmt.Errorf("message event seq value size mismatch: %d", len(value))
	}
	return binary.BigEndian.Uint64(value), nil
}

func (p *pebbleMessageEventStore) SetSeq(channelID string, channelType uint8, clientMsgNo string, seq uint64) error {
	clientMsgNo = strings.TrimSpace(clientMsgNo)
	if clientMsgNo == "" {
		return fmt.Errorf("client msg no is required")
	}
	bucket, err := p.bucket()
	if err != nil {
		return err
	}
	value := make([]byte, 8)
	binary.BigEndian.PutUint64(value, seq)
	return bucket.DB().Set(v3key.EncodeMessageEventSeqAuxKey(p.slotID, channelID, channelType, clientMsgNo), value, p.writeOptions())
}

func (p *pebbleMessageEventStore) bucket() (*PebbleBucket, error) {
	return p.buckets.Bucket(p.slotID)
}

func (p *pebbleMessageEventStore) writeOptions() *pebble.WriteOptions {
	bucket, _ := p.bucket()
	if bucket == nil {
		return pebble.Sync
	}
	return bucket.SnapshotStore().writeOptions
}

func encodeMessageEventState(state wkdb.MessageEventState) ([]byte, error) {
	return json.Marshal(state)
}

func decodeMessageEventState(data []byte) (wkdb.MessageEventState, error) {
	var state wkdb.MessageEventState
	if err := json.Unmarshal(data, &state); err != nil {
		return wkdb.MessageEventState{}, err
	}
	state.ClientMsgNo = strings.TrimSpace(state.ClientMsgNo)
	state.EventKey = normalizeMessageEventKey(state.EventKey)
	state.SnapshotPayload = append([]byte(nil), state.SnapshotPayload...)
	return state, nil
}

func cloneMessageEventStates(states []wkdb.MessageEventState) []wkdb.MessageEventState {
	if len(states) == 0 {
		return nil
	}
	out := make([]wkdb.MessageEventState, 0, len(states))
	for _, state := range states {
		state.SnapshotPayload = append([]byte(nil), state.SnapshotPayload...)
		out = append(out, state)
	}
	return out
}

func normalizeMessageEventKey(eventKey string) string {
	eventKey = strings.TrimSpace(eventKey)
	if eventKey == "" {
		return wkdb.EventKeyDefault
	}
	return eventKey
}
