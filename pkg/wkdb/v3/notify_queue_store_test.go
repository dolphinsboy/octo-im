package v3

import (
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkproto "github.com/WuKongIM/WuKongIMGoProto"
	"github.com/stretchr/testify/require"
)

func TestPebbleNotifyQueueStorePutListDelete(t *testing.T) {
	db := newTestPebbleDBWithNow(t, time.Now)
	defer db.Close()

	store := db.Local().NotifyQueue()
	messages := []wkdb.Message{
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
		{
			RecvPacket: wkproto.RecvPacket{
				MessageID:   2,
				ChannelID:   "channel-2",
				ChannelType: 2,
				FromUID:     "user-2",
				ClientMsgNo: "client-2",
				Timestamp:   200,
				Payload:     []byte("world"),
			},
		},
	}

	require.NoError(t, store.Put(messages))

	listed, err := store.List(10)
	require.NoError(t, err)
	require.Len(t, listed, 2)
	require.Equal(t, int64(1), listed[0].MessageID)
	require.Equal(t, int64(2), listed[1].MessageID)

	require.NoError(t, store.Delete([]int64{1}))
	listed, err = store.List(10)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, int64(2), listed[0].MessageID)

	require.NoError(t, store.DeleteCount(1))
	listed, err = store.List(10)
	require.NoError(t, err)
	require.Nil(t, listed)
}
