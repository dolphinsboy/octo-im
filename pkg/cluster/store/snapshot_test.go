package store

import (
	"context"
	"testing"
	"time"

	rafttypes "github.com/WuKongIM/WuKongIM/pkg/raft/types"
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/stretchr/testify/require"
)

func TestStoreSlotSnapshotRoundTrip(t *testing.T) {
	var maintenanceCalls []string

	router, err := wkdbv3.NewStaticBucketRouter(4)
	require.NoError(t, err)

	db, err := wkdbv3.NewPebbleDB(wkdbv3.PebbleDBOptions{
		DataDir:      "/wkdb-v3",
		Router:       router,
		FS:           vfs.NewMem(),
		WriteOptions: pebble.NoSync,
		PebbleOptions: &pebble.Options{
			FormatMajorVersion: pebble.FormatNewest,
		},
		SnapshotNow: func() int64 { return 1710000000 },
		Now:         func() time.Time { return time.Unix(1710000000, 0).UTC() },
		ClearSlotCachesFn: func(slotID uint32) {
			maintenanceCalls = append(maintenanceCalls, "clear")
		},
		RebuildDerivedState: func(_ context.Context, slotID uint32) error {
			maintenanceCalls = append(maintenanceCalls, "rebuild")
			return nil
		},
	})
	require.NoError(t, err)
	require.NoError(t, db.Open())
	defer func() {
		require.NoError(t, db.Close())
	}()

	slot7 := db.Slots().Scope(7)
	slot8 := db.Slots().Scope(8)
	createdAt := time.Unix(1710000000, 0).UTC()
	updatedAt := time.Unix(1710000000, 0).UTC()
	mutatedAt := time.Unix(1710000100, 0).UTC()

	require.NoError(t, slot7.Conversations().Put("user-1", []wkdb.Conversation{
		{Id: 1, Uid: "user-1", ChannelId: "group-100", ChannelType: 2, ReadToMsgSeq: 100, CreatedAt: &createdAt, UpdatedAt: &updatedAt},
	}))
	require.NoError(t, slot8.Conversations().Put("user-8", []wkdb.Conversation{
		{Id: 8, Uid: "user-8", ChannelId: "group-800", ChannelType: 2, ReadToMsgSeq: 800, CreatedAt: &createdAt, UpdatedAt: &updatedAt},
	}))

	s := New(NewOptions(WithSlotSnapshotBackend(db)))
	snapshot, err := s.CreateSlotSnapshot(7)
	require.NoError(t, err)

	require.NoError(t, slot7.Conversations().Put("user-2", []wkdb.Conversation{
		{Id: 2, Uid: "user-2", ChannelId: "group-200", ChannelType: 2, ReadToMsgSeq: 200, CreatedAt: &mutatedAt, UpdatedAt: &mutatedAt},
	}))

	require.NoError(t, s.ApplySlotSnapshot(7, snapshot))
	require.Equal(t, []string{"clear", "rebuild"}, maintenanceCalls)

	conv, err := slot7.Conversations().Get("user-1", "group-100", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(100), conv.ReadToMsgSeq)

	_, err = slot7.Conversations().Get("user-2", "group-200", 2)
	require.ErrorIs(t, err, wkdb.ErrNotFound)

	conv8, err := slot8.Conversations().Get("user-8", "group-800", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(800), conv8.ReadToMsgSeq)
}

func TestStoreCreateSlotSnapshotUnsupported(t *testing.T) {
	s := New(NewOptions())

	_, err := s.CreateSlotSnapshot(7)
	require.ErrorIs(t, err, rafttypes.ErrSnapshotNotSupported)
}

func TestStoreApplySlotSnapshotRejectsSlotMismatch(t *testing.T) {
	router, err := wkdbv3.NewStaticBucketRouter(4)
	require.NoError(t, err)

	db, err := wkdbv3.NewPebbleDB(wkdbv3.PebbleDBOptions{
		DataDir:      "/wkdb-v3",
		Router:       router,
		FS:           vfs.NewMem(),
		WriteOptions: pebble.NoSync,
		PebbleOptions: &pebble.Options{
			FormatMajorVersion: pebble.FormatNewest,
		},
		SnapshotNow:       func() int64 { return 1710000000 },
		Now:               func() time.Time { return time.Unix(1710000000, 0).UTC() },
		ClearSlotCachesFn: func(slotID uint32) {},
		RebuildDerivedState: func(_ context.Context, slotID uint32) error {
			return nil
		},
	})
	require.NoError(t, err)
	require.NoError(t, db.Open())
	defer func() {
		require.NoError(t, db.Close())
	}()

	s := New(NewOptions(WithSlotSnapshotBackend(db)))
	snapshot, err := s.CreateSlotSnapshot(7)
	require.NoError(t, err)

	err = s.ApplySlotSnapshot(8, snapshot)
	require.ErrorContains(t, err, "slot snapshot slot mismatch")
}
