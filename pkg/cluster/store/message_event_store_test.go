package store

import (
	"context"
	"testing"

	"github.com/WuKongIM/WuKongIM/pkg/raft/types"
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/stretchr/testify/require"
)

type stubSlot struct {
	lastKey  string
	lastData []byte
}

func (s *stubSlot) SlotLeaderId(slotId uint32) uint64 {
	return 0
}

func (s *stubSlot) GetSlotId(v string) uint32 {
	s.lastKey = v
	return 7
}

func (s *stubSlot) Propose(slotId uint32, data []byte) (*types.ProposeResp, error) {
	s.lastData = append([]byte(nil), data...)
	return &types.ProposeResp{Index: 1}, nil
}

func (s *stubSlot) ProposeUntilApplied(slotId uint32, data []byte) (*types.ProposeResp, error) {
	s.lastData = append([]byte(nil), data...)
	return &types.ProposeResp{Index: 1}, nil
}

func (s *stubSlot) ProposeUntilAppliedTimeout(ctx context.Context, slotId uint32, data []byte) (*types.ProposeResp, error) {
	s.lastData = append([]byte(nil), data...)
	return &types.ProposeResp{Index: 1}, nil
}

type stubMessageEventStore struct {
	appendFn         func(event *wkdb.MessageEvent) (*wkdb.MessageEvent, *wkdb.MessageEventState, error)
	getByEventIDFn   func(channelId string, channelType uint8, clientMsgNo, eventID string) (*wkdb.MessageEvent, error)
	listFn           func(channelId string, channelType uint8, clientMsgNo string, fromMsgEventSeq uint64, eventKey string, limit int) ([]wkdb.MessageEvent, error)
	getStatesFn      func(channelId string, channelType uint8, clientMsgNo string) ([]wkdb.MessageEventState, error)
	getStatesBatchFn func(channelId string, channelType uint8, clientMsgNos []string) (map[string][]wkdb.MessageEventState, error)
	getStateFn       func(channelId string, channelType uint8, clientMsgNo, eventKey string) (*wkdb.MessageEventState, error)
}

var _ MessageEventStore = (*stubMessageEventStore)(nil)

func (s *stubMessageEventStore) AppendMessageEventWithState(event *wkdb.MessageEvent) (*wkdb.MessageEvent, *wkdb.MessageEventState, error) {
	if s.appendFn != nil {
		return s.appendFn(event)
	}
	return nil, nil, nil
}

func (s *stubMessageEventStore) GetMessageEventByEventID(channelId string, channelType uint8, clientMsgNo, eventID string) (*wkdb.MessageEvent, error) {
	if s.getByEventIDFn != nil {
		return s.getByEventIDFn(channelId, channelType, clientMsgNo, eventID)
	}
	return nil, nil
}

func (s *stubMessageEventStore) ListMessageEvents(channelId string, channelType uint8, clientMsgNo string, fromMsgEventSeq uint64, eventKey string, limit int) ([]wkdb.MessageEvent, error) {
	if s.listFn != nil {
		return s.listFn(channelId, channelType, clientMsgNo, fromMsgEventSeq, eventKey, limit)
	}
	return nil, nil
}

func (s *stubMessageEventStore) GetMessageEventStates(channelId string, channelType uint8, clientMsgNo string) ([]wkdb.MessageEventState, error) {
	if s.getStatesFn != nil {
		return s.getStatesFn(channelId, channelType, clientMsgNo)
	}
	return nil, nil
}

func (s *stubMessageEventStore) GetMessageEventStatesBatch(channelId string, channelType uint8, clientMsgNos []string) (map[string][]wkdb.MessageEventState, error) {
	if s.getStatesBatchFn != nil {
		return s.getStatesBatchFn(channelId, channelType, clientMsgNos)
	}
	return nil, nil
}

func (s *stubMessageEventStore) GetMessageEventState(channelId string, channelType uint8, clientMsgNo, eventKey string) (*wkdb.MessageEventState, error) {
	if s.getStateFn != nil {
		return s.getStateFn(channelId, channelType, clientMsgNo, eventKey)
	}
	return nil, nil
}

func TestStoreMessageEventQueriesUseDedicatedEventStore(t *testing.T) {
	event := wkdb.MessageEvent{ClientMsgNo: "client-1", EventID: "evt-1"}
	state := &wkdb.MessageEventState{ClientMsgNo: "client-1", EventKey: wkdb.EventKeyDefault, LastMsgEventSeq: 3}
	list := []wkdb.MessageEvent{{ClientMsgNo: "client-1", MsgEventSeq: 4}}
	states := []wkdb.MessageEventState{{ClientMsgNo: "client-1", EventKey: wkdb.EventKeyDefault}}
	batch := map[string][]wkdb.MessageEventState{"client-1": states}
	slot := &stubSlot{}

	store := &Store{
		opts: &Options{Slot: slot},
		eventStore: &stubMessageEventStore{
			getStateFn: func(channelId string, channelType uint8, clientMsgNo, eventKey string) (*wkdb.MessageEventState, error) {
				require.Equal(t, "channel-1", channelId)
				require.Equal(t, uint8(2), channelType)
				require.Equal(t, "client-1", clientMsgNo)
				require.Equal(t, wkdb.EventKeyDefault, eventKey)
				return state, nil
			},
			getByEventIDFn: func(channelId string, channelType uint8, clientMsgNo, eventID string) (*wkdb.MessageEvent, error) {
				require.Equal(t, "evt-1", eventID)
				return &event, nil
			},
			listFn: func(channelId string, channelType uint8, clientMsgNo string, fromMsgEventSeq uint64, eventKey string, limit int) ([]wkdb.MessageEvent, error) {
				require.Equal(t, uint64(2), fromMsgEventSeq)
				require.Equal(t, "main", eventKey)
				require.Equal(t, 10, limit)
				return list, nil
			},
			getStatesFn: func(channelId string, channelType uint8, clientMsgNo string) ([]wkdb.MessageEventState, error) {
				return states, nil
			},
			getStatesBatchFn: func(channelId string, channelType uint8, clientMsgNos []string) (map[string][]wkdb.MessageEventState, error) {
				require.Equal(t, []string{"client-1"}, clientMsgNos)
				return batch, nil
			},
		},
	}

	stored, gotState, err := store.AppendMessageEventWithState("channel-1", 2, &wkdb.MessageEvent{ClientMsgNo: "client-1", EventID: "evt-1"})
	require.NoError(t, err)
	require.Equal(t, "channel-1", slot.lastKey)
	require.NotEmpty(t, slot.lastData)
	require.Equal(t, uint64(3), stored.MsgEventSeq)
	require.Equal(t, wkdb.EventKeyDefault, stored.EventKey)
	require.Equal(t, state, gotState)

	gotEvent, err := store.GetMessageEventByEventID("channel-1", 2, "client-1", "evt-1")
	require.NoError(t, err)
	require.Equal(t, &event, gotEvent)

	gotList, err := store.ListMessageEvents("channel-1", 2, "client-1", 2, "main", 10)
	require.NoError(t, err)
	require.Equal(t, list, gotList)

	gotStates, err := store.GetMessageEventStates("channel-1", 2, "client-1")
	require.NoError(t, err)
	require.Equal(t, states, gotStates)

	gotBatch, err := store.GetMessageEventStatesBatch("channel-1", 2, []string{"client-1"})
	require.NoError(t, err)
	require.Equal(t, batch, gotBatch)

	gotSingleState, err := store.GetMessageEventState("channel-1", 2, "client-1", wkdb.EventKeyDefault)
	require.NoError(t, err)
	require.Equal(t, state, gotSingleState)
}
