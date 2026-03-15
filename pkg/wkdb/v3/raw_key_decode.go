package v3

import (
	"fmt"
	"time"

	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
)

type SlotRawKeyInfo struct {
	Decoded     any
	UID         string
	ChannelID   string
	ChannelType *uint8
	ClientMsgNo string
	EventKey    string
}

type RawUserRowKey struct {
	SlotID uint32 `json:"slot_id"`
	UID    string `json:"uid"`
}

type RawUserCreatedAtSecondIndexKey struct {
	SlotID    uint32    `json:"slot_id"`
	CreatedAt time.Time `json:"created_at"`
	UID       string    `json:"uid"`
}

type RawDeviceRowKey struct {
	SlotID     uint32 `json:"slot_id"`
	UID        string `json:"uid"`
	DeviceFlag uint64 `json:"device_flag"`
}

type RawDeviceCreatedAtSecondIndexKey struct {
	SlotID     uint32    `json:"slot_id"`
	CreatedAt  time.Time `json:"created_at"`
	UID        string    `json:"uid"`
	DeviceFlag uint64    `json:"device_flag"`
}

type RawConversationRowKey struct {
	SlotID      uint32 `json:"slot_id"`
	UID         string `json:"uid"`
	ChannelID   string `json:"channel_id"`
	ChannelType uint8  `json:"channel_type"`
}

type RawConversationUpdatedAtSecondIndexKey struct {
	SlotID      uint32    `json:"slot_id"`
	UID         string    `json:"uid"`
	UpdatedAt   time.Time `json:"updated_at"`
	ChannelID   string    `json:"channel_id"`
	ChannelType uint8     `json:"channel_type"`
}

type RawChannelRowKey struct {
	SlotID      uint32 `json:"slot_id"`
	ChannelID   string `json:"channel_id"`
	ChannelType uint8  `json:"channel_type"`
}

type RawChannelCreatedAtSecondIndexKey struct {
	SlotID      uint32    `json:"slot_id"`
	CreatedAt   time.Time `json:"created_at"`
	ChannelID   string    `json:"channel_id"`
	ChannelType uint8     `json:"channel_type"`
}

type RawMemberRowKey struct {
	SlotID      uint32 `json:"slot_id"`
	ChannelID   string `json:"channel_id"`
	ChannelType uint8  `json:"channel_type"`
	UID         string `json:"uid"`
}

type RawMessageEventStateRowKey struct {
	SlotID      uint32 `json:"slot_id"`
	ChannelID   string `json:"channel_id"`
	ChannelType uint8  `json:"channel_type"`
	ClientMsgNo string `json:"client_msg_no"`
	EventKey    string `json:"event_key"`
}

type RawMessageEventSeqAuxKey struct {
	SlotID      uint32 `json:"slot_id"`
	ChannelID   string `json:"channel_id"`
	ChannelType uint8  `json:"channel_type"`
	ClientMsgNo string `json:"client_msg_no"`
}

func DescribeSlotRawKey(meta v3key.DecodedKey, key []byte) (SlotRawKeyInfo, error) {
	if !meta.Scope.IsSlot() {
		return SlotRawKeyInfo{}, fmt.Errorf("describe slot raw key requires slot scope, got %s", meta.Scope.String())
	}
	switch meta.Kind {
	case v3key.KindRow:
		return describeSlotRowKey(meta, key)
	case v3key.KindSecondIndex:
		return describeSlotSecondIndexKey(meta, key)
	case v3key.KindAux:
		return describeSlotAuxKey(meta, key)
	default:
		return SlotRawKeyInfo{}, nil
	}
}

func describeSlotRowKey(meta v3key.DecodedKey, key []byte) (SlotRawKeyInfo, error) {
	switch meta.Table {
	case v3key.TableUser:
		slotID, uid, err := v3key.ParseUserRowKey(key)
		if err != nil {
			return SlotRawKeyInfo{}, err
		}
		return SlotRawKeyInfo{
			Decoded: RawUserRowKey{SlotID: slotID, UID: uid},
			UID:     uid,
		}, nil
	case v3key.TableDevice:
		slotID, uid, deviceFlag, err := v3key.ParseDeviceRowKey(key)
		if err != nil {
			return SlotRawKeyInfo{}, err
		}
		return SlotRawKeyInfo{
			Decoded: RawDeviceRowKey{SlotID: slotID, UID: uid, DeviceFlag: deviceFlag},
			UID:     uid,
		}, nil
	case v3key.TableConversation:
		slotID, uid, channelID, channelType, err := v3key.ParseConversationRowKey(key)
		if err != nil {
			return SlotRawKeyInfo{}, err
		}
		return SlotRawKeyInfo{
			Decoded:     RawConversationRowKey{SlotID: slotID, UID: uid, ChannelID: channelID, ChannelType: channelType},
			UID:         uid,
			ChannelID:   channelID,
			ChannelType: uint8Ptr(channelType),
		}, nil
	case v3key.TableChannelInfo:
		slotID, channelID, channelType, err := v3key.ParseChannelInfoRowKey(key)
		if err != nil {
			return SlotRawKeyInfo{}, err
		}
		return SlotRawKeyInfo{
			Decoded:     RawChannelRowKey{SlotID: slotID, ChannelID: channelID, ChannelType: channelType},
			ChannelID:   channelID,
			ChannelType: uint8Ptr(channelType),
		}, nil
	case v3key.TableSubscriber:
		return describeMemberRowKey(v3key.ParseSubscriberRowKey(key))
	case v3key.TableAllowlist:
		return describeMemberRowKey(v3key.ParseAllowlistRowKey(key))
	case v3key.TableDenylist:
		return describeMemberRowKey(v3key.ParseDenylistRowKey(key))
	case v3key.TableChannelClusterConfig:
		slotID, channelID, channelType, err := v3key.ParseChannelClusterConfigRowKey(key)
		if err != nil {
			return SlotRawKeyInfo{}, err
		}
		return SlotRawKeyInfo{
			Decoded:     RawChannelRowKey{SlotID: slotID, ChannelID: channelID, ChannelType: channelType},
			ChannelID:   channelID,
			ChannelType: uint8Ptr(channelType),
		}, nil
	case v3key.TableMessageEventState:
		slotID, channelID, channelType, clientMsgNo, eventKey, err := v3key.ParseMessageEventStateRowKey(key)
		if err != nil {
			return SlotRawKeyInfo{}, err
		}
		return SlotRawKeyInfo{
			Decoded: RawMessageEventStateRowKey{
				SlotID:      slotID,
				ChannelID:   channelID,
				ChannelType: channelType,
				ClientMsgNo: clientMsgNo,
				EventKey:    eventKey,
			},
			ChannelID:   channelID,
			ChannelType: uint8Ptr(channelType),
			ClientMsgNo: clientMsgNo,
			EventKey:    eventKey,
		}, nil
	default:
		return SlotRawKeyInfo{}, nil
	}
}

func describeSlotSecondIndexKey(meta v3key.DecodedKey, key []byte) (SlotRawKeyInfo, error) {
	switch meta.Table {
	case v3key.TableUser:
		slotID, createdAt, uid, err := v3key.ParseUserCreatedAtSecondIndexKey(key)
		if err != nil {
			return SlotRawKeyInfo{}, err
		}
		return SlotRawKeyInfo{
			Decoded: RawUserCreatedAtSecondIndexKey{
				SlotID:    slotID,
				CreatedAt: time.Unix(0, int64(createdAt)).UTC(),
				UID:       uid,
			},
			UID: uid,
		}, nil
	case v3key.TableDevice:
		slotID, createdAt, uid, deviceFlag, err := v3key.ParseDeviceCreatedAtSecondIndexKey(key)
		if err != nil {
			return SlotRawKeyInfo{}, err
		}
		return SlotRawKeyInfo{
			Decoded: RawDeviceCreatedAtSecondIndexKey{
				SlotID:     slotID,
				CreatedAt:  time.Unix(0, int64(createdAt)).UTC(),
				UID:        uid,
				DeviceFlag: deviceFlag,
			},
			UID: uid,
		}, nil
	case v3key.TableConversation:
		slotID, uid, updatedAt, channelID, channelType, err := v3key.ParseConversationUpdatedAtSecondIndexKey(key)
		if err != nil {
			return SlotRawKeyInfo{}, err
		}
		return SlotRawKeyInfo{
			Decoded: RawConversationUpdatedAtSecondIndexKey{
				SlotID:      slotID,
				UID:         uid,
				UpdatedAt:   time.Unix(0, int64(updatedAt)).UTC(),
				ChannelID:   channelID,
				ChannelType: channelType,
			},
			UID:         uid,
			ChannelID:   channelID,
			ChannelType: uint8Ptr(channelType),
		}, nil
	case v3key.TableChannelInfo:
		slotID, createdAt, channelID, channelType, err := v3key.ParseChannelInfoCreatedAtSecondIndexKey(key)
		if err != nil {
			return SlotRawKeyInfo{}, err
		}
		return SlotRawKeyInfo{
			Decoded: RawChannelCreatedAtSecondIndexKey{
				SlotID:      slotID,
				CreatedAt:   time.Unix(0, int64(createdAt)).UTC(),
				ChannelID:   channelID,
				ChannelType: channelType,
			},
			ChannelID:   channelID,
			ChannelType: uint8Ptr(channelType),
		}, nil
	case v3key.TableChannelClusterConfig:
		slotID, createdAt, channelID, channelType, err := v3key.ParseChannelClusterConfigCreatedAtSecondIndexKey(key)
		if err != nil {
			return SlotRawKeyInfo{}, err
		}
		return SlotRawKeyInfo{
			Decoded: RawChannelCreatedAtSecondIndexKey{
				SlotID:      slotID,
				CreatedAt:   time.Unix(0, int64(createdAt)).UTC(),
				ChannelID:   channelID,
				ChannelType: channelType,
			},
			ChannelID:   channelID,
			ChannelType: uint8Ptr(channelType),
		}, nil
	default:
		return SlotRawKeyInfo{}, nil
	}
}

func describeSlotAuxKey(meta v3key.DecodedKey, key []byte) (SlotRawKeyInfo, error) {
	switch meta.Table {
	case v3key.TableMessageEventSeq:
		slotID, channelID, channelType, clientMsgNo, err := v3key.ParseMessageEventSeqAuxKey(key)
		if err != nil {
			return SlotRawKeyInfo{}, err
		}
		return SlotRawKeyInfo{
			Decoded: RawMessageEventSeqAuxKey{
				SlotID:      slotID,
				ChannelID:   channelID,
				ChannelType: channelType,
				ClientMsgNo: clientMsgNo,
			},
			ChannelID:   channelID,
			ChannelType: uint8Ptr(channelType),
			ClientMsgNo: clientMsgNo,
		}, nil
	default:
		return SlotRawKeyInfo{}, nil
	}
}

func describeMemberRowKey(slotID uint32, channelID string, channelType uint8, uid string, err error) (SlotRawKeyInfo, error) {
	if err != nil {
		return SlotRawKeyInfo{}, err
	}
	return SlotRawKeyInfo{
		Decoded: RawMemberRowKey{
			SlotID:      slotID,
			ChannelID:   channelID,
			ChannelType: channelType,
			UID:         uid,
		},
		UID:         uid,
		ChannelID:   channelID,
		ChannelType: uint8Ptr(channelType),
	}, nil
}

func uint8Ptr(v uint8) *uint8 {
	return &v
}
