package key

type ScopeType byte

const (
	ScopeSlotPrimary   ScopeType = 0x10
	ScopeSlotIndex     ScopeType = 0x11
	ScopeSlotSecondIdx ScopeType = 0x12
	ScopeSlotAux       ScopeType = 0x13

	ScopeMetaPrimary   ScopeType = 0x20
	ScopeMetaIndex     ScopeType = 0x21
	ScopeMetaSecondIdx ScopeType = 0x22

	ScopeLocal ScopeType = 0x30
)

func (s ScopeType) IsSlot() bool {
	return s >= ScopeSlotPrimary && s <= ScopeSlotAux
}

func (s ScopeType) IsMeta() bool {
	return s >= ScopeMetaPrimary && s <= ScopeMetaSecondIdx
}

func (s ScopeType) String() string {
	switch s {
	case ScopeSlotPrimary:
		return "slot_primary"
	case ScopeSlotIndex:
		return "slot_index"
	case ScopeSlotSecondIdx:
		return "slot_second_index"
	case ScopeSlotAux:
		return "slot_aux"
	case ScopeMetaPrimary:
		return "meta_primary"
	case ScopeMetaIndex:
		return "meta_index"
	case ScopeMetaSecondIdx:
		return "meta_second_index"
	case ScopeLocal:
		return "local"
	default:
		return "scope_unknown"
	}
}

type KeyKind byte

const (
	KindRow         KeyKind = 0x01
	KindIndex       KeyKind = 0x02
	KindSecondIndex KeyKind = 0x03
	KindAux         KeyKind = 0x04
)

func (k KeyKind) String() string {
	switch k {
	case KindRow:
		return "row"
	case KindIndex:
		return "index"
	case KindSecondIndex:
		return "second_index"
	case KindAux:
		return "aux"
	default:
		return "kind_unknown"
	}
}

type TableID uint16

const (
	TableUser                 TableID = 0x0201
	TableDevice               TableID = 0x0301
	TableSubscriber           TableID = 0x0401
	TableSubscriberRelation   TableID = 0x0501
	TableChannelInfo          TableID = 0x0601
	TableDenylist             TableID = 0x0701
	TableAllowlist            TableID = 0x0801
	TableConversation         TableID = 0x0901
	TableMessageNotifyQueue   TableID = 0x0A01
	TableChannelClusterConfig TableID = 0x0B01
	TableMessageEventState    TableID = 0x1901
	TableMessageEventSeq      TableID = 0x1A01

	TableSystemUID  TableID = 0x1001
	TableTester     TableID = 0x1401
	TablePlugin     TableID = 0x1501
	TablePluginUser TableID = 0x1601
)

func (t TableID) String() string {
	switch t {
	case TableUser:
		return "user"
	case TableDevice:
		return "device"
	case TableSubscriber:
		return "subscriber"
	case TableSubscriberRelation:
		return "subscriber_relation"
	case TableChannelInfo:
		return "channel_info"
	case TableDenylist:
		return "denylist"
	case TableAllowlist:
		return "allowlist"
	case TableConversation:
		return "conversation"
	case TableMessageNotifyQueue:
		return "message_notify_queue"
	case TableChannelClusterConfig:
		return "channel_cluster_config"
	case TableMessageEventState:
		return "message_event_state"
	case TableMessageEventSeq:
		return "message_event_seq"
	case TableSystemUID:
		return "system_uid"
	case TableTester:
		return "tester"
	case TablePlugin:
		return "plugin"
	case TablePluginUser:
		return "plugin_user"
	default:
		return "table_unknown"
	}
}

type ScopedRange struct {
	Scope ScopeType
	Lower []byte
	Upper []byte
}

type DecodedKey struct {
	Version uint8
	Scope   ScopeType
	SlotID  uint32
	Table   TableID
	Kind    KeyKind
	Tail    []byte
}

const Version uint8 = 1
