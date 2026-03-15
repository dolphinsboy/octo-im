package v3

import (
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/stretchr/testify/require"
)

func TestPebbleMessageEventStorePutGetListDelete(t *testing.T) {
	db := newTestPebbleDBWithNow(t, time.Now)
	defer db.Close()

	store := db.Slots().Scope(7).MessageEvents()
	require.NoError(t, store.PutState(wkdb.MessageEventState{
		ChannelId:       "group-1",
		ChannelType:     2,
		ClientMsgNo:     "client-1",
		EventKey:        "",
		Status:          wkdb.EventStatusOpen,
		LastMsgEventSeq: 1,
		LastEventID:     "evt-1",
		SnapshotPayload: []byte("hello"),
	}))
	require.NoError(t, store.PutState(wkdb.MessageEventState{
		ChannelId:       "group-1",
		ChannelType:     2,
		ClientMsgNo:     "client-1",
		EventKey:        "tool",
		Status:          wkdb.EventStatusClosed,
		LastMsgEventSeq: 2,
		LastEventID:     "evt-2",
		SnapshotPayload: []byte("world"),
	}))
	require.NoError(t, store.SetSeq("group-1", 2, "client-1", 9))

	state, err := store.GetState("group-1", 2, "client-1", "")
	require.NoError(t, err)
	require.NotNil(t, state)
	require.Equal(t, wkdb.EventKeyDefault, state.EventKey)
	require.Equal(t, uint64(1), state.LastMsgEventSeq)
	require.Equal(t, []byte("hello"), state.SnapshotPayload)

	states, err := store.GetStates("group-1", 2, "client-1")
	require.NoError(t, err)
	require.Len(t, states, 2)

	seq, err := store.GetSeq("group-1", 2, "client-1")
	require.NoError(t, err)
	require.Equal(t, uint64(9), seq)

	require.NoError(t, store.DeleteStates("group-1", 2, "client-1"))
	state, err = store.GetState("group-1", 2, "client-1", "")
	require.NoError(t, err)
	require.Nil(t, state)
	states, err = store.GetStates("group-1", 2, "client-1")
	require.NoError(t, err)
	require.Nil(t, states)
	seq, err = store.GetSeq("group-1", 2, "client-1")
	require.NoError(t, err)
	require.Equal(t, uint64(0), seq)
}

func TestPebbleMessageEventStoreEmptyInputs(t *testing.T) {
	db := newTestPebbleDBWithNow(t, time.Now)
	defer db.Close()

	store := db.Slots().Scope(7).MessageEvents()
	state, err := store.GetState("group-1", 2, "", "")
	require.NoError(t, err)
	require.Nil(t, state)

	states, err := store.GetStates("group-1", 2, "")
	require.NoError(t, err)
	require.Nil(t, states)

	seq, err := store.GetSeq("group-1", 2, "")
	require.NoError(t, err)
	require.Equal(t, uint64(0), seq)

	err = store.SetSeq("group-1", 2, "", 1)
	require.Error(t, err)
}
