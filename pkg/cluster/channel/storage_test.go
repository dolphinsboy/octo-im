package channel

import (
	"testing"

	rafttypes "github.com/WuKongIM/WuKongIM/pkg/raft/types"
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/WuKongIM/WuKongIM/pkg/wkutil"
	wkproto "github.com/WuKongIM/WuKongIMGoProto"
	"github.com/stretchr/testify/require"
)

type stubChannelLogStore struct {
	lastMsg        wkdb.Message
	lastMsgErr     error
	appended       []wkdb.Message
	termStartKey   string
	termStartTerm  uint32
	termStartIndex uint64
}

var _ ChannelLogStore = (*stubChannelLogStore)(nil)

func (s *stubChannelLogStore) GetLastMsg(channelId string, channelType uint8) (wkdb.Message, error) {
	return s.lastMsg, s.lastMsgErr
}

func (s *stubChannelLogStore) AppendMessages(channelId string, channelType uint8, msgs []wkdb.Message) error {
	s.appended = append(s.appended, msgs...)
	return nil
}

func (s *stubChannelLogStore) LoadNextRangeMsgsForSize(channelId string, channelType uint8, startMessageSeq, endMessageSeq, limitSize uint64) ([]wkdb.Message, error) {
	return nil, nil
}

func (s *stubChannelLogStore) TruncateLogTo(channelId string, channelType uint8, messageSeq uint64) error {
	return nil
}

func (s *stubChannelLogStore) GetChannelLastMessageSeq(channelId string, channelType uint8) (seq uint64, lastTime uint64, err error) {
	return uint64(s.lastMsg.MessageSeq), uint64(s.lastMsg.Timestamp), nil
}

func (s *stubChannelLogStore) SetLeaderTermStartIndex(shardNo string, term uint32, index uint64) error {
	s.termStartKey = shardNo
	s.termStartTerm = term
	s.termStartIndex = index
	return nil
}

func (s *stubChannelLogStore) LeaderTermStartIndex(shardNo string, term uint32) (uint64, error) {
	return 0, nil
}

func (s *stubChannelLogStore) LeaderLastTerm(shardNo string) (uint32, error) {
	return 0, nil
}

func (s *stubChannelLogStore) LeaderLastTermGreaterEqThan(shardNo string, term uint32) (uint32, error) {
	return 0, nil
}

func (s *stubChannelLogStore) DeleteLeaderTermStartIndexGreaterThanTerm(shardNo string, term uint32) error {
	return nil
}

func TestStorageGetStateUsesChannelLogStore(t *testing.T) {
	store := &stubChannelLogStore{
		lastMsg: wkdb.Message{
			RecvPacket: wkproto.RecvPacket{
				MessageID: 7,
			},
			Term: 11,
		},
	}
	store.lastMsg.MessageSeq = 23

	s := newStorage(store, &Server{})
	state, err := s.GetState("channel-1", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(23), state.LastLogIndex)
	require.Equal(t, uint32(11), state.LastTerm)
	require.Equal(t, uint64(23), state.AppliedIndex)
}

func TestStorageAppendLogsUsesNarrowChannelLogStore(t *testing.T) {
	store := &stubChannelLogStore{}
	s := newStorage(store, &Server{})

	msg := wkdb.Message{
		RecvPacket: wkproto.RecvPacket{
			MessageID:   99,
			ChannelID:   "channel-1",
			ChannelType: 2,
			FromUID:     "user-1",
			ClientMsgNo: "client-1",
			Timestamp:   100,
			Payload:     []byte("hello"),
		},
	}
	data, err := msg.Marshal()
	require.NoError(t, err)

	channelKey := wkutil.ChannelToKey("channel-1", 2)
	err = s.AppendLogs(channelKey, []rafttypes.Log{{
		Id:    99,
		Index: 8,
		Term:  3,
		Data:  data,
	}}, &rafttypes.TermStartIndexInfo{
		Term:  3,
		Index: 8,
	})
	require.NoError(t, err)

	require.Len(t, store.appended, 1)
	require.Equal(t, uint32(8), store.appended[0].MessageSeq)
	require.Equal(t, uint64(3), store.appended[0].Term)
	require.Equal(t, int64(99), store.appended[0].MessageID)
	require.Equal(t, channelKey, store.termStartKey)
	require.Equal(t, uint32(3), store.termStartTerm)
	require.Equal(t, uint64(8), store.termStartIndex)
}
