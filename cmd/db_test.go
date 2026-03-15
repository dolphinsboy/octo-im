package cmd

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/internal/options"
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	"github.com/WuKongIM/WuKongIM/pkg/wkutil"
	"github.com/cockroachdb/pebble"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type slotScopedUserPayload struct {
	SlotID   uint32    `json:"slot_id"`
	BucketID uint32    `json:"bucket_id"`
	Data     wkdb.User `json:"data"`
}

func TestDBTablesCommand(t *testing.T) {
	dataDir := createTestDBV3Data(t)
	cmd := newDbCMD(&WuKongIMContext{}).CMD()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)
	cmd.SetArgs([]string{"tables", "--data-dir", dataDir, "--format", "json"})

	require.NoError(t, cmd.Execute())

	var summary map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &summary))
	require.Equal(t, dataDir, summary["data_dir"])
	require.Equal(t, float64(4), summary["bucket_count"])
	tables, ok := summary["tables"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, tables)
}

func TestDBGetUserCommand(t *testing.T) {
	dataDir := createTestDBV3Data(t)
	cmd := newDbCMD(&WuKongIMContext{}).CMD()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)
	cmd.SetArgs([]string{"get", "user", "--data-dir", dataDir, "--slot", "7", "--uid", "user-1", "--format", "json"})

	require.NoError(t, cmd.Execute())

	var user wkdb.User
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &user))
	require.Equal(t, "user-1", user.Uid)
	require.Equal(t, "plugin-a", user.PluginNo)
}

func TestDBListSubscribersTableCommand(t *testing.T) {
	dataDir := createTestDBV3Data(t)
	cmd := newDbCMD(&WuKongIMContext{}).CMD()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)
	cmd.SetArgs([]string{"list", "subscribers", "--data-dir", dataDir, "--slot", "7", "--channel-id", "group-1", "--channel-type", "2", "--format", "table"})

	require.NoError(t, cmd.Execute())
	output := stdout.String()
	require.Contains(t, output, "uid")
	require.Contains(t, output, "user-1")
}

func TestDBRawDumpTableCommand(t *testing.T) {
	dataDir := createTestDBV3Data(t)
	cmd := newDbCMD(&WuKongIMContext{}).CMD()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)
	cmd.SetArgs([]string{"raw", "dump", "--data-dir", dataDir, "--slot", "7", "--limit", "2", "--format", "table"})

	require.NoError(t, cmd.Execute())
	output := stdout.String()
	require.Contains(t, output, "scope")
	require.Contains(t, output, "table")
	require.Contains(t, output, "key_decoded")
	require.Contains(t, output, "decoded")
	require.Contains(t, output, "user")
}

func TestDBRawDumpJSONFiltersAndDecodes(t *testing.T) {
	dataDir := createTestDBV3Data(t)
	cmd := newDbCMD(&WuKongIMContext{}).CMD()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)
	cmd.SetArgs([]string{
		"raw", "dump",
		"--data-dir", dataDir,
		"--slot", "7",
		"--table", "user",
		"--scope", "second_index",
		"--format", "json",
	})

	require.NoError(t, cmd.Execute())

	var payload []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	require.Len(t, payload, 1)
	require.Equal(t, "slot_second_index", payload[0]["scope"])
	require.Equal(t, "user", payload[0]["table"])
	require.NotNil(t, payload[0]["key_decoded"])
	require.Equal(t, "user-1", payload[0]["decoded"])
}

func TestDBRawDumpDisableDecode(t *testing.T) {
	dataDir := createTestDBV3Data(t)
	cmd := newDbCMD(&WuKongIMContext{}).CMD()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)
	cmd.SetArgs([]string{
		"raw", "dump",
		"--data-dir", dataDir,
		"--slot", "7",
		"--table", "user",
		"--scope", "second_index",
		"--decode=false",
		"--format", "json",
	})

	require.NoError(t, cmd.Execute())

	var payload []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	require.Len(t, payload, 1)
	_, exists := payload[0]["decoded"]
	require.False(t, exists)
	_, exists = payload[0]["key_decoded"]
	require.True(t, exists)
}

func TestDBRawDumpBusinessKeyFilters(t *testing.T) {
	dataDir := createTestDBV3Data(t)
	cmd := newDbCMD(&WuKongIMContext{}).CMD()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)
	cmd.SetArgs([]string{
		"raw", "dump",
		"--data-dir", dataDir,
		"--slot", "7",
		"--table", "message-event-state,message-event-seq",
		"--channel-id", "group-1",
		"--channel-type", "2",
		"--client-msg-no", "client-1",
		"--format", "json",
	})

	require.NoError(t, cmd.Execute())

	var payload []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	require.Len(t, payload, 2)
	for _, item := range payload {
		keyDecoded, ok := item["key_decoded"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "group-1", keyDecoded["channel_id"])
		require.Equal(t, "client-1", keyDecoded["client_msg_no"])
	}
}

func TestDBRawDumpTailPrefixFilter(t *testing.T) {
	dataDir := createTestDBV3Data(t)
	key := v3key.EncodeUserCreatedAtSecondIndexKey(7, uint64(time.Unix(1710000000, 0).UTC().UnixNano()), "user-1")
	meta, err := v3key.Decode(key)
	require.NoError(t, err)

	cmd := newDbCMD(&WuKongIMContext{}).CMD()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)
	cmd.SetArgs([]string{
		"raw", "dump",
		"--data-dir", dataDir,
		"--slot", "7",
		"--table", "user",
		"--tail-prefix", hex.EncodeToString(meta.Tail),
		"--format", "json",
	})

	require.NoError(t, cmd.Execute())

	var payload []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	require.Len(t, payload, 1)
	require.Equal(t, "user", payload[0]["table"])
	require.Equal(t, "slot_second_index", payload[0]["scope"])
}

func TestParseSlotSelector(t *testing.T) {
	selector, err := parseSlotSelector("1,3-5,8")
	require.NoError(t, err)
	require.False(t, selector.all)
	require.Equal(t, []slotInterval{
		{start: 1, end: 1},
		{start: 3, end: 5},
		{start: 8, end: 8},
	}, selector.intervals)

	selector, err = parseSlotSelector("all")
	require.NoError(t, err)
	require.True(t, selector.all)

	_, err = parseSlotSelector("9-7")
	require.ErrorContains(t, err, "start is greater than end")
}

func TestDBListUsersRangeCommand(t *testing.T) {
	dataDir := createTestDBV3Data(t)
	cmd := newDbCMD(&WuKongIMContext{}).CMD()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)
	cmd.SetArgs([]string{"list", "users", "--data-dir", dataDir, "--slot", "7-9", "--format", "json"})

	require.NoError(t, cmd.Execute())

	var payload []slotScopedUserPayload
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	require.Len(t, payload, 2)
	require.Equal(t, uint32(7), payload[0].SlotID)
	require.Equal(t, "user-1", payload[0].Data.Uid)
	require.Equal(t, uint32(9), payload[1].SlotID)
	require.Equal(t, "user-9", payload[1].Data.Uid)
}

func TestDBGetUserAllSlotsCommand(t *testing.T) {
	dataDir := createTestDBV3Data(t)
	cmd := newDbCMD(&WuKongIMContext{}).CMD()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)
	cmd.SetArgs([]string{"get", "user", "--data-dir", dataDir, "--slot", "all", "--uid", "user-9", "--format", "json"})

	require.NoError(t, cmd.Execute())

	var payload []slotScopedUserPayload
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	require.Len(t, payload, 1)
	require.Equal(t, uint32(9), payload[0].SlotID)
	require.Equal(t, "user-9", payload[0].Data.Uid)
}

func TestDBGetUserAutoSlotFromUIDCommand(t *testing.T) {
	dataDir := createTestDBV3Data(t)
	cmd := newDbCMD(&WuKongIMContext{}).CMD()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)
	cmd.SetArgs([]string{"get", "user", "--data-dir", dataDir, "--slot-count", "64", "--uid", "auto-user-1", "--format", "json"})

	require.NoError(t, cmd.Execute())

	var user wkdb.User
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &user))
	require.Equal(t, "auto-user-1", user.Uid)
}

func TestDBGetChannelAutoSlotFromChannelIDCommand(t *testing.T) {
	dataDir := createTestDBV3Data(t)
	cmd := newDbCMD(&WuKongIMContext{}).CMD()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)
	cmd.SetArgs([]string{"get", "channel", "--data-dir", dataDir, "--slot-count", "64", "--channel-id", "auto-channel-1", "--channel-type", "2", "--format", "json"})

	require.NoError(t, cmd.Execute())

	var channel wkdb.ChannelInfo
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &channel))
	require.Equal(t, "auto-channel-1", channel.ChannelId)
}

func TestDBGetUserAutoSlotMissingSlotCountFails(t *testing.T) {
	oldOpts := serverOpts
	serverOpts = options.New()
	serverOpts.Cluster.SlotCount = 0
	defer func() {
		serverOpts = oldOpts
	}()

	dataDir := createTestDBV3Data(t)
	cmd := newDbCMD(&WuKongIMContext{}).CMD()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)
	cmd.SetArgs([]string{"get", "user", "--data-dir", dataDir, "--uid", "auto-user-1", "--format", "json"})

	err := cmd.Execute()
	require.ErrorContains(t, err, "slot count is required")
}

func TestTopLevelCommandDetectsDBAfterPersistentFlags(t *testing.T) {
	require.Equal(t, "db", topLevelCommand([]string{"--config", "/tmp/test.yaml", "--mode", "release", "db", "tables"}))
	require.Equal(t, "stop", topLevelCommand([]string{"-d", "stop"}))
}

func TestConfigureDBInspectOptionsDoesNotNeedConfigureWithViper(t *testing.T) {
	oldOpts := serverOpts
	serverOpts = options.New()
	defer func() {
		serverOpts = oldOpts
	}()

	vp := viper.New()
	vp.Set("rootDir", "/tmp/wukongim-root")
	vp.Set("cluster.slotCount", 128)
	configureDBInspectOptions(vp)

	require.Equal(t, "/tmp/wukongim-root", serverOpts.RootDir)
	require.Equal(t, "/tmp/wukongim-root/data", serverOpts.DataDir)
	require.Equal(t, 128, serverOpts.Cluster.SlotCount)
}

func TestParseRawDumpOptions(t *testing.T) {
	rawCmd := newDbCMD(&WuKongIMContext{}).newDBRawDumpCmd(&dbCommandOptions{})
	require.NoError(t, rawCmd.ParseFlags([]string{"--channel-type", "2"}))

	opts, err := parseRawDumpOptions(rawCmd, 10, rawDumpFlags{
		tables:     []string{"user,channel-cluster-config"},
		scopes:     []string{"primary", "aux"},
		decode:     true,
		keyPrefix:  "0x0102",
		tailPrefix: "aabb",
		channelFlags: channelFlags{
			channelType: 2,
		},
	})
	require.NoError(t, err)
	require.Equal(t, 10, opts.Limit)
	require.Equal(t, true, opts.DecodeValues)
	require.Len(t, opts.Tables, 2)
	require.Len(t, opts.Scopes, 2)
	require.NotNil(t, opts.KeyFilter.ChannelType)
	require.Equal(t, uint8(2), *opts.KeyFilter.ChannelType)
	require.Equal(t, []byte{0x01, 0x02}, opts.KeyFilter.KeyPrefix)
	require.Equal(t, []byte{0xaa, 0xbb}, opts.KeyFilter.TailPrefix)

	rawCmdNoChannelType := newDbCMD(&WuKongIMContext{}).newDBRawDumpCmd(&dbCommandOptions{})
	opts, err = parseRawDumpOptions(rawCmdNoChannelType, 0, rawDumpFlags{
		tables: []string{"all"},
		scopes: []string{"all"},
	})
	require.NoError(t, err)
	require.Nil(t, opts.Tables)
	require.Nil(t, opts.Scopes)
	require.Nil(t, opts.KeyFilter.ChannelType)

	_, err = parseRawDumpOptions(rawCmdNoChannelType, 0, rawDumpFlags{
		keyPrefix: "abc",
	})
	require.ErrorContains(t, err, "hex prefix must have even length")
}

func createTestDBV3Data(t *testing.T) string {
	t.Helper()

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
	defer func() {
		require.NoError(t, db.Close())
	}()

	now := time.Unix(1710000000, 0).UTC()
	scope := db.Slots().Scope(7)
	require.NoError(t, scope.Users().Put(wkdb.User{Id: 11, Uid: "user-1", PluginNo: "plugin-a", CreatedAt: &now, UpdatedAt: &now}))
	require.NoError(t, scope.Devices().Put(wkdb.Device{Id: 21, Uid: "user-1", DeviceFlag: 1, DeviceLevel: 1, Token: "token-1", CreatedAt: &now, UpdatedAt: &now}))
	require.NoError(t, scope.Conversations().Put("user-1", []wkdb.Conversation{{Id: 31, Uid: "user-1", Type: wkdb.ConversationTypeChat, ChannelId: "group-1", ChannelType: 2, ReadToMsgSeq: 88, CreatedAt: &now, UpdatedAt: &now}}))
	require.NoError(t, scope.Channels().Put(wkdb.ChannelInfo{Id: 41, ChannelId: "group-1", ChannelType: 2, CreatedAt: &now, UpdatedAt: &now}))
	require.NoError(t, scope.Subscribers().Put("group-1", 2, []wkdb.Member{{Uid: "user-1", CreatedAt: &now, UpdatedAt: &now}}))
	require.NoError(t, scope.ChannelClusterConfigs().Put(wkdb.ChannelClusterConfig{Id: 51, ChannelId: "group-1", ChannelType: 2, ReplicaMaxCount: 3, Replicas: []uint64{1001, 1002, 1003}, LeaderId: 1001, ConfVersion: 9, CreatedAt: &now, UpdatedAt: &now}))
	require.NoError(t, scope.MessageEvents().PutState(wkdb.MessageEventState{ChannelId: "group-1", ChannelType: 2, ClientMsgNo: "client-1", EventKey: wkdb.EventKeyDefault, LastMsgEventSeq: 3, SnapshotPayload: []byte(`{"kind":"text"}`)}))
	require.NoError(t, scope.MessageEvents().SetSeq("group-1", 2, "client-1", 9))

	scope9 := db.Slots().Scope(9)
	require.NoError(t, scope9.Users().Put(wkdb.User{Id: 19, Uid: "user-9", PluginNo: "plugin-z", CreatedAt: &now, UpdatedAt: &now}))

	autoUserSlot := wkutil.GetSlotNum(64, "auto-user-1")
	scopeAutoUser := db.Slots().Scope(autoUserSlot)
	require.NoError(t, scopeAutoUser.Users().Put(wkdb.User{Id: 29, Uid: "auto-user-1", PluginNo: "plugin-auto", CreatedAt: &now, UpdatedAt: &now}))

	autoChannelSlot := wkutil.GetSlotNum(64, "auto-channel-1")
	scopeAutoChannel := db.Slots().Scope(autoChannelSlot)
	require.NoError(t, scopeAutoChannel.Channels().Put(wkdb.ChannelInfo{Id: 49, ChannelId: "auto-channel-1", ChannelType: 2, CreatedAt: &now, UpdatedAt: &now}))

	return dataDir
}
