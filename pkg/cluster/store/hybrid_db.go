package store

import (
	"fmt"

	"github.com/WuKongIM/WuKongIM/pkg/raft/types"
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
)

// HybridDB keeps legacy wkdb for unmigrated capabilities while routing slot-owned state to wkdb/v3.
type HybridDB struct {
	wkdb.DB

	slotDB    wkdbv3.DB
	slotCount uint32
	routeSlot func(key string) uint32
}

var _ wkdb.DB = (*HybridDB)(nil)
var _ SlotSnapshotBackend = (*HybridDB)(nil)

func NewHybridDB(legacy wkdb.DB, slotDB wkdbv3.DB, slotCount uint32, routeSlot func(key string) uint32) *HybridDB {
	return &HybridDB{
		DB:        legacy,
		slotDB:    slotDB,
		slotCount: slotCount,
		routeSlot: routeSlot,
	}
}

func (h *HybridDB) Open() error {
	if h.DB == nil {
		return fmt.Errorf("legacy wkdb is required")
	}
	if err := h.DB.Open(); err != nil {
		return err
	}
	if h.slotDB == nil {
		return nil
	}
	if err := h.slotDB.Open(); err != nil {
		_ = h.DB.Close()
		return err
	}
	return nil
}

func (h *HybridDB) Close() error {
	var err error
	if h.slotDB != nil {
		err = h.slotDB.Close()
	}
	if h.DB != nil {
		if closeErr := h.DB.Close(); closeErr != nil {
			if err != nil {
				return fmt.Errorf("close slot db: %v; close legacy db: %w", err, closeErr)
			}
			return closeErr
		}
	}
	return err
}

func (h *HybridDB) Snapshotter() wkdbv3.SlotSnapshotter {
	if h.slotDB == nil {
		return nil
	}
	return h.slotDB.Snapshotter()
}

func (h *HybridDB) Maintenance() wkdbv3.Maintenance {
	if h.slotDB == nil {
		return nil
	}
	return h.slotDB.Maintenance()
}

func (h *HybridDB) hasSlotDB() bool {
	return h.slotDB != nil && h.routeSlot != nil && h.slotCount > 0
}

func (h *HybridDB) hasV3DB() bool {
	return h.slotDB != nil
}

func (h *HybridDB) slotIDForKey(key string) uint32 {
	if !h.hasSlotDB() {
		return 0
	}
	return h.routeSlot(key)
}

func (h *HybridDB) slotScopeByKey(key string) wkdbv3.SlotScope {
	return h.slotDB.Slots().Scope(h.slotIDForKey(key))
}

func (h *HybridDB) slotScope(slotID uint32) wkdbv3.SlotScope {
	return h.slotDB.Slots().Scope(slotID)
}

func (h *HybridDB) eachSlot(fn func(slotID uint32, scope wkdbv3.SlotScope) error) error {
	if !h.hasSlotDB() {
		return fmt.Errorf("slot db is not configured")
	}
	for slotID := uint32(0); slotID < h.slotCount; slotID++ {
		if err := fn(slotID, h.slotScope(slotID)); err != nil {
			return err
		}
	}
	return nil
}

func (h *HybridDB) AddUser(u wkdb.User) error {
	if !h.hasSlotDB() {
		return h.DB.AddUser(u)
	}
	return h.slotScopeByKey(u.Uid).Users().Put(u)
}

func (h *HybridDB) UpdateUser(u wkdb.User) error {
	if !h.hasSlotDB() {
		return h.DB.UpdateUser(u)
	}
	return h.slotScopeByKey(u.Uid).Users().Put(u)
}

func (h *HybridDB) GetUser(uid string) (wkdb.User, error) {
	if !h.hasSlotDB() {
		return h.DB.GetUser(uid)
	}
	return h.slotScopeByKey(uid).Users().Get(uid)
}

func (h *HybridDB) ExistUser(uid string) (bool, error) {
	if !h.hasSlotDB() {
		return h.DB.ExistUser(uid)
	}
	return h.slotScopeByKey(uid).Users().Exists(uid)
}

func (h *HybridDB) AddDevice(device wkdb.Device) error {
	if !h.hasSlotDB() {
		return h.DB.AddDevice(device)
	}
	return h.slotScopeByKey(device.Uid).Devices().Put(device)
}

func (h *HybridDB) UpdateDevice(device wkdb.Device) error {
	if !h.hasSlotDB() {
		return h.DB.UpdateDevice(device)
	}
	return h.slotScopeByKey(device.Uid).Devices().Put(device)
}

func (h *HybridDB) GetDevice(uid string, deviceFlag uint64) (wkdb.Device, error) {
	if !h.hasSlotDB() {
		return h.DB.GetDevice(uid, deviceFlag)
	}
	return h.slotScopeByKey(uid).Devices().Get(uid, deviceFlag)
}

func (h *HybridDB) GetDevices(uid string) ([]wkdb.Device, error) {
	if !h.hasSlotDB() {
		return h.DB.GetDevices(uid)
	}
	return h.slotScopeByKey(uid).Devices().ListByUID(uid)
}

func (h *HybridDB) GetDeviceCount(uid string) (int, error) {
	if !h.hasSlotDB() {
		return h.DB.GetDeviceCount(uid)
	}
	devices, err := h.GetDevices(uid)
	if err != nil {
		return 0, err
	}
	return len(devices), nil
}

func (h *HybridDB) AddSubscribers(channelID string, channelType uint8, members []wkdb.Member) error {
	if !h.hasSlotDB() {
		return h.DB.AddSubscribers(channelID, channelType, members)
	}
	return h.slotScopeByKey(channelID).Subscribers().Put(channelID, channelType, members)
}

func (h *HybridDB) RemoveSubscribers(channelID string, channelType uint8, uids []string) error {
	if !h.hasSlotDB() {
		return h.DB.RemoveSubscribers(channelID, channelType, uids)
	}
	return h.slotScopeByKey(channelID).Subscribers().Delete(channelID, channelType, uids)
}

func (h *HybridDB) ExistSubscriber(channelID string, channelType uint8, uid string) (bool, error) {
	if !h.hasSlotDB() {
		return h.DB.ExistSubscriber(channelID, channelType, uid)
	}
	members, err := h.GetSubscribers(channelID, channelType)
	if err != nil {
		return false, err
	}
	for _, member := range members {
		if member.Uid == uid {
			return true, nil
		}
	}
	return false, nil
}

func (h *HybridDB) RemoveAllSubscriber(channelID string, channelType uint8) error {
	if !h.hasSlotDB() {
		return h.DB.RemoveAllSubscriber(channelID, channelType)
	}
	return h.slotScopeByKey(channelID).Subscribers().DeleteAll(channelID, channelType)
}

func (h *HybridDB) GetSubscribers(channelID string, channelType uint8) ([]wkdb.Member, error) {
	if !h.hasSlotDB() {
		return h.DB.GetSubscribers(channelID, channelType)
	}
	return h.slotScopeByKey(channelID).Subscribers().List(channelID, channelType)
}

func (h *HybridDB) GetSubscriberCount(channelID string, channelType uint8) (int, error) {
	if !h.hasSlotDB() {
		return h.DB.GetSubscriberCount(channelID, channelType)
	}
	members, err := h.GetSubscribers(channelID, channelType)
	if err != nil {
		return 0, err
	}
	return len(members), nil
}

func (h *HybridDB) AddChannel(info wkdb.ChannelInfo) (uint64, error) {
	if !h.hasSlotDB() {
		return h.DB.AddChannel(info)
	}
	if err := h.slotScopeByKey(info.ChannelId).Channels().Put(info); err != nil {
		return 0, err
	}
	return info.Id, nil
}

func (h *HybridDB) UpdateChannel(info wkdb.ChannelInfo) error {
	if !h.hasSlotDB() {
		return h.DB.UpdateChannel(info)
	}
	return h.slotScopeByKey(info.ChannelId).Channels().Put(info)
}

func (h *HybridDB) GetChannel(channelID string, channelType uint8) (wkdb.ChannelInfo, error) {
	if !h.hasSlotDB() {
		return h.DB.GetChannel(channelID, channelType)
	}
	return h.slotScopeByKey(channelID).Channels().Get(channelID, channelType)
}

func (h *HybridDB) ExistChannel(channelID string, channelType uint8) (bool, error) {
	if !h.hasSlotDB() {
		return h.DB.ExistChannel(channelID, channelType)
	}
	return h.slotScopeByKey(channelID).Channels().Exists(channelID, channelType)
}

func (h *HybridDB) DeleteChannel(channelID string, channelType uint8) error {
	if !h.hasSlotDB() {
		return h.DB.DeleteChannel(channelID, channelType)
	}
	return h.slotScopeByKey(channelID).Channels().Delete(channelID, channelType)
}

func (h *HybridDB) AddDenylist(channelID string, channelType uint8, members []wkdb.Member) error {
	if !h.hasSlotDB() {
		return h.DB.AddDenylist(channelID, channelType, members)
	}
	return h.slotScopeByKey(channelID).Denylists().Put(channelID, channelType, members)
}

func (h *HybridDB) GetDenylist(channelID string, channelType uint8) ([]wkdb.Member, error) {
	if !h.hasSlotDB() {
		return h.DB.GetDenylist(channelID, channelType)
	}
	return h.slotScopeByKey(channelID).Denylists().List(channelID, channelType)
}

func (h *HybridDB) RemoveDenylist(channelID string, channelType uint8, uids []string) error {
	if !h.hasSlotDB() {
		return h.DB.RemoveDenylist(channelID, channelType, uids)
	}
	return h.slotScopeByKey(channelID).Denylists().Delete(channelID, channelType, uids)
}

func (h *HybridDB) RemoveAllDenylist(channelID string, channelType uint8) error {
	if !h.hasSlotDB() {
		return h.DB.RemoveAllDenylist(channelID, channelType)
	}
	return h.slotScopeByKey(channelID).Denylists().DeleteAll(channelID, channelType)
}

func (h *HybridDB) ExistDenylist(channelID string, channelType uint8, uid string) (bool, error) {
	if !h.hasSlotDB() {
		return h.DB.ExistDenylist(channelID, channelType, uid)
	}
	members, err := h.GetDenylist(channelID, channelType)
	if err != nil {
		return false, err
	}
	for _, member := range members {
		if member.Uid == uid {
			return true, nil
		}
	}
	return false, nil
}

func (h *HybridDB) AddAllowlist(channelID string, channelType uint8, members []wkdb.Member) error {
	if !h.hasSlotDB() {
		return h.DB.AddAllowlist(channelID, channelType, members)
	}
	return h.slotScopeByKey(channelID).Allowlists().Put(channelID, channelType, members)
}

func (h *HybridDB) GetAllowlist(channelID string, channelType uint8) ([]wkdb.Member, error) {
	if !h.hasSlotDB() {
		return h.DB.GetAllowlist(channelID, channelType)
	}
	return h.slotScopeByKey(channelID).Allowlists().List(channelID, channelType)
}

func (h *HybridDB) RemoveAllowlist(channelID string, channelType uint8, uids []string) error {
	if !h.hasSlotDB() {
		return h.DB.RemoveAllowlist(channelID, channelType, uids)
	}
	return h.slotScopeByKey(channelID).Allowlists().Delete(channelID, channelType, uids)
}

func (h *HybridDB) RemoveAllAllowlist(channelID string, channelType uint8) error {
	if !h.hasSlotDB() {
		return h.DB.RemoveAllAllowlist(channelID, channelType)
	}
	return h.slotScopeByKey(channelID).Allowlists().DeleteAll(channelID, channelType)
}

func (h *HybridDB) ExistAllowlist(channelID string, channelType uint8, uid string) (bool, error) {
	if !h.hasSlotDB() {
		return h.DB.ExistAllowlist(channelID, channelType, uid)
	}
	members, err := h.GetAllowlist(channelID, channelType)
	if err != nil {
		return false, err
	}
	for _, member := range members {
		if member.Uid == uid {
			return true, nil
		}
	}
	return false, nil
}

func (h *HybridDB) HasAllowlist(channelID string, channelType uint8) (bool, error) {
	if !h.hasSlotDB() {
		return h.DB.HasAllowlist(channelID, channelType)
	}
	members, err := h.GetAllowlist(channelID, channelType)
	if err != nil {
		return false, err
	}
	return len(members) > 0, nil
}

func (h *HybridDB) AddOrUpdateConversations(conversations []wkdb.Conversation) error {
	if !h.hasSlotDB() {
		return h.DB.AddOrUpdateConversations(conversations)
	}
	grouped := make(map[string][]wkdb.Conversation)
	for _, conversation := range conversations {
		if conversation.Id == 0 {
			conversation.Id = h.NextPrimaryKey()
		}
		grouped[conversation.Uid] = append(grouped[conversation.Uid], conversation)
	}
	for uid, items := range grouped {
		if err := h.slotScopeByKey(uid).Conversations().Put(uid, items); err != nil {
			return err
		}
	}
	return nil
}

func (h *HybridDB) AddOrUpdateConversationsBatchIfNotExist(conversations []wkdb.Conversation) error {
	if !h.hasSlotDB() {
		return h.DB.AddOrUpdateConversationsBatchIfNotExist(conversations)
	}
	grouped := make(map[string][]wkdb.Conversation)
	for _, conversation := range conversations {
		exists, err := h.ExistConversation(conversation.Uid, conversation.ChannelId, conversation.ChannelType)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if conversation.Id == 0 {
			conversation.Id = h.NextPrimaryKey()
		}
		grouped[conversation.Uid] = append(grouped[conversation.Uid], conversation)
	}
	for uid, items := range grouped {
		if err := h.slotScopeByKey(uid).Conversations().Put(uid, items); err != nil {
			return err
		}
	}
	return nil
}

func (h *HybridDB) AddOrUpdateConversationsWithUser(uid string, conversations []wkdb.Conversation) error {
	if !h.hasSlotDB() {
		return h.DB.AddOrUpdateConversationsWithUser(uid, conversations)
	}
	for i := range conversations {
		if conversations[i].Id == 0 {
			conversations[i].Id = h.NextPrimaryKey()
		}
	}
	return h.slotScopeByKey(uid).Conversations().Put(uid, conversations)
}

func (h *HybridDB) DeleteConversation(uid string, channelID string, channelType uint8) error {
	if !h.hasSlotDB() {
		return h.DB.DeleteConversation(uid, channelID, channelType)
	}
	return h.slotScopeByKey(uid).Conversations().Delete(uid, channelID, channelType)
}

func (h *HybridDB) DeleteConversations(uid string, channels []wkdb.Channel) error {
	if !h.hasSlotDB() {
		return h.DB.DeleteConversations(uid, channels)
	}
	return h.slotScopeByKey(uid).Conversations().DeleteBatch(uid, channels)
}

func (h *HybridDB) UpdateConversationDeletedAtMsgSeq(uid string, channelID string, channelType uint8, seq uint64) error {
	if !h.hasSlotDB() {
		return h.DB.UpdateConversationDeletedAtMsgSeq(uid, channelID, channelType, seq)
	}
	return h.slotScopeByKey(uid).Conversations().UpdateDeletedAt(uid, channelID, channelType, seq)
}

func (h *HybridDB) UpdateConversationIfSeqGreater(uid string, channelID string, channelType uint8, seq uint64) error {
	if !h.hasSlotDB() {
		return h.DB.UpdateConversationIfSeqGreater(uid, channelID, channelType, seq)
	}
	return h.slotScopeByKey(uid).Conversations().UpdateIfSeqGreater(uid, channelID, channelType, seq)
}

func (h *HybridDB) GetConversations(uid string) ([]wkdb.Conversation, error) {
	if !h.hasSlotDB() {
		return h.DB.GetConversations(uid)
	}
	return h.slotScopeByKey(uid).Conversations().ListByUID(uid)
}

func (h *HybridDB) GetConversationsByType(uid string, tp wkdb.ConversationType) ([]wkdb.Conversation, error) {
	if !h.hasSlotDB() {
		return h.DB.GetConversationsByType(uid, tp)
	}
	conversations, err := h.GetConversations(uid)
	if err != nil {
		return nil, err
	}
	filtered := make([]wkdb.Conversation, 0, len(conversations))
	for _, conversation := range conversations {
		if conversation.Type != tp {
			continue
		}
		filtered = append(filtered, cloneConversation(conversation))
	}
	return filtered, nil
}

func (h *HybridDB) GetConversation(uid string, channelID string, channelType uint8) (wkdb.Conversation, error) {
	if !h.hasSlotDB() {
		return h.DB.GetConversation(uid, channelID, channelType)
	}
	return h.slotScopeByKey(uid).Conversations().Get(uid, channelID, channelType)
}

func (h *HybridDB) GetLastConversations(uid string, tp wkdb.ConversationType, updatedAt uint64, excludeChannelTypes []uint8, limit int) ([]wkdb.Conversation, error) {
	if !h.hasSlotDB() {
		return h.DB.GetLastConversations(uid, tp, updatedAt, excludeChannelTypes, limit)
	}
	conversations, err := h.GetConversations(uid)
	if err != nil {
		return nil, err
	}
	excluded := make(map[uint8]struct{}, len(excludeChannelTypes))
	for _, channelType := range excludeChannelTypes {
		excluded[channelType] = struct{}{}
	}
	filtered := make([]wkdb.Conversation, 0, len(conversations))
	for _, conversation := range conversations {
		if conversation.Type != tp {
			continue
		}
		if _, ok := excluded[conversation.ChannelType]; ok {
			continue
		}
		if updatedAt > 0 && uint64(conversationSortTime(conversation)) >= updatedAt {
			continue
		}
		filtered = append(filtered, conversation)
	}
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	out := make([]wkdb.Conversation, 0, len(filtered))
	for _, conversation := range filtered {
		out = append(out, cloneConversation(conversation))
	}
	return out, nil
}

func (h *HybridDB) ExistConversation(uid string, channelID string, channelType uint8) (bool, error) {
	if !h.hasSlotDB() {
		return h.DB.ExistConversation(uid, channelID, channelType)
	}
	_, err := h.slotScopeByKey(uid).Conversations().Get(uid, channelID, channelType)
	if err == nil {
		return true, nil
	}
	if err == wkdb.ErrNotFound {
		return false, nil
	}
	return false, err
}

func (h *HybridDB) SaveChannelClusterConfigs(cfgs []wkdb.ChannelClusterConfig) error {
	if !h.hasSlotDB() {
		return h.DB.SaveChannelClusterConfigs(cfgs)
	}
	for _, cfg := range cfgs {
		if err := h.slotScopeByKey(cfg.ChannelId).ChannelClusterConfigs().Put(cfg); err != nil {
			return err
		}
	}
	return nil
}

func (h *HybridDB) GetChannelClusterConfig(channelID string, channelType uint8) (wkdb.ChannelClusterConfig, error) {
	if !h.hasSlotDB() {
		return h.DB.GetChannelClusterConfig(channelID, channelType)
	}
	return h.slotScopeByKey(channelID).ChannelClusterConfigs().Get(channelID, channelType)
}

func (h *HybridDB) GetChannelClusterConfigWithSlotId(slotID uint32) ([]wkdb.ChannelClusterConfig, error) {
	if !h.hasSlotDB() {
		return h.DB.GetChannelClusterConfigWithSlotId(slotID)
	}
	return h.slotScope(slotID).ChannelClusterConfigs().Search(wkdb.ChannelClusterConfigSearchReq{Limit: 0})
}

func (h *HybridDB) GetChannelClusterConfigVersion(channelID string, channelType uint8) (uint64, error) {
	if !h.hasSlotDB() {
		return h.DB.GetChannelClusterConfigVersion(channelID, channelType)
	}
	cfg, err := h.GetChannelClusterConfig(channelID, channelType)
	if err != nil {
		return 0, err
	}
	return cfg.ConfVersion, nil
}

func (h *HybridDB) AddSystemUids(uids []string) error {
	if !h.hasV3DB() {
		return h.DB.AddSystemUids(uids)
	}
	return h.slotDB.Meta().SystemUIDs().PutAll(uids)
}

func (h *HybridDB) RemoveSystemUids(uids []string) error {
	if !h.hasV3DB() {
		return h.DB.RemoveSystemUids(uids)
	}
	return h.slotDB.Meta().SystemUIDs().Delete(uids)
}

func (h *HybridDB) GetSystemUids() ([]string, error) {
	if !h.hasV3DB() {
		return h.DB.GetSystemUids()
	}
	return h.slotDB.Meta().SystemUIDs().GetAll()
}

func (h *HybridDB) AddOrUpdateTester(tester wkdb.Tester) error {
	if !h.hasV3DB() {
		return h.DB.AddOrUpdateTester(tester)
	}
	return h.slotDB.Meta().Testers().Put(tester)
}

func (h *HybridDB) GetTester(no string) (wkdb.Tester, error) {
	if !h.hasV3DB() {
		return h.DB.GetTester(no)
	}
	tester, err := h.slotDB.Meta().Testers().Get(no)
	if err == wkdb.ErrNotFound {
		return wkdb.Tester{}, nil
	}
	return tester, err
}

func (h *HybridDB) GetTesters() ([]wkdb.Tester, error) {
	if !h.hasV3DB() {
		return h.DB.GetTesters()
	}
	return h.slotDB.Meta().Testers().List()
}

func (h *HybridDB) RemoveTester(no string) error {
	if !h.hasV3DB() {
		return h.DB.RemoveTester(no)
	}
	return h.slotDB.Meta().Testers().Delete(no)
}

func (h *HybridDB) AddOrUpdatePlugin(plugin wkdb.Plugin) error {
	if !h.hasV3DB() {
		return h.DB.AddOrUpdatePlugin(plugin)
	}
	return h.slotDB.Meta().Plugins().Put(plugin)
}

func (h *HybridDB) DeletePlugin(no string) error {
	if !h.hasV3DB() {
		return h.DB.DeletePlugin(no)
	}
	return h.slotDB.Meta().Plugins().Delete(no)
}

func (h *HybridDB) GetPlugins() ([]wkdb.Plugin, error) {
	if !h.hasV3DB() {
		return h.DB.GetPlugins()
	}
	return h.slotDB.Meta().Plugins().List()
}

func (h *HybridDB) GetPlugin(no string) (wkdb.Plugin, error) {
	if !h.hasV3DB() {
		return h.DB.GetPlugin(no)
	}
	plugin, err := h.slotDB.Meta().Plugins().Get(no)
	if err == wkdb.ErrNotFound {
		return wkdb.Plugin{}, nil
	}
	return plugin, err
}

func (h *HybridDB) AddOrUpdatePluginUsers(pluginUsers []wkdb.PluginUser) error {
	if !h.hasV3DB() {
		return h.DB.AddOrUpdatePluginUsers(pluginUsers)
	}
	return h.slotDB.Meta().PluginUsers().Put(pluginUsers)
}

func (h *HybridDB) RemovePluginUser(pluginNo string, uid string) error {
	if !h.hasV3DB() {
		return h.DB.RemovePluginUser(pluginNo, uid)
	}
	return h.slotDB.Meta().PluginUsers().Delete(pluginNo, uid)
}

func (h *HybridDB) GetPluginUsers(pluginNo string) ([]wkdb.PluginUser, error) {
	if !h.hasV3DB() {
		return h.DB.GetPluginUsers(pluginNo)
	}
	return h.slotDB.Meta().PluginUsers().ListByPlugin(pluginNo)
}

func (h *HybridDB) GetHighestPriorityPluginByUid(uid string) (string, error) {
	if !h.hasV3DB() {
		return h.DB.GetHighestPriorityPluginByUid(uid)
	}
	pluginUsers, err := h.slotDB.Meta().PluginUsers().Search(wkdb.SearchPluginUserReq{Uid: uid})
	if err != nil {
		return "", err
	}
	var (
		bestPlugin  wkdb.Plugin
		maxPriority uint32
	)
	for _, pluginUser := range pluginUsers {
		plugin, err := h.slotDB.Meta().Plugins().Get(pluginUser.PluginNo)
		if err != nil {
			if err == wkdb.ErrNotFound {
				continue
			}
			return "", err
		}
		if bestPlugin.No == "" || plugin.Priority > maxPriority {
			bestPlugin = plugin
			maxPriority = plugin.Priority
		}
	}
	return bestPlugin.No, nil
}

func (h *HybridDB) ExistPluginByUid(uid string) (bool, error) {
	if !h.hasV3DB() {
		return h.DB.ExistPluginByUid(uid)
	}
	pluginUsers, err := h.slotDB.Meta().PluginUsers().Search(wkdb.SearchPluginUserReq{Uid: uid})
	if err != nil {
		return false, err
	}
	return len(pluginUsers) > 0, nil
}

func (h *HybridDB) UpdatePluginConfig(no string, config map[string]interface{}) error {
	if !h.hasV3DB() {
		return h.DB.UpdatePluginConfig(no, config)
	}
	return h.slotDB.Meta().Plugins().UpdateConfig(no, config)
}

func (h *HybridDB) GetPluginConfig(no string) (map[string]interface{}, error) {
	if !h.hasV3DB() {
		return h.DB.GetPluginConfig(no)
	}
	plugin, err := h.slotDB.Meta().Plugins().Get(no)
	if err != nil {
		return nil, err
	}
	return clonePluginConfig(plugin.Config), nil
}

func (h *HybridDB) SearchPluginUsers(req wkdb.SearchPluginUserReq) ([]wkdb.PluginUser, error) {
	if !h.hasV3DB() {
		return h.DB.SearchPluginUsers(req)
	}
	return h.slotDB.Meta().PluginUsers().Search(req)
}

func (h *HybridDB) AppendMessageOfNotifyQueue(messages []wkdb.Message) error {
	if !h.hasV3DB() {
		return h.DB.AppendMessageOfNotifyQueue(messages)
	}
	return h.slotDB.Local().NotifyQueue().Put(messages)
}

func (h *HybridDB) GetMessagesOfNotifyQueue(count int) ([]wkdb.Message, error) {
	if !h.hasV3DB() {
		return h.DB.GetMessagesOfNotifyQueue(count)
	}
	return h.slotDB.Local().NotifyQueue().List(count)
}

func (h *HybridDB) RemoveMessagesOfNotifyQueue(messageIDs []int64) error {
	if !h.hasV3DB() {
		return h.DB.RemoveMessagesOfNotifyQueue(messageIDs)
	}
	return h.slotDB.Local().NotifyQueue().Delete(messageIDs)
}

func (h *HybridDB) RemoveMessagesOfNotifyQueueCount(count int) error {
	if !h.hasV3DB() {
		return h.DB.RemoveMessagesOfNotifyQueueCount(count)
	}
	return h.slotDB.Local().NotifyQueue().DeleteCount(count)
}

func clonePluginConfig(config map[string]interface{}) map[string]interface{} {
	if len(config) == 0 {
		return nil
	}
	cloned := make(map[string]interface{}, len(config))
	for key, value := range config {
		cloned[key] = value
	}
	return cloned
}

func slotSnapshotNotSupported(op string) error {
	return fmt.Errorf("slot state db does not support %s: %w", op, types.ErrSnapshotNotSupported)
}
