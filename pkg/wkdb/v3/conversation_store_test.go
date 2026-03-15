package v3

import (
	"context"
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/stretchr/testify/require"
)

func TestPebbleConversationStorePutGetListAndSearch(t *testing.T) {
	now := time.Unix(1710000000, 0)
	db := newTestPebbleDBWithNow(t, func() time.Time { return now })
	defer db.Close()

	store := db.Slots().Scope(7).Conversations()
	created1 := time.Unix(1710000000, 100)
	updated1 := time.Unix(1710000001, 0)
	created2 := time.Unix(1710000000, 200)
	updated2 := time.Unix(1710000002, 0)

	err := store.Put("user-1", []wkdb.Conversation{
		{
			Id:           1,
			Type:         wkdb.ConversationTypeChat,
			ChannelId:    "group-100",
			ChannelType:  2,
			UnreadCount:  10,
			ReadToMsgSeq: 5,
			CreatedAt:    &created1,
			UpdatedAt:    &updated1,
		},
		{
			Id:              2,
			Type:            wkdb.ConversationTypeCMD,
			ChannelId:       "group-200",
			ChannelType:     3,
			UnreadCount:     20,
			ReadToMsgSeq:    8,
			DeletedAtMsgSeq: 7,
			CreatedAt:       &created2,
			UpdatedAt:       &updated2,
		},
	})
	require.NoError(t, err)

	conversation, err := store.Get("user-1", "group-200", 3)
	require.NoError(t, err)
	require.Equal(t, uint64(2), conversation.Id)
	require.Equal(t, uint64(7), conversation.DeletedAtMsgSeq)
	require.Equal(t, uint32(20), conversation.UnreadCount)

	list, err := store.ListByUID("user-1")
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, "group-200", list[0].ChannelId)
	require.Equal(t, "group-100", list[1].ChannelId)

	page, err := store.Search(wkdb.ConversationSearchReq{
		Uid:         "user-1",
		Limit:       1,
		CurrentPage: 2,
	})
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "group-100", page[0].ChannelId)

	rawRecords := make([]SlotSnapshotRecord, 0)
	err = db.Slots().Scope(7).Raw().Range(context.Background(), func(key, value []byte) error {
		rawRecords = append(rawRecords, SlotSnapshotRecord{Key: key, Value: value})
		return nil
	})
	require.NoError(t, err)
	require.Len(t, rawRecords, 4)

	var secondIndexCount int
	for _, record := range rawRecords {
		decoded, err := v3key.Decode(record.Key)
		require.NoError(t, err)
		if decoded.Scope == v3key.ScopeSlotSecondIdx && decoded.Table == v3key.TableConversation {
			secondIndexCount++
		}
	}
	require.Equal(t, 2, secondIndexCount)
}

func TestPebbleConversationStoreUpdateAndDelete(t *testing.T) {
	current := time.Unix(1710000000, 0)
	db := newTestPebbleDBWithNow(t, func() time.Time { return current })
	defer db.Close()

	store := db.Slots().Scope(7).Conversations()
	err := store.Put("user-1", []wkdb.Conversation{{
		Id:           1,
		Type:         wkdb.ConversationTypeChat,
		ChannelId:    "group-100",
		ChannelType:  2,
		ReadToMsgSeq: 10,
	}})
	require.NoError(t, err)

	current = current.Add(time.Second)
	require.NoError(t, store.UpdateIfSeqGreater("user-1", "group-100", 2, 9))
	conversation, err := store.Get("user-1", "group-100", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(10), conversation.ReadToMsgSeq)

	current = current.Add(time.Second)
	require.NoError(t, store.UpdateIfSeqGreater("user-1", "group-100", 2, 11))
	conversation, err = store.Get("user-1", "group-100", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(11), conversation.ReadToMsgSeq)

	current = current.Add(time.Second)
	require.NoError(t, store.UpdateDeletedAt("user-1", "group-100", 2, 99))
	conversation, err = store.Get("user-1", "group-100", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(99), conversation.DeletedAtMsgSeq)

	require.NoError(t, store.Put("user-1", []wkdb.Conversation{{
		Id:          2,
		Type:        wkdb.ConversationTypeChat,
		ChannelId:   "group-200",
		ChannelType: 2,
	}}))

	require.NoError(t, store.DeleteBatch("user-1", []wkdb.Channel{{
		ChannelId:   "group-100",
		ChannelType: 2,
	}}))
	_, err = store.Get("user-1", "group-100", 2)
	require.ErrorIs(t, err, wkdb.ErrNotFound)

	require.NoError(t, store.Delete("user-1", "group-200", 2))
	_, err = store.Get("user-1", "group-200", 2)
	require.ErrorIs(t, err, wkdb.ErrNotFound)
}

func TestPebbleConversationStorePutRejectsUIDMismatch(t *testing.T) {
	db := newTestPebbleDBWithNow(t, time.Now)
	defer db.Close()

	err := db.Slots().Scope(7).Conversations().Put("user-1", []wkdb.Conversation{{
		Uid:         "user-2",
		ChannelId:   "group-100",
		ChannelType: 2,
	}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "uid mismatch")
}

func newTestPebbleDBWithNow(t *testing.T, now func() time.Time) *PebbleDB {
	t.Helper()

	router, err := NewStaticBucketRouter(4)
	require.NoError(t, err)

	db, err := NewPebbleDB(PebbleDBOptions{
		DataDir:      "/wkdb-v3",
		Router:       router,
		FS:           vfs.NewMem(),
		WriteOptions: pebble.NoSync,
		PebbleOptions: &pebble.Options{
			FormatMajorVersion: pebble.FormatNewest,
		},
		SnapshotNow: func() int64 { return now().Unix() },
		Now:         now,
	})
	require.NoError(t, err)
	require.NoError(t, db.Open())
	return db
}
