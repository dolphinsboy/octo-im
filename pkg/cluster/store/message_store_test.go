package store

import (
	"testing"

	"github.com/WuKongIM/WuKongIM/pkg/cluster/channel"
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkproto "github.com/WuKongIM/WuKongIMGoProto"
	"github.com/stretchr/testify/require"
)

var _ channel.ChannelLogStore = (*LegacyMessageStore)(nil)

type stubMessageStore struct {
	loadNextRangeMsgsFn        func(channelID string, channelType uint8, startMessageSeq, endMessageSeq uint64, limit int) ([]wkdb.Message, error)
	loadMsgFn                  func(channelID string, channelType uint8, seq uint64) (wkdb.Message, error)
	loadLastMsgsFn             func(channelID string, channelType uint8, limit int) ([]wkdb.Message, error)
	loadLastMsgsWithEndFn      func(channelID string, channelType uint8, endMessageSeq uint64, limit int) ([]wkdb.Message, error)
	loadPrevRangeMsgsFn        func(channelID string, channelType uint8, start, end uint64, limit int) ([]wkdb.Message, error)
	getChannelLastMessageSeqFn func(channelID string, channelType uint8) (uint64, uint64, error)
	searchMessagesFn           func(req wkdb.MessageSearchReq) ([]wkdb.Message, error)
	loadMsgByClientMsgNoFn     func(channelID string, channelType uint8, clientMsgNo string) (wkdb.Message, error)
	getUserLastMsgSeqFn        func(fromUID, channelID string, channelType uint8) (uint64, error)
	loadMsgsBatchFn            func(requests []wkdb.BatchMsgRequest) ([]wkdb.BatchMsgResponse, error)
	getUserLastMsgSeqBatchFn   func(fromUID string, channels []wkdb.Channel) (map[string]uint64, error)
	getNotifyQueueFn           func(count int) ([]wkdb.Message, error)
	appendNotifyQueueFn        func(messages []wkdb.Message) error
	removeNotifyQueueFn        func(messageIDs []int64) error
}

var _ MessageStore = (*stubMessageStore)(nil)

func (s *stubMessageStore) GetLastMsg(channelId string, channelType uint8) (wkdb.Message, error) {
	return wkdb.EmptyMessage, nil
}

func (s *stubMessageStore) AppendMessages(channelId string, channelType uint8, msgs []wkdb.Message) error {
	return nil
}

func (s *stubMessageStore) LoadNextRangeMsgsForSize(channelId string, channelType uint8, startMessageSeq, endMessageSeq, limitSize uint64) ([]wkdb.Message, error) {
	return nil, nil
}

func (s *stubMessageStore) TruncateLogTo(channelId string, channelType uint8, messageSeq uint64) error {
	return nil
}

func (s *stubMessageStore) GetChannelLastMessageSeq(channelID string, channelType uint8) (uint64, uint64, error) {
	if s.getChannelLastMessageSeqFn != nil {
		return s.getChannelLastMessageSeqFn(channelID, channelType)
	}
	return 0, 0, nil
}

func (s *stubMessageStore) SetLeaderTermStartIndex(shardNo string, term uint32, index uint64) error {
	return nil
}

func (s *stubMessageStore) LeaderTermStartIndex(shardNo string, term uint32) (uint64, error) {
	return 0, nil
}

func (s *stubMessageStore) LeaderLastTerm(shardNo string) (uint32, error) {
	return 0, nil
}

func (s *stubMessageStore) LeaderLastTermGreaterEqThan(shardNo string, term uint32) (uint32, error) {
	return 0, nil
}

func (s *stubMessageStore) DeleteLeaderTermStartIndexGreaterThanTerm(shardNo string, term uint32) error {
	return nil
}

func (s *stubMessageStore) LoadNextRangeMsgs(channelID string, channelType uint8, startMessageSeq, endMessageSeq uint64, limit int) ([]wkdb.Message, error) {
	if s.loadNextRangeMsgsFn != nil {
		return s.loadNextRangeMsgsFn(channelID, channelType, startMessageSeq, endMessageSeq, limit)
	}
	return nil, nil
}

func (s *stubMessageStore) LoadMsg(channelID string, channelType uint8, seq uint64) (wkdb.Message, error) {
	if s.loadMsgFn != nil {
		return s.loadMsgFn(channelID, channelType, seq)
	}
	return wkdb.EmptyMessage, nil
}

func (s *stubMessageStore) LoadLastMsgs(channelID string, channelType uint8, limit int) ([]wkdb.Message, error) {
	if s.loadLastMsgsFn != nil {
		return s.loadLastMsgsFn(channelID, channelType, limit)
	}
	return nil, nil
}

func (s *stubMessageStore) LoadLastMsgsWithEnd(channelID string, channelType uint8, endMessageSeq uint64, limit int) ([]wkdb.Message, error) {
	if s.loadLastMsgsWithEndFn != nil {
		return s.loadLastMsgsWithEndFn(channelID, channelType, endMessageSeq, limit)
	}
	return nil, nil
}

func (s *stubMessageStore) LoadPrevRangeMsgs(channelID string, channelType uint8, start, end uint64, limit int) ([]wkdb.Message, error) {
	if s.loadPrevRangeMsgsFn != nil {
		return s.loadPrevRangeMsgsFn(channelID, channelType, start, end, limit)
	}
	return nil, nil
}

func (s *stubMessageStore) GetMessagesOfNotifyQueue(count int) ([]wkdb.Message, error) {
	if s.getNotifyQueueFn != nil {
		return s.getNotifyQueueFn(count)
	}
	return nil, nil
}

func (s *stubMessageStore) AppendMessageOfNotifyQueue(messages []wkdb.Message) error {
	if s.appendNotifyQueueFn != nil {
		return s.appendNotifyQueueFn(messages)
	}
	return nil
}

func (s *stubMessageStore) RemoveMessagesOfNotifyQueue(messageIDs []int64) error {
	if s.removeNotifyQueueFn != nil {
		return s.removeNotifyQueueFn(messageIDs)
	}
	return nil
}

func (s *stubMessageStore) RemoveMessagesOfNotifyQueueCount(count int) error {
	return nil
}

func (s *stubMessageStore) SearchMessages(req wkdb.MessageSearchReq) ([]wkdb.Message, error) {
	if s.searchMessagesFn != nil {
		return s.searchMessagesFn(req)
	}
	return nil, nil
}

func (s *stubMessageStore) LoadMsgByClientMsgNo(channelID string, channelType uint8, clientMsgNo string) (wkdb.Message, error) {
	if s.loadMsgByClientMsgNoFn != nil {
		return s.loadMsgByClientMsgNoFn(channelID, channelType, clientMsgNo)
	}
	return wkdb.EmptyMessage, nil
}

func (s *stubMessageStore) GetUserLastMsgSeq(fromUID string, channelID string, channelType uint8) (uint64, error) {
	if s.getUserLastMsgSeqFn != nil {
		return s.getUserLastMsgSeqFn(fromUID, channelID, channelType)
	}
	return 0, nil
}

func (s *stubMessageStore) LoadMsgsBatch(requests []wkdb.BatchMsgRequest) ([]wkdb.BatchMsgResponse, error) {
	if s.loadMsgsBatchFn != nil {
		return s.loadMsgsBatchFn(requests)
	}
	return nil, nil
}

func (s *stubMessageStore) GetUserLastMsgSeqBatch(fromUID string, channels []wkdb.Channel) (map[string]uint64, error) {
	if s.getUserLastMsgSeqBatchFn != nil {
		return s.getUserLastMsgSeqBatchFn(fromUID, channels)
	}
	return nil, nil
}

func TestStoreMessageQueriesUseDedicatedMessageStore(t *testing.T) {
	searchReq := wkdb.MessageSearchReq{Limit: 10}
	batchRequests := []wkdb.BatchMsgRequest{{ChannelId: "channel-1", ChannelType: 2, MsgSeq: 3, Limit: 5}}
	channels := []wkdb.Channel{{ChannelId: "channel-1", ChannelType: 2}}
	rangeMsgs := []wkdb.Message{{}}
	rangeMsgs[0].MessageSeq = 4
	lastMsgs := []wkdb.Message{{}}
	lastMsgs[0].MessageSeq = 9
	batchResp := []wkdb.BatchMsgResponse{{ChannelId: "channel-1", Messages: []wkdb.Message{{}}}}
	batchResp[0].Messages[0].MessageSeq = 7
	msgBySeq := wkdb.Message{}
	msgBySeq.MessageSeq = 5
	msgByClient := wkdb.Message{}
	msgByClient.MessageSeq = 6
	notifyQueueMessages := []wkdb.Message{{}}
	notifyQueueMessages[0].MessageSeq = 13

	store := &Store{
		messageStore: &stubMessageStore{
			loadNextRangeMsgsFn: func(channelID string, channelType uint8, startMessageSeq, endMessageSeq uint64, limit int) ([]wkdb.Message, error) {
				require.Equal(t, "channel-1", channelID)
				require.Equal(t, uint8(2), channelType)
				require.Equal(t, uint64(3), startMessageSeq)
				require.Equal(t, uint64(8), endMessageSeq)
				require.Equal(t, 5, limit)
				return rangeMsgs, nil
			},
			loadMsgFn: func(channelID string, channelType uint8, seq uint64) (wkdb.Message, error) {
				require.Equal(t, "channel-1", channelID)
				require.Equal(t, uint8(2), channelType)
				require.Equal(t, uint64(5), seq)
				return msgBySeq, nil
			},
			loadLastMsgsFn: func(channelID string, channelType uint8, limit int) ([]wkdb.Message, error) {
				require.Equal(t, "channel-1", channelID)
				require.Equal(t, uint8(2), channelType)
				require.Equal(t, 3, limit)
				return lastMsgs, nil
			},
			loadLastMsgsWithEndFn: func(channelID string, channelType uint8, endMessageSeq uint64, limit int) ([]wkdb.Message, error) {
				require.Equal(t, "channel-1", channelID)
				require.Equal(t, uint8(2), channelType)
				require.Equal(t, uint64(11), endMessageSeq)
				require.Equal(t, 2, limit)
				return lastMsgs, nil
			},
			loadPrevRangeMsgsFn: func(channelID string, channelType uint8, start, end uint64, limit int) ([]wkdb.Message, error) {
				require.Equal(t, "channel-1", channelID)
				require.Equal(t, uint8(2), channelType)
				require.Equal(t, uint64(20), start)
				require.Equal(t, uint64(10), end)
				require.Equal(t, 4, limit)
				return rangeMsgs, nil
			},
			getChannelLastMessageSeqFn: func(channelID string, channelType uint8) (uint64, uint64, error) {
				require.Equal(t, "channel-1", channelID)
				require.Equal(t, uint8(2), channelType)
				return 12, 34, nil
			},
			searchMessagesFn: func(req wkdb.MessageSearchReq) ([]wkdb.Message, error) {
				require.Equal(t, searchReq, req)
				return rangeMsgs, nil
			},
			loadMsgByClientMsgNoFn: func(channelID string, channelType uint8, clientMsgNo string) (wkdb.Message, error) {
				require.Equal(t, "channel-1", channelID)
				require.Equal(t, uint8(2), channelType)
				require.Equal(t, "client-1", clientMsgNo)
				return msgByClient, nil
			},
			getUserLastMsgSeqFn: func(fromUID, channelID string, channelType uint8) (uint64, error) {
				require.Equal(t, "user-1", fromUID)
				require.Equal(t, "channel-1", channelID)
				require.Equal(t, uint8(2), channelType)
				return 22, nil
			},
			loadMsgsBatchFn: func(requests []wkdb.BatchMsgRequest) ([]wkdb.BatchMsgResponse, error) {
				require.Equal(t, batchRequests, requests)
				return batchResp, nil
			},
			getUserLastMsgSeqBatchFn: func(fromUID string, gotChannels []wkdb.Channel) (map[string]uint64, error) {
				require.Equal(t, "user-1", fromUID)
				require.Equal(t, channels, gotChannels)
				return map[string]uint64{"channel-1-2": 33}, nil
			},
			getNotifyQueueFn: func(count int) ([]wkdb.Message, error) {
				require.Equal(t, 2, count)
				return notifyQueueMessages, nil
			},
			appendNotifyQueueFn: func(messages []wkdb.Message) error {
				require.Equal(t, notifyQueueMessages, messages)
				return nil
			},
			removeNotifyQueueFn: func(messageIDs []int64) error {
				require.Equal(t, []int64{101, 102}, messageIDs)
				return nil
			},
		},
	}

	msgs, err := store.LoadNextRangeMsgs("channel-1", 2, 3, 8, 5)
	require.NoError(t, err)
	require.Equal(t, rangeMsgs, msgs)

	msg, err := store.LoadMsg("channel-1", 2, 5)
	require.NoError(t, err)
	require.Equal(t, msgBySeq, msg)

	msgs, err = store.LoadLastMsgs("channel-1", 2, 3)
	require.NoError(t, err)
	require.Equal(t, lastMsgs, msgs)

	msgs, err = store.LoadLastMsgsWithEnd("channel-1", 2, 11, 2)
	require.NoError(t, err)
	require.Equal(t, lastMsgs, msgs)

	msgs, err = store.LoadPrevRangeMsgs("channel-1", 2, 20, 10, 4)
	require.NoError(t, err)
	require.Equal(t, rangeMsgs, msgs)

	seq, err := store.GetLastMsgSeq("channel-1", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(12), seq)

	seq, appendTime, err := store.GetChannelLastMessageSeqAndTime("channel-1", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(12), seq)
	require.Equal(t, uint64(34), appendTime)

	seq, err = store.GetChannelLastMessageSeq("channel-1", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(12), seq)

	msgs, err = store.SearchMessages(searchReq)
	require.NoError(t, err)
	require.Equal(t, rangeMsgs, msgs)

	msg, err = store.LoadMsgByClientMsgNo("channel-1", 2, "client-1")
	require.NoError(t, err)
	require.Equal(t, msgByClient, msg)

	seq, err = store.GetUserLastMsgSeq("user-1", "channel-1", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(22), seq)

	responses, err := store.LoadMsgsBatch(batchRequests)
	require.NoError(t, err)
	require.Equal(t, batchResp, responses)

	userSeqMap, err := store.GetUserLastMsgSeqBatch("user-1", channels)
	require.NoError(t, err)
	require.Equal(t, map[string]uint64{"channel-1-2": 33}, userSeqMap)

	notifyMessages, err := store.GetMessagesOfNotifyQueue(2)
	require.NoError(t, err)
	require.Equal(t, notifyQueueMessages, notifyMessages)

	require.NoError(t, store.AppendMessageOfNotifyQueue(notifyQueueMessages))
	require.NoError(t, store.RemoveMessagesOfNotifyQueue([]int64{101, 102}))
}

func TestLegacyMessageStoreUsesHybridNotifyQueueOverrides(t *testing.T) {
	hybrid, legacy := newHybridMetaLocalTestDB(t)
	messageStore := NewLegacyMessageStore(hybrid)

	require.NoError(t, messageStore.AppendMessageOfNotifyQueue([]wkdb.Message{
		{
			RecvPacket: wkproto.RecvPacket{
				MessageID:   1,
				ChannelID:   "channel-1",
				ChannelType: 1,
				FromUID:     "user-1",
				ClientMsgNo: "client-1",
				Timestamp:   100,
				Payload:     []byte("hello"),
			},
		},
	}))

	messages, err := messageStore.GetMessagesOfNotifyQueue(10)
	require.NoError(t, err)
	require.Len(t, messages, 1)

	legacyMessages, err := legacy.GetMessagesOfNotifyQueue(10)
	require.NoError(t, err)
	require.Empty(t, legacyMessages)
}
