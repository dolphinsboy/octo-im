package v3

import (
	"encoding/binary"
	"fmt"

	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
)

type RawConversationRowRef struct {
	SlotID      uint32 `json:"slot_id"`
	UID         string `json:"uid"`
	ChannelID   string `json:"channel_id"`
	ChannelType uint8  `json:"channel_type"`
}

func DecodeSlotRawValue(meta v3key.DecodedKey, value []byte) (any, error) {
	if !meta.Scope.IsSlot() {
		return nil, fmt.Errorf("decode slot raw value requires slot scope, got %s", meta.Scope.String())
	}
	switch meta.Kind {
	case v3key.KindRow:
		return decodeSlotRowValue(meta, value)
	case v3key.KindSecondIndex:
		return decodeSlotSecondIndexValue(meta, value)
	case v3key.KindAux:
		return decodeSlotAuxValue(meta, value)
	default:
		return nil, nil
	}
}

func decodeSlotRowValue(meta v3key.DecodedKey, value []byte) (any, error) {
	switch meta.Table {
	case v3key.TableUser:
		return decodeUserValue(value)
	case v3key.TableDevice:
		return decodeDeviceValue(value)
	case v3key.TableConversation:
		return decodeConversationValue(value)
	case v3key.TableChannelInfo:
		return decodeChannelInfoValue(value)
	case v3key.TableSubscriber, v3key.TableAllowlist, v3key.TableDenylist:
		return decodeMemberValue(value)
	case v3key.TableChannelClusterConfig:
		return decodeChannelClusterConfigValue(value)
	case v3key.TableMessageEventState:
		return decodeMessageEventState(value)
	default:
		return nil, nil
	}
}

func decodeSlotSecondIndexValue(meta v3key.DecodedKey, value []byte) (any, error) {
	switch meta.Table {
	case v3key.TableUser, v3key.TableDevice:
		return string(value), nil
	case v3key.TableConversation:
		slotID, uid, channelID, channelType, err := v3key.ParseConversationRowKey(value)
		if err != nil {
			return nil, err
		}
		return RawConversationRowRef{
			SlotID:      slotID,
			UID:         uid,
			ChannelID:   channelID,
			ChannelType: channelType,
		}, nil
	case v3key.TableChannelInfo, v3key.TableChannelClusterConfig:
		if len(value) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("unexpected non-empty second index value for %s", meta.Table.String())
	default:
		return nil, nil
	}
}

func decodeSlotAuxValue(meta v3key.DecodedKey, value []byte) (any, error) {
	switch meta.Table {
	case v3key.TableMessageEventSeq:
		if len(value) != 8 {
			return nil, fmt.Errorf("message event seq value size mismatch: %d", len(value))
		}
		return binary.BigEndian.Uint64(value), nil
	default:
		return nil, nil
	}
}
