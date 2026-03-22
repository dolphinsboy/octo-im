package v3

import (
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/WuKongIM/WuKongIM/pkg/wkutil"
	wkproto "github.com/WuKongIM/WuKongIMGoProto"
	"github.com/stretchr/testify/require"
)

func TestPebbleChannelLogStoreMessageLifecycle(t *testing.T) {
	db := newTestPebbleDBWithNow(t, time.Now)
	defer db.Close()

	store := db.ChannelLogs()
	channelID := "channel-1"
	channelType := uint8(2)

	messages := []wkdb.Message{
		{
			RecvPacket: wkproto.RecvPacket{
				MessageID:   101,
				MessageSeq:  1,
				ChannelID:   channelID,
				ChannelType: channelType,
				FromUID:     "user-1",
				ClientMsgNo: "client-1",
				Timestamp:   100,
				Payload:     []byte("hello"),
			},
			Term: 1,
		},
		{
			RecvPacket: wkproto.RecvPacket{
				MessageID:   102,
				MessageSeq:  2,
				ChannelID:   channelID,
				ChannelType: channelType,
				FromUID:     "user-1",
				ClientMsgNo: "client-2",
				Timestamp:   200,
				Payload:     []byte("world"),
			},
			Term: 1,
		},
		{
			RecvPacket: wkproto.RecvPacket{
				MessageID:   103,
				MessageSeq:  3,
				ChannelID:   channelID,
				ChannelType: channelType,
				FromUID:     "user-2",
				ClientMsgNo: "client-3",
				Timestamp:   300,
				Payload:     []byte("bye"),
			},
			Term: 2,
		},
	}

	require.NoError(t, store.AppendMessages(channelID, channelType, messages))

	lastSeq, lastTime, err := store.GetChannelLastMessageSeq(channelID, channelType)
	require.NoError(t, err)
	require.Equal(t, uint64(3), lastSeq)
	require.NotZero(t, lastTime)

	msg, err := store.GetMessage(102)
	require.NoError(t, err)
	require.Equal(t, uint32(2), msg.MessageSeq)

	msg, err = store.LoadMsg(channelID, channelType, 2)
	require.NoError(t, err)
	require.Equal(t, int64(102), msg.MessageID)

	nextMsgs, err := store.LoadNextRangeMsgs(channelID, channelType, 1, 0, 2)
	require.NoError(t, err)
	require.Len(t, nextMsgs, 2)
	require.Equal(t, uint32(1), nextMsgs[0].MessageSeq)
	require.Equal(t, uint32(2), nextMsgs[1].MessageSeq)

	prevMsgs, err := store.LoadPrevRangeMsgs(channelID, channelType, 3, 0, 2)
	require.NoError(t, err)
	require.Len(t, prevMsgs, 2)
	require.Equal(t, uint32(2), prevMsgs[0].MessageSeq)
	require.Equal(t, uint32(3), prevMsgs[1].MessageSeq)

	lastMsgs, err := store.LoadLastMsgs(channelID, channelType, 2)
	require.NoError(t, err)
	require.Len(t, lastMsgs, 2)
	require.Equal(t, uint32(2), lastMsgs[0].MessageSeq)
	require.Equal(t, uint32(3), lastMsgs[1].MessageSeq)

	lastMsgsWithEnd, err := store.LoadLastMsgsWithEnd(channelID, channelType, 1, 2)
	require.NoError(t, err)
	require.Len(t, lastMsgsWithEnd, 2)
	require.Equal(t, uint32(2), lastMsgsWithEnd[0].MessageSeq)
	require.Equal(t, uint32(3), lastMsgsWithEnd[1].MessageSeq)

	msg, err = store.LoadMsgByClientMsgNo(channelID, channelType, "client-2")
	require.NoError(t, err)
	require.Equal(t, int64(102), msg.MessageID)

	seq, err := store.GetUserLastMsgSeq("user-1", channelID, channelType)
	require.NoError(t, err)
	require.Equal(t, uint64(2), seq)

	seqMap, err := store.GetUserLastMsgSeqBatch("user-1", []wkdb.Channel{
		{ChannelId: channelID, ChannelType: channelType},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(2), seqMap[channelID+":"+string(channelType)])

	searchByClient, err := store.SearchMessages(wkdb.MessageSearchReq{
		ChannelId:   channelID,
		ChannelType: channelType,
		ClientMsgNo: "client-2",
		Limit:       10,
	})
	require.NoError(t, err)
	require.Len(t, searchByClient, 1)
	require.Equal(t, int64(102), searchByClient[0].MessageID)

	searchBySender, err := store.SearchMessages(wkdb.MessageSearchReq{
		FromUid: "user-1",
		Limit:   10,
	})
	require.NoError(t, err)
	require.Len(t, searchBySender, 2)

	count, err := store.CountMessages()
	require.NoError(t, err)
	require.Equal(t, 3, count)

	batch, err := store.LoadMsgsBatch([]wkdb.BatchMsgRequest{
		{ChannelId: channelID, ChannelType: channelType, MsgSeq: 2, Limit: 2},
		{ChannelId: channelID, ChannelType: channelType, MsgSeq: 1, Limit: 2, OrderByLast: true},
	})
	require.NoError(t, err)
	require.Len(t, batch, 2)
	require.Len(t, batch[0].Messages, 2)
	require.Len(t, batch[1].Messages, 2)

	require.NoError(t, store.TruncateLogTo(channelID, channelType, 2))

	lastSeq, _, err = store.GetChannelLastMessageSeq(channelID, channelType)
	require.NoError(t, err)
	require.Equal(t, uint64(2), lastSeq)

	truncated, err := store.LoadNextRangeMsgs(channelID, channelType, 3, 0, 10)
	require.NoError(t, err)
	require.Empty(t, truncated)

	searchAfterTruncate, err := store.SearchMessages(wkdb.MessageSearchReq{
		MessageId: 103,
		Limit:     10,
	})
	require.NoError(t, err)
	require.Nil(t, searchAfterTruncate)
}

func TestPebbleChannelLogStoreLeaderTermState(t *testing.T) {
	db := newTestPebbleDBWithNow(t, time.Now)
	defer db.Close()

	store := db.ChannelLogs()
	shardNo := wkutil.ChannelToKey("channel-1", 2)

	require.NoError(t, store.SetLeaderTermStartIndex(shardNo, 1, 10))
	require.NoError(t, store.SetLeaderTermStartIndex(shardNo, 3, 30))

	index, err := store.LeaderTermStartIndex(shardNo, 1)
	require.NoError(t, err)
	require.Equal(t, uint64(10), index)

	lastTerm, err := store.LeaderLastTerm(shardNo)
	require.NoError(t, err)
	require.Equal(t, uint32(3), lastTerm)

	lastGreaterEq, err := store.LeaderLastTermGreaterEqThan(shardNo, 2)
	require.NoError(t, err)
	require.Equal(t, uint32(3), lastGreaterEq)

	require.NoError(t, store.DeleteLeaderTermStartIndexGreaterThanTerm(shardNo, 1))

	lastTerm, err = store.LeaderLastTerm(shardNo)
	require.NoError(t, err)
	require.Equal(t, uint32(1), lastTerm)

	index, err = store.LeaderTermStartIndex(shardNo, 3)
	require.NoError(t, err)
	require.Zero(t, index)
}
