package key

import (
	"fmt"
	"strings"
)

func ParseScopeType(value string) (ScopeType, error) {
	switch normalizeLookupName(value) {
	case "primary", "slotprimary":
		return ScopeSlotPrimary, nil
	case "index", "slotindex":
		return ScopeSlotIndex, nil
	case "secondindex", "secondidx", "slotsecondindex", "slotsecondidx":
		return ScopeSlotSecondIdx, nil
	case "aux", "slotaux":
		return ScopeSlotAux, nil
	case "metaprimary":
		return ScopeMetaPrimary, nil
	case "metaindex":
		return ScopeMetaIndex, nil
	case "metasecondindex", "metasecondidx":
		return ScopeMetaSecondIdx, nil
	case "local":
		return ScopeLocal, nil
	default:
		return 0, fmt.Errorf("unknown scope type: %q", strings.TrimSpace(value))
	}
}

func ParseTableID(value string) (TableID, error) {
	switch normalizeLookupName(value) {
	case "user":
		return TableUser, nil
	case "device":
		return TableDevice, nil
	case "subscriber":
		return TableSubscriber, nil
	case "subscriberrelation":
		return TableSubscriberRelation, nil
	case "channel", "channelinfo":
		return TableChannelInfo, nil
	case "denylist":
		return TableDenylist, nil
	case "allowlist":
		return TableAllowlist, nil
	case "conversation":
		return TableConversation, nil
	case "messagenotifyqueue", "notifyqueue":
		return TableMessageNotifyQueue, nil
	case "channelclusterconfig":
		return TableChannelClusterConfig, nil
	case "messageeventstate":
		return TableMessageEventState, nil
	case "messageeventseq":
		return TableMessageEventSeq, nil
	case "systemuid":
		return TableSystemUID, nil
	case "tester":
		return TableTester, nil
	case "plugin":
		return TablePlugin, nil
	case "pluginuser":
		return TablePluginUser, nil
	default:
		return 0, fmt.Errorf("unknown table id: %q", strings.TrimSpace(value))
	}
}

func normalizeLookupName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, " ", "")
	return value
}
