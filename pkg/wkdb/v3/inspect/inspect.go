package inspect

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
)

var bucketDirPattern = regexp.MustCompile(`^bucket-(\d+)$`)
var errInspectLimitReached = errors.New("inspect: limit reached")

type Options struct {
	DataDir       string
	FS            vfs.FS
	PebbleOptions *pebble.Options
}

type Inspector struct {
	db          *wkdbv3.PebbleDB
	dataDir     string
	bucketCount uint32
}

type TableInfo struct {
	Name        string   `json:"name"`
	Commands    []string `json:"commands"`
	Description string   `json:"description"`
}

type RawKVRecord struct {
	Index      int    `json:"index"`
	SlotID     uint32 `json:"slot_id"`
	BucketID   uint32 `json:"bucket_id"`
	Scope      string `json:"scope"`
	Table      string `json:"table"`
	Kind       string `json:"kind"`
	KeyHex     string `json:"key_hex"`
	TailHex    string `json:"tail_hex"`
	ValueHex   string `json:"value_hex"`
	ValueSize  int    `json:"value_size"`
	KeyDecoded any    `json:"key_decoded,omitempty"`
	Decoded    any    `json:"decoded,omitempty"`
}

type RawKeyFilter struct {
	UID         string
	ChannelID   string
	ChannelType *uint8
	ClientMsgNo string
	EventKey    string
	KeyPrefix   []byte
	TailPrefix  []byte
}

type RawDumpOptions struct {
	Limit        int
	Tables       []v3key.TableID
	Scopes       []v3key.ScopeType
	KeyFilter    RawKeyFilter
	DecodeValues bool
}

var availableTables = []TableInfo{
	{Name: "user", Commands: []string{"get", "list"}, Description: "slot-owned users"},
	{Name: "device", Commands: []string{"list"}, Description: "slot-owned devices"},
	{Name: "conversation", Commands: []string{"list"}, Description: "slot-owned conversations"},
	{Name: "channel", Commands: []string{"get", "list"}, Description: "channel metadata"},
	{Name: "subscriber", Commands: []string{"list"}, Description: "channel subscribers"},
	{Name: "allowlist", Commands: []string{"list"}, Description: "channel allowlist members"},
	{Name: "denylist", Commands: []string{"list"}, Description: "channel denylist members"},
	{Name: "channel-cluster-config", Commands: []string{"get", "list"}, Description: "channel cluster config rows"},
	{Name: "message-event-state", Commands: []string{"get"}, Description: "message event state rows"},
	{Name: "raw", Commands: []string{"dump"}, Description: "raw slot key/value dump"},
}

func Tables() []TableInfo {
	out := make([]TableInfo, 0, len(availableTables))
	for _, info := range availableTables {
		copied := info
		copied.Commands = append([]string(nil), info.Commands...)
		out = append(out, copied)
	}
	return out
}

func Open(opts Options) (*Inspector, error) {
	if strings.TrimSpace(opts.DataDir) == "" {
		return nil, fmt.Errorf("inspect open requires data dir")
	}
	fs := opts.FS
	if fs == nil {
		fs = vfs.Default
	}
	bucketCount, err := detectBucketCountFS(fs, opts.DataDir)
	if err != nil {
		return nil, err
	}
	router, err := wkdbv3.NewStaticBucketRouter(bucketCount)
	if err != nil {
		return nil, err
	}
	pebbleOptions := clonePebbleOptions(opts.PebbleOptions)
	pebbleOptions.ReadOnly = true
	pebbleOptions.FS = fs

	db, err := wkdbv3.NewPebbleDB(wkdbv3.PebbleDBOptions{
		DataDir:       opts.DataDir,
		Router:        router,
		PebbleOptions: pebbleOptions,
		WriteOptions:  pebble.NoSync,
		FS:            fs,
		ReadOnly:      true,
		ClearSlotCachesFn: func(slotID uint32) {
		},
		RebuildDerivedState: func(ctx context.Context, slotID uint32) error {
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	if err := db.Open(); err != nil {
		return nil, err
	}
	return &Inspector{db: db, dataDir: opts.DataDir, bucketCount: bucketCount}, nil
}

func DetectBucketCount(dataDir string) (uint32, error) {
	return detectBucketCountFS(vfs.Default, dataDir)
}

func (i *Inspector) Close() error {
	if i == nil || i.db == nil {
		return nil
	}
	return i.db.Close()
}

func (i *Inspector) DataDir() string {
	if i == nil {
		return ""
	}
	return i.dataDir
}

func (i *Inspector) BucketCount() uint32 {
	if i == nil {
		return 0
	}
	return i.bucketCount
}

func (i *Inspector) BucketForSlot(slotID uint32) uint32 {
	if i == nil || i.db == nil {
		return 0
	}
	return i.db.Maintenance().BucketForSlot(slotID)
}

func (i *Inspector) ExistingSlots(ctx context.Context) ([]uint32, error) {
	if i == nil || i.db == nil {
		return nil, nil
	}
	seen := make(map[uint32]struct{})
	for bucketID := uint32(0); bucketID < i.bucketCount; bucketID++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		bucket, err := i.db.BucketByID(bucketID)
		if err != nil {
			return nil, err
		}
		iter := bucket.DB().NewIter(&pebble.IterOptions{})
		for iter.First(); iter.Valid(); iter.Next() {
			if err := ctx.Err(); err != nil {
				iter.Close()
				return nil, err
			}
			decoded, err := v3key.Decode(iter.Key())
			if err != nil {
				iter.Close()
				return nil, err
			}
			if !decoded.Scope.IsSlot() {
				continue
			}
			seen[decoded.SlotID] = struct{}{}
		}
		if err := iter.Error(); err != nil {
			iter.Close()
			return nil, err
		}
		if err := iter.Close(); err != nil {
			return nil, err
		}
	}
	slots := make([]uint32, 0, len(seen))
	for slotID := range seen {
		slots = append(slots, slotID)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	return slots, nil
}

func (i *Inspector) GetUser(slotID uint32, uid string) (wkdb.User, error) {
	return i.db.Slots().Scope(slotID).Users().Get(uid)
}

func (i *Inspector) ListUsers(slotID uint32, limit int) ([]wkdb.User, error) {
	return i.db.Slots().Scope(slotID).Users().Search(wkdb.UserSearchReq{Limit: normalizeLimit(limit)})
}

func (i *Inspector) ListDevices(slotID uint32, uid string, limit int) ([]wkdb.Device, error) {
	return i.db.Slots().Scope(slotID).Devices().Search(wkdb.DeviceSearchReq{
		Uid:   strings.TrimSpace(uid),
		Limit: normalizeLimit(limit),
	})
}

func (i *Inspector) ListConversations(slotID uint32, uid string, limit int) ([]wkdb.Conversation, error) {
	return i.db.Slots().Scope(slotID).Conversations().Search(wkdb.ConversationSearchReq{
		Uid:         strings.TrimSpace(uid),
		CurrentPage: 1,
		Limit:       normalizeLimit(limit),
	})
}

func (i *Inspector) GetChannel(slotID uint32, channelID string, channelType uint8) (wkdb.ChannelInfo, error) {
	return i.db.Slots().Scope(slotID).Channels().Get(channelID, channelType)
}

func (i *Inspector) ListChannels(slotID uint32, limit int) ([]wkdb.ChannelInfo, error) {
	return i.db.Slots().Scope(slotID).Channels().Search(wkdb.ChannelSearchReq{Limit: normalizeLimit(limit)})
}

func (i *Inspector) ListSubscribers(slotID uint32, channelID string, channelType uint8) ([]wkdb.Member, error) {
	return i.db.Slots().Scope(slotID).Subscribers().List(channelID, channelType)
}

func (i *Inspector) ListAllowlists(slotID uint32, channelID string, channelType uint8) ([]wkdb.Member, error) {
	return i.db.Slots().Scope(slotID).Allowlists().List(channelID, channelType)
}

func (i *Inspector) ListDenylists(slotID uint32, channelID string, channelType uint8) ([]wkdb.Member, error) {
	return i.db.Slots().Scope(slotID).Denylists().List(channelID, channelType)
}

func (i *Inspector) GetChannelClusterConfig(slotID uint32, channelID string, channelType uint8) (wkdb.ChannelClusterConfig, error) {
	return i.db.Slots().Scope(slotID).ChannelClusterConfigs().Get(channelID, channelType)
}

func (i *Inspector) ListChannelClusterConfigs(slotID uint32, limit int) ([]wkdb.ChannelClusterConfig, error) {
	return i.db.Slots().Scope(slotID).ChannelClusterConfigs().Search(wkdb.ChannelClusterConfigSearchReq{Limit: normalizeLimit(limit)})
}

func (i *Inspector) GetMessageEventState(slotID uint32, channelID string, channelType uint8, clientMsgNo, eventKey string) (*wkdb.MessageEventState, error) {
	return i.db.Slots().Scope(slotID).MessageEvents().GetState(channelID, channelType, clientMsgNo, eventKey)
}

func (i *Inspector) RawDumpSlot(ctx context.Context, slotID uint32, limit int) ([]RawKVRecord, error) {
	return i.RawDumpSlotWithOptions(ctx, slotID, RawDumpOptions{Limit: limit})
}

func (i *Inspector) RawDumpSlotWithOptions(ctx context.Context, slotID uint32, opts RawDumpOptions) ([]RawKVRecord, error) {
	bucketID := i.BucketForSlot(slotID)
	records := make([]RawKVRecord, 0)
	limit := normalizeLimit(opts.Limit)
	tableFilter := make(map[v3key.TableID]struct{}, len(opts.Tables))
	for _, table := range opts.Tables {
		tableFilter[table] = struct{}{}
	}
	scopeFilter := make(map[v3key.ScopeType]struct{}, len(opts.Scopes))
	for _, scope := range opts.Scopes {
		scopeFilter[scope] = struct{}{}
	}
	err := i.db.Slots().Scope(slotID).Raw().Range(ctx, func(key, value []byte) error {
		if limit > 0 && len(records) >= limit {
			return errInspectLimitReached
		}
		decoded, err := v3key.Decode(key)
		if err != nil {
			return err
		}
		if len(tableFilter) > 0 {
			if _, ok := tableFilter[decoded.Table]; !ok {
				return nil
			}
		}
		if len(scopeFilter) > 0 {
			if _, ok := scopeFilter[decoded.Scope]; !ok {
				return nil
			}
		}
		if len(opts.KeyFilter.KeyPrefix) > 0 && !bytes.HasPrefix(key, opts.KeyFilter.KeyPrefix) {
			return nil
		}
		if len(opts.KeyFilter.TailPrefix) > 0 && !bytes.HasPrefix(decoded.Tail, opts.KeyFilter.TailPrefix) {
			return nil
		}
		keyInfo, err := wkdbv3.DescribeSlotRawKey(decoded, key)
		if err != nil {
			return err
		}
		if !matchRawKeyFilter(keyInfo, opts.KeyFilter) {
			return nil
		}
		var decodedValue any
		if opts.DecodeValues {
			decodedValue, err = wkdbv3.DecodeSlotRawValue(decoded, value)
			if err != nil {
				return err
			}
		}
		records = append(records, RawKVRecord{
			Index:      len(records),
			SlotID:     decoded.SlotID,
			BucketID:   bucketID,
			Scope:      decoded.Scope.String(),
			Table:      decoded.Table.String(),
			Kind:       decoded.Kind.String(),
			KeyHex:     hex.EncodeToString(key),
			TailHex:    hex.EncodeToString(decoded.Tail),
			ValueHex:   hex.EncodeToString(value),
			ValueSize:  len(value),
			KeyDecoded: keyInfo.Decoded,
			Decoded:    decodedValue,
		})
		return nil
	})
	if errors.Is(err, errInspectLimitReached) {
		return records, nil
	}
	return records, err
}

func matchRawKeyFilter(info wkdbv3.SlotRawKeyInfo, filter RawKeyFilter) bool {
	if strings.TrimSpace(filter.UID) != "" && info.UID != strings.TrimSpace(filter.UID) {
		return false
	}
	if strings.TrimSpace(filter.ChannelID) != "" && info.ChannelID != strings.TrimSpace(filter.ChannelID) {
		return false
	}
	if filter.ChannelType != nil {
		if info.ChannelType == nil || *info.ChannelType != *filter.ChannelType {
			return false
		}
	}
	if strings.TrimSpace(filter.ClientMsgNo) != "" && info.ClientMsgNo != strings.TrimSpace(filter.ClientMsgNo) {
		return false
	}
	if strings.TrimSpace(filter.EventKey) != "" && info.EventKey != strings.TrimSpace(filter.EventKey) {
		return false
	}
	return true
}

func normalizeLimit(limit int) int {
	if limit < 0 {
		return 0
	}
	return limit
}

func clonePebbleOptions(opts *pebble.Options) *pebble.Options {
	if opts == nil {
		return &pebble.Options{}
	}
	cloned := *opts
	return &cloned
}

func detectBucketCountFS(fs vfs.FS, dataDir string) (uint32, error) {
	if strings.TrimSpace(dataDir) == "" {
		return 0, fmt.Errorf("detect bucket count requires data dir")
	}
	bucketRoot := fs.PathJoin(dataDir, "buckets")
	entries, err := fs.List(bucketRoot)
	if err != nil {
		return 0, err
	}
	ids := make([]uint32, 0)
	for _, entry := range entries {
		match := bucketDirPattern.FindStringSubmatch(entry)
		if len(match) != 2 {
			continue
		}
		id, err := strconv.ParseUint(match[1], 10, 32)
		if err != nil {
			return 0, fmt.Errorf("parse bucket id %q: %w", entry, err)
		}
		info, err := fs.Stat(fs.PathJoin(bucketRoot, entry))
		if err != nil {
			return 0, err
		}
		if !info.IsDir() {
			continue
		}
		ids = append(ids, uint32(id))
	}
	if len(ids) == 0 {
		return 0, fmt.Errorf("no bucket directories found under %s", bucketRoot)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for expected, actual := range ids {
		if uint32(expected) != actual {
			return 0, fmt.Errorf("bucket directories are not contiguous under %s", bucketRoot)
		}
	}
	return uint32(len(ids)), nil
}
