package v3

import "github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"

type unsupportedMetaDB struct{}

func (unsupportedMetaDB) SystemUIDs() SystemUIDStore { return unsupportedSystemUIDStore{} }
func (unsupportedMetaDB) Testers() TesterStore       { return unsupportedTesterStore{} }
func (unsupportedMetaDB) Plugins() PluginStore       { return unsupportedPluginStore{} }
func (unsupportedMetaDB) PluginUsers() PluginUserStore {
	return unsupportedPluginUserStore{}
}

type unsupportedLocalDB struct{}

func (unsupportedLocalDB) NotifyQueue() NotifyQueueStore { return unsupportedNotifyQueueStore{} }

type unsupportedUserStore struct{}

func (unsupportedUserStore) Get(uid string) (wkdb.User, error) {
	return wkdb.User{}, errNotImplemented("slot.users.get")
}
func (unsupportedUserStore) Exists(uid string) (bool, error) {
	return false, errNotImplemented("slot.users.exists")
}
func (unsupportedUserStore) Put(u wkdb.User) error   { return errNotImplemented("slot.users.put") }
func (unsupportedUserStore) Delete(uid string) error { return errNotImplemented("slot.users.delete") }
func (unsupportedUserStore) Search(req wkdb.UserSearchReq) ([]wkdb.User, error) {
	return nil, errNotImplemented("slot.users.search")
}

type unsupportedDeviceStore struct{}

func (unsupportedDeviceStore) Get(uid string, deviceFlag uint64) (wkdb.Device, error) {
	return wkdb.Device{}, errNotImplemented("slot.devices.get")
}
func (unsupportedDeviceStore) ListByUID(uid string) ([]wkdb.Device, error) {
	return nil, errNotImplemented("slot.devices.list_by_uid")
}
func (unsupportedDeviceStore) Put(d wkdb.Device) error { return errNotImplemented("slot.devices.put") }
func (unsupportedDeviceStore) Delete(uid string, deviceFlag uint64) error {
	return errNotImplemented("slot.devices.delete")
}
func (unsupportedDeviceStore) Search(req wkdb.DeviceSearchReq) ([]wkdb.Device, error) {
	return nil, errNotImplemented("slot.devices.search")
}

type unsupportedConversationStore struct{}

func (unsupportedConversationStore) Get(uid, channelID string, channelType uint8) (wkdb.Conversation, error) {
	return wkdb.Conversation{}, errNotImplemented("slot.conversations.get")
}
func (unsupportedConversationStore) ListByUID(uid string) ([]wkdb.Conversation, error) {
	return nil, errNotImplemented("slot.conversations.list_by_uid")
}
func (unsupportedConversationStore) Put(uid string, cs []wkdb.Conversation) error {
	return errNotImplemented("slot.conversations.put")
}
func (unsupportedConversationStore) Delete(uid, channelID string, channelType uint8) error {
	return errNotImplemented("slot.conversations.delete")
}
func (unsupportedConversationStore) DeleteBatch(uid string, channels []wkdb.Channel) error {
	return errNotImplemented("slot.conversations.delete_batch")
}
func (unsupportedConversationStore) UpdateIfSeqGreater(uid, channelID string, channelType uint8, seq uint64) error {
	return errNotImplemented("slot.conversations.update_if_seq_greater")
}
func (unsupportedConversationStore) UpdateDeletedAt(uid, channelID string, channelType uint8, seq uint64) error {
	return errNotImplemented("slot.conversations.update_deleted_at")
}
func (unsupportedConversationStore) Search(req wkdb.ConversationSearchReq) ([]wkdb.Conversation, error) {
	return nil, errNotImplemented("slot.conversations.search")
}

type unsupportedChannelStore struct{}

func (unsupportedChannelStore) Get(channelID string, channelType uint8) (wkdb.ChannelInfo, error) {
	return wkdb.ChannelInfo{}, errNotImplemented("slot.channels.get")
}
func (unsupportedChannelStore) Exists(channelID string, channelType uint8) (bool, error) {
	return false, errNotImplemented("slot.channels.exists")
}
func (unsupportedChannelStore) Put(info wkdb.ChannelInfo) error {
	return errNotImplemented("slot.channels.put")
}
func (unsupportedChannelStore) Delete(channelID string, channelType uint8) error {
	return errNotImplemented("slot.channels.delete")
}
func (unsupportedChannelStore) Search(req wkdb.ChannelSearchReq) ([]wkdb.ChannelInfo, error) {
	return nil, errNotImplemented("slot.channels.search")
}

type unsupportedSubscriberStore struct{}

func (unsupportedSubscriberStore) List(channelID string, channelType uint8) ([]wkdb.Member, error) {
	return nil, errNotImplemented("slot.subscribers.list")
}
func (unsupportedSubscriberStore) Put(channelID string, channelType uint8, members []wkdb.Member) error {
	return errNotImplemented("slot.subscribers.put")
}
func (unsupportedSubscriberStore) Delete(channelID string, channelType uint8, uids []string) error {
	return errNotImplemented("slot.subscribers.delete")
}
func (unsupportedSubscriberStore) DeleteAll(channelID string, channelType uint8) error {
	return errNotImplemented("slot.subscribers.delete_all")
}

type unsupportedAllowlistStore struct{}

func (unsupportedAllowlistStore) List(channelID string, channelType uint8) ([]wkdb.Member, error) {
	return nil, errNotImplemented("slot.allowlists.list")
}
func (unsupportedAllowlistStore) Put(channelID string, channelType uint8, members []wkdb.Member) error {
	return errNotImplemented("slot.allowlists.put")
}
func (unsupportedAllowlistStore) Delete(channelID string, channelType uint8, uids []string) error {
	return errNotImplemented("slot.allowlists.delete")
}
func (unsupportedAllowlistStore) DeleteAll(channelID string, channelType uint8) error {
	return errNotImplemented("slot.allowlists.delete_all")
}

type unsupportedDenylistStore struct{}

func (unsupportedDenylistStore) List(channelID string, channelType uint8) ([]wkdb.Member, error) {
	return nil, errNotImplemented("slot.denylists.list")
}
func (unsupportedDenylistStore) Put(channelID string, channelType uint8, members []wkdb.Member) error {
	return errNotImplemented("slot.denylists.put")
}
func (unsupportedDenylistStore) Delete(channelID string, channelType uint8, uids []string) error {
	return errNotImplemented("slot.denylists.delete")
}
func (unsupportedDenylistStore) DeleteAll(channelID string, channelType uint8) error {
	return errNotImplemented("slot.denylists.delete_all")
}

type unsupportedChannelClusterConfigStore struct{}

func (unsupportedChannelClusterConfigStore) Get(channelID string, channelType uint8) (wkdb.ChannelClusterConfig, error) {
	return wkdb.ChannelClusterConfig{}, errNotImplemented("slot.channel_cluster_configs.get")
}
func (unsupportedChannelClusterConfigStore) Put(cfg wkdb.ChannelClusterConfig) error {
	return errNotImplemented("slot.channel_cluster_configs.put")
}
func (unsupportedChannelClusterConfigStore) Search(req wkdb.ChannelClusterConfigSearchReq) ([]wkdb.ChannelClusterConfig, error) {
	return nil, errNotImplemented("slot.channel_cluster_configs.search")
}

type unsupportedMessageEventStore struct{}

func (unsupportedMessageEventStore) GetState(channelID string, channelType uint8, clientMsgNo, eventKey string) (*wkdb.MessageEventState, error) {
	return nil, errNotImplemented("slot.message_events.get_state")
}
func (unsupportedMessageEventStore) GetStates(channelID string, channelType uint8, clientMsgNo string) ([]wkdb.MessageEventState, error) {
	return nil, errNotImplemented("slot.message_events.get_states")
}
func (unsupportedMessageEventStore) PutState(state wkdb.MessageEventState) error {
	return errNotImplemented("slot.message_events.put_state")
}
func (unsupportedMessageEventStore) DeleteStates(channelID string, channelType uint8, clientMsgNo string) error {
	return errNotImplemented("slot.message_events.delete_states")
}
func (unsupportedMessageEventStore) GetSeq(channelID string, channelType uint8, clientMsgNo string) (uint64, error) {
	return 0, errNotImplemented("slot.message_events.get_seq")
}
func (unsupportedMessageEventStore) SetSeq(channelID string, channelType uint8, clientMsgNo string, seq uint64) error {
	return errNotImplemented("slot.message_events.set_seq")
}

type unsupportedSystemUIDStore struct{}

func (unsupportedSystemUIDStore) GetAll() ([]string, error) {
	return nil, errNotImplemented("meta.system_uids.get_all")
}
func (unsupportedSystemUIDStore) PutAll(uids []string) error {
	return errNotImplemented("meta.system_uids.put_all")
}
func (unsupportedSystemUIDStore) Delete(uids []string) error {
	return errNotImplemented("meta.system_uids.delete")
}

type unsupportedTesterStore struct{}

func (unsupportedTesterStore) Get(no string) (wkdb.Tester, error) {
	return wkdb.Tester{}, errNotImplemented("meta.testers.get")
}
func (unsupportedTesterStore) List() ([]wkdb.Tester, error) {
	return nil, errNotImplemented("meta.testers.list")
}
func (unsupportedTesterStore) Put(tester wkdb.Tester) error {
	return errNotImplemented("meta.testers.put")
}
func (unsupportedTesterStore) Delete(no string) error {
	return errNotImplemented("meta.testers.delete")
}

type unsupportedPluginStore struct{}

func (unsupportedPluginStore) Get(no string) (wkdb.Plugin, error) {
	return wkdb.Plugin{}, errNotImplemented("meta.plugins.get")
}
func (unsupportedPluginStore) List() ([]wkdb.Plugin, error) {
	return nil, errNotImplemented("meta.plugins.list")
}
func (unsupportedPluginStore) Put(plugin wkdb.Plugin) error {
	return errNotImplemented("meta.plugins.put")
}
func (unsupportedPluginStore) Delete(no string) error {
	return errNotImplemented("meta.plugins.delete")
}
func (unsupportedPluginStore) UpdateConfig(no string, config map[string]interface{}) error {
	return errNotImplemented("meta.plugins.update_config")
}

type unsupportedPluginUserStore struct{}

func (unsupportedPluginUserStore) ListByPlugin(pluginNo string) ([]wkdb.PluginUser, error) {
	return nil, errNotImplemented("meta.plugin_users.list_by_plugin")
}
func (unsupportedPluginUserStore) Search(req wkdb.SearchPluginUserReq) ([]wkdb.PluginUser, error) {
	return nil, errNotImplemented("meta.plugin_users.search")
}
func (unsupportedPluginUserStore) Put(pluginUsers []wkdb.PluginUser) error {
	return errNotImplemented("meta.plugin_users.put")
}
func (unsupportedPluginUserStore) Delete(pluginNo, uid string) error {
	return errNotImplemented("meta.plugin_users.delete")
}

type unsupportedNotifyQueueStore struct{}

func (unsupportedNotifyQueueStore) Put(messages []wkdb.Message) error {
	return errNotImplemented("local.notify_queue.put")
}
func (unsupportedNotifyQueueStore) List(count int) ([]wkdb.Message, error) {
	return nil, errNotImplemented("local.notify_queue.list")
}
func (unsupportedNotifyQueueStore) Delete(messageIDs []int64) error {
	return errNotImplemented("local.notify_queue.delete")
}
func (unsupportedNotifyQueueStore) DeleteCount(count int) error {
	return errNotImplemented("local.notify_queue.delete_count")
}
