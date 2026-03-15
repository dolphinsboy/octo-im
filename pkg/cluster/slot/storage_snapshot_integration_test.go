package slot

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	clusterstore "github.com/WuKongIM/WuKongIM/pkg/cluster/store"
	rafttypes "github.com/WuKongIM/WuKongIM/pkg/raft/types"
	"github.com/WuKongIM/WuKongIM/pkg/trace"
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
	"github.com/WuKongIM/WuKongIM/pkg/wkutil"
	"github.com/cockroachdb/pebble"
	"github.com/stretchr/testify/require"
)

func TestPebbleShardLogStorageSnapshotRoundTripWithStoreV3(t *testing.T) {
	const slotCount uint32 = 64

	sourceBundle := newSnapshotTestBundle(t, slotCount)
	defer sourceBundle.close()
	targetBundle := newSnapshotTestBundle(t, slotCount)
	defer targetBundle.close()

	uid := "user-snapshot-1"
	slotID := wkutil.GetSlotNum(int(slotCount), uid)
	channelID := channelOnSlot(slotCount, slotID)

	logs := []rafttypes.Log{
		mustStoreLog(t, 1, clusterstore.NewCMD(clusterstore.CMDAddUser, clusterstore.EncodeCMDUser(wkdb.User{Id: 11, Uid: uid}))),
		mustStoreLog(t, 2, clusterstore.NewCMD(clusterstore.CMDAddDevice, clusterstore.EncodeCMDDevice(wkdb.Device{Id: 21, Uid: uid, DeviceFlag: 1, Token: "token-1"}))),
		mustStoreLog(t, 3, mustChannelInfoCMD(t, clusterstore.CMDAddChannelInfo, wkdb.ChannelInfo{Id: 31, ChannelId: channelID, ChannelType: 2, Ban: false})),
		mustStoreLog(t, 4, clusterstore.NewCMD(clusterstore.CMDAddSubscribers, clusterstore.EncodeMembers(channelID, 2, []wkdb.Member{{Uid: uid}}))),
		mustStoreLog(t, 5, mustConversationsWithUserCMD(t, uid, []wkdb.Conversation{{Id: 41, Uid: uid, Type: wkdb.ConversationTypeChat, ChannelId: channelID, ChannelType: 2, ReadToMsgSeq: 88}})),
		mustStoreLog(t, 6, mustChannelClusterConfigCMD(t, channelID, 2, wkdb.ChannelClusterConfig{ChannelId: channelID, ChannelType: 2, LeaderId: 1001, ReplicaMaxCount: 3, Replicas: []uint64{1001, 1002, 1003}})),
		mustStoreLog(t, 7, mustMessageEventCMD(t, &wkdb.MessageEvent{ChannelId: channelID, ChannelType: 2, ClientMsgNo: "client-1", EventID: "evt-1", EventKey: wkdb.EventKeyDefault, EventType: wkdb.EventTypeStreamSnapshot, Payload: []byte(`{"kind":"text","text":"hello"}`)})),
	}

	require.NoError(t, sourceBundle.store.ApplySlotLogs(slotID, logs))

	storedUser, err := sourceBundle.hybrid.GetUser(uid)
	require.NoError(t, err)
	require.Equal(t, uint64(11), storedUser.Id)

	storedConversation, err := sourceBundle.hybrid.GetConversation(uid, channelID, 2)
	require.NoError(t, err)
	require.Equal(t, uint64(88), storedConversation.ReadToMsgSeq)

	shardNo := SlotIdToKey(slotID)
	snapshotBytes, err := sourceBundle.storage.CreateSnapshot(shardNo, uint64(len(logs)))
	require.NoError(t, err)
	require.NotEmpty(t, snapshotBytes)

	require.NoError(t, targetBundle.store.ApplySlotLogs(slotID, []rafttypes.Log{
		mustStoreLog(t, 1, clusterstore.NewCMD(clusterstore.CMDAddUser, clusterstore.EncodeCMDUser(wkdb.User{Id: 99, Uid: uid, PluginNo: "stale"}))),
	}))
	require.NoError(t, targetBundle.storage.AppendLogs(shardNo, makeLogs(1, 3, 1), &rafttypes.TermStartIndexInfo{Term: 1, Index: 1}))
	require.NoError(t, targetBundle.storage.SetAppliedIndex(shardNo, 3))

	snapshotData := rafttypes.SnapshotData{
		Meta: rafttypes.Snapshot{
			LastIncludedIndex: uint64(len(logs)),
			LastIncludedTerm:  1,
			Size:              uint64(len(snapshotBytes)),
			CreatedAt:         time.Now().UnixNano(),
		},
		Data: snapshotBytes,
	}
	require.NoError(t, targetBundle.storage.SaveSnapshot(shardNo, snapshotData))
	require.NoError(t, targetBundle.storage.ApplySnapshot(shardNo, snapshotData))

	restoredUser, err := targetBundle.hybrid.GetUser(uid)
	require.NoError(t, err)
	require.Equal(t, uint64(11), restoredUser.Id)

	restoredDevice, err := targetBundle.hybrid.GetDevice(uid, 1)
	require.NoError(t, err)
	require.Equal(t, "token-1", restoredDevice.Token)

	restoredChannel, err := targetBundle.hybrid.GetChannel(channelID, 2)
	require.NoError(t, err)
	require.Equal(t, 1, restoredChannel.SubscriberCount)

	restoredSubscribers, err := targetBundle.hybrid.GetSubscribers(channelID, 2)
	require.NoError(t, err)
	require.Len(t, restoredSubscribers, 1)
	require.Equal(t, uid, restoredSubscribers[0].Uid)

	restoredConversation, err := targetBundle.hybrid.GetConversation(uid, channelID, 2)
	require.NoError(t, err)
	require.Equal(t, uint64(88), restoredConversation.ReadToMsgSeq)

	restoredCfg, err := targetBundle.hybrid.GetChannelClusterConfig(channelID, 2)
	require.NoError(t, err)
	require.Equal(t, uint64(6), restoredCfg.ConfVersion)
	require.Equal(t, uint64(1001), restoredCfg.LeaderId)

	restoredEventState, err := targetBundle.hybrid.GetMessageEventState(channelID, 2, "client-1", wkdb.EventKeyDefault)
	require.NoError(t, err)
	require.NotNil(t, restoredEventState)
	require.Equal(t, uint64(1), restoredEventState.LastMsgEventSeq)
	require.Equal(t, []byte(`{"kind":"text","text":"hello"}`), restoredEventState.SnapshotPayload)

	state, err := targetBundle.storage.GetState(shardNo)
	require.NoError(t, err)
	require.Equal(t, uint64(len(logs)), state.AppliedIndex)
	require.Equal(t, uint64(len(logs)), state.CompactedIndex)

	remainingLogs, err := targetBundle.storage.GetLogs(shardNo, 1, 100, 0)
	require.NoError(t, err)
	require.Empty(t, remainingLogs)
}

type snapshotTestBundle struct {
	legacy  wkdb.DB
	hybrid  *clusterstore.HybridDB
	store   *clusterstore.Store
	storage *PebbleShardLogStorage
	cleanup func()
}

func newSnapshotTestBundle(t *testing.T, slotCount uint32) *snapshotTestBundle {
	t.Helper()

	if trace.GlobalTrace == nil {
		trace.SetGlobalTrace(trace.New(context.Background(), trace.NewOptions()))
	}

	rootDir := t.TempDir()
	legacy := wkdb.NewWukongDB(wkdb.NewOptions(
		wkdb.WithDir(filepath.Join(rootDir, "legacy")),
		wkdb.WithShardNum(1),
		wkdb.WithNodeId(1),
		wkdb.WithSlotCount(int(slotCount)),
	))

	router, err := wkdbv3.NewStaticBucketRouter(4)
	require.NoError(t, err)
	slotDB, err := wkdbv3.NewPebbleDB(wkdbv3.PebbleDBOptions{
		DataDir:      filepath.Join(rootDir, "v3"),
		Router:       router,
		WriteOptions: pebble.NoSync,
		PebbleOptions: &pebble.Options{
			FormatMajorVersion: pebble.FormatNewest,
		},
		ClearSlotCachesFn: func(slotID uint32) {},
		RebuildDerivedState: func(ctx context.Context, slotID uint32) error {
			return nil
		},
	})
	require.NoError(t, err)

	hybrid := clusterstore.NewHybridDB(legacy, slotDB, slotCount, func(key string) uint32 {
		return wkutil.GetSlotNum(int(slotCount), key)
	})
	require.NoError(t, hybrid.Open())

	st := clusterstore.New(clusterstore.NewOptions(clusterstore.WithDB(hybrid)))
	require.NoError(t, st.Start())

	server := &Server{opts: &Options{
		OnCreateSnapshot: st.CreateSlotSnapshot,
		OnApplySnapshot:  st.ApplySlotSnapshot,
	}}
	storage := NewPebbleShardLogStorage(server, filepath.Join(rootDir, "slot"), 1)
	require.NoError(t, storage.Open())

	return &snapshotTestBundle{
		legacy:  legacy,
		hybrid:  hybrid,
		store:   st,
		storage: storage,
		cleanup: func() {
			st.Stop()
			require.NoError(t, storage.Close())
			require.NoError(t, hybrid.Close())
		},
	}
}

func (b *snapshotTestBundle) close() {
	if b.cleanup != nil {
		b.cleanup()
	}
}

func channelOnSlot(slotCount, slotID uint32) string {
	for i := 0; i < 10000; i++ {
		channelID := fmt.Sprintf("channel-%d", i)
		if wkutil.GetSlotNum(int(slotCount), channelID) == slotID {
			return channelID
		}
	}
	panic("unable to find channel id for slot")
}

func mustStoreLog(t *testing.T, index uint64, cmd *clusterstore.CMD) rafttypes.Log {
	t.Helper()
	data, err := cmd.Marshal()
	require.NoError(t, err)
	return rafttypes.Log{Id: index, Index: index, Term: 1, Data: data}
}

func mustChannelInfoCMD(t *testing.T, cmdType clusterstore.CMDType, info wkdb.ChannelInfo) *clusterstore.CMD {
	t.Helper()
	data, err := clusterstore.EncodeChannelInfo(info, clusterstore.CmdVersionChannelInfo)
	require.NoError(t, err)
	return clusterstore.NewCMDWithVersion(cmdType, data, clusterstore.CmdVersionChannelInfo)
}

func mustConversationsWithUserCMD(t *testing.T, uid string, conversations []wkdb.Conversation) *clusterstore.CMD {
	t.Helper()
	data, err := clusterstore.EncodeCMDAddOrUpdateUserConversations(uid, conversations)
	require.NoError(t, err)
	return clusterstore.NewCMD(clusterstore.CMDAddOrUpdateUserConversations, data)
}

func mustChannelClusterConfigCMD(t *testing.T, channelID string, channelType uint8, cfg wkdb.ChannelClusterConfig) *clusterstore.CMD {
	t.Helper()
	cfgData, err := cfg.Marshal()
	require.NoError(t, err)
	data, err := clusterstore.EncodeCMDChannelClusterConfigSave(channelID, channelType, cfgData)
	require.NoError(t, err)
	return clusterstore.NewCMD(clusterstore.CMDChannelClusterConfigSave, data)
}

func mustMessageEventCMD(t *testing.T, event *wkdb.MessageEvent) *clusterstore.CMD {
	t.Helper()
	data := clusterstore.EncodeCMDMessageEvent(event, clusterstore.CmdVersionMessageEvent)
	return clusterstore.NewCMDWithVersion(clusterstore.CMDAppendMessageEvent, data, clusterstore.CmdVersionMessageEvent)
}
