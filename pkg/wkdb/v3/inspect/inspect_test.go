package inspect

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	"github.com/cockroachdb/pebble"
	"github.com/stretchr/testify/require"
)

func TestInspectorReadOnlyQueriesAndRawDump(t *testing.T) {
	dataDir := t.TempDir()
	router, err := wkdbv3.NewStaticBucketRouter(4)
	require.NoError(t, err)

	db, err := wkdbv3.NewPebbleDB(wkdbv3.PebbleDBOptions{
		DataDir:       dataDir,
		Router:        router,
		WriteOptions:  pebble.NoSync,
		PebbleOptions: &pebble.Options{FormatMajorVersion: pebble.FormatNewest},
		ClearSlotCachesFn: func(slotID uint32) {
		},
		RebuildDerivedState: func(ctx context.Context, slotID uint32) error {
			return nil
		},
	})
	require.NoError(t, err)
	require.NoError(t, db.Open())

	slotID := uint32(7)
	scope := db.Slots().Scope(slotID)
	now := time.Unix(1710000000, 0).UTC()
	require.NoError(t, scope.Users().Put(wkdb.User{Id: 11, Uid: "user-1", PluginNo: "plugin-a", CreatedAt: &now, UpdatedAt: &now}))
	require.NoError(t, scope.Devices().Put(wkdb.Device{Id: 21, Uid: "user-1", DeviceFlag: 1, DeviceLevel: 1, Token: "token-1", CreatedAt: &now, UpdatedAt: &now}))
	require.NoError(t, scope.Conversations().Put("user-1", []wkdb.Conversation{{Id: 31, Uid: "user-1", Type: wkdb.ConversationTypeChat, ChannelId: "group-1", ChannelType: 2, ReadToMsgSeq: 88, CreatedAt: &now, UpdatedAt: &now}}))
	require.NoError(t, scope.Channels().Put(wkdb.ChannelInfo{Id: 41, ChannelId: "group-1", ChannelType: 2, CreatedAt: &now, UpdatedAt: &now}))
	require.NoError(t, scope.Subscribers().Put("group-1", 2, []wkdb.Member{{Uid: "user-1", CreatedAt: &now, UpdatedAt: &now}}))
	require.NoError(t, scope.Allowlists().Put("group-1", 2, []wkdb.Member{{Uid: "allow-1", CreatedAt: &now, UpdatedAt: &now}}))
	require.NoError(t, scope.Denylists().Put("group-1", 2, []wkdb.Member{{Uid: "deny-1", CreatedAt: &now, UpdatedAt: &now}}))
	require.NoError(t, scope.ChannelClusterConfigs().Put(wkdb.ChannelClusterConfig{Id: 51, ChannelId: "group-1", ChannelType: 2, ReplicaMaxCount: 3, Replicas: []uint64{1001, 1002, 1003}, LeaderId: 1001, ConfVersion: 9, CreatedAt: &now, UpdatedAt: &now}))
	require.NoError(t, scope.MessageEvents().PutState(wkdb.MessageEventState{ChannelId: "group-1", ChannelType: 2, ClientMsgNo: "client-1", EventKey: wkdb.EventKeyDefault, LastMsgEventSeq: 3, SnapshotPayload: []byte(`{"kind":"text"}`)}))
	require.NoError(t, scope.MessageEvents().SetSeq("group-1", 2, "client-1", 9))
	require.NoError(t, db.Close())

	inspector, err := Open(Options{DataDir: dataDir})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, inspector.Close())
	}()

	require.Equal(t, uint32(4), inspector.BucketCount())
	require.Equal(t, uint32(3), inspector.BucketForSlot(slotID))
	existingSlots, err := inspector.ExistingSlots(context.Background())
	require.NoError(t, err)
	require.Equal(t, []uint32{slotID}, existingSlots)

	user, err := inspector.GetUser(slotID, "user-1")
	require.NoError(t, err)
	require.Equal(t, "plugin-a", user.PluginNo)

	devices, err := inspector.ListDevices(slotID, "user-1", 10)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	require.Equal(t, "token-1", devices[0].Token)

	conversations, err := inspector.ListConversations(slotID, "user-1", 10)
	require.NoError(t, err)
	require.Len(t, conversations, 1)
	require.Equal(t, uint64(88), conversations[0].ReadToMsgSeq)

	channel, err := inspector.GetChannel(slotID, "group-1", 2)
	require.NoError(t, err)
	require.Equal(t, 1, channel.SubscriberCount)

	subscribers, err := inspector.ListSubscribers(slotID, "group-1", 2)
	require.NoError(t, err)
	require.Len(t, subscribers, 1)
	require.Equal(t, "user-1", subscribers[0].Uid)

	allowlist, err := inspector.ListAllowlists(slotID, "group-1", 2)
	require.NoError(t, err)
	require.Len(t, allowlist, 1)
	require.Equal(t, "allow-1", allowlist[0].Uid)

	denylist, err := inspector.ListDenylists(slotID, "group-1", 2)
	require.NoError(t, err)
	require.Len(t, denylist, 1)
	require.Equal(t, "deny-1", denylist[0].Uid)

	cfg, err := inspector.GetChannelClusterConfig(slotID, "group-1", 2)
	require.NoError(t, err)
	require.Equal(t, uint64(1001), cfg.LeaderId)
	require.Equal(t, uint64(9), cfg.ConfVersion)

	state, err := inspector.GetMessageEventState(slotID, "group-1", 2, "client-1", wkdb.EventKeyDefault)
	require.NoError(t, err)
	require.NotNil(t, state)
	require.Equal(t, uint64(3), state.LastMsgEventSeq)

	records, err := inspector.RawDumpSlot(context.Background(), slotID, 4)
	require.NoError(t, err)
	require.Len(t, records, 4)
	for _, record := range records {
		require.Equal(t, slotID, record.SlotID)
		require.NotEmpty(t, record.KeyHex)
		require.NotEmpty(t, record.Table)
	}

	filtered, err := inspector.RawDumpSlotWithOptions(context.Background(), slotID, RawDumpOptions{
		Tables:       []v3key.TableID{v3key.TableUser},
		Scopes:       []v3key.ScopeType{v3key.ScopeSlotSecondIdx},
		DecodeValues: true,
	})
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	require.Equal(t, "slot_second_index", filtered[0].Scope)
	require.Equal(t, "user", filtered[0].Table)
	require.NotNil(t, filtered[0].KeyDecoded)
	require.Equal(t, "user-1", filtered[0].Decoded)

	auxRecords, err := inspector.RawDumpSlotWithOptions(context.Background(), slotID, RawDumpOptions{
		Tables:       []v3key.TableID{v3key.TableMessageEventSeq},
		Scopes:       []v3key.ScopeType{v3key.ScopeSlotAux},
		DecodeValues: true,
	})
	require.NoError(t, err)
	require.Len(t, auxRecords, 1)
	require.Equal(t, uint64(9), auxRecords[0].Decoded)

	userSecondIndexKey := v3key.EncodeUserCreatedAtSecondIndexKey(slotID, uint64(now.UnixNano()), "user-1")
	userSecondIndexMeta, err := v3key.Decode(userSecondIndexKey)
	require.NoError(t, err)

	prefixFiltered, err := inspector.RawDumpSlotWithOptions(context.Background(), slotID, RawDumpOptions{
		KeyFilter: RawKeyFilter{
			KeyPrefix:  userSecondIndexKey,
			TailPrefix: userSecondIndexMeta.Tail,
		},
	})
	require.NoError(t, err)
	require.Len(t, prefixFiltered, 1)
	require.Equal(t, "user", prefixFiltered[0].Table)
	require.Equal(t, "slot_second_index", prefixFiltered[0].Scope)

	channelFiltered, err := inspector.RawDumpSlotWithOptions(context.Background(), slotID, RawDumpOptions{
		KeyFilter: RawKeyFilter{
			ChannelID:   "group-1",
			ChannelType: func() *uint8 { v := uint8(2); return &v }(),
			ClientMsgNo: "client-1",
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, channelFiltered)
	for _, record := range channelFiltered {
		require.NotNil(t, record.KeyDecoded)
	}
}

func TestDetectBucketCountRejectsSparseBuckets(t *testing.T) {
	dataDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "buckets", "bucket-000"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "buckets", "bucket-002"), 0o755))

	_, err := DetectBucketCount(dataDir)
	require.ErrorContains(t, err, "not contiguous")
}
