package key

func EncodeChannelClusterConfigRowKey(slotID uint32, channelID string, channelType uint8) []byte {
	key := newSlotKey(slotID, ScopeSlotPrimary, TableChannelClusterConfig, KindRow)
	key = appendString(key, channelID)
	return append(key, channelType)
}

func EncodeChannelClusterConfigCreatedAtSecondIndexKey(slotID uint32, createdAt uint64, channelID string, channelType uint8) []byte {
	key := newSlotKey(slotID, ScopeSlotSecondIdx, TableChannelClusterConfig, KindSecondIndex)
	key = appendUint64(key, createdAt)
	key = appendString(key, channelID)
	return append(key, channelType)
}

func ParseChannelClusterConfigRowKey(key []byte) (slotID uint32, channelID string, channelType uint8, err error) {
	slotID, off, err := parseSlotKeyHeader(key, ScopeSlotPrimary, TableChannelClusterConfig, KindRow)
	if err != nil {
		return 0, "", 0, err
	}
	channelID, off, err = readString(key, off)
	if err != nil {
		return 0, "", 0, err
	}
	if len(key) <= off {
		return 0, "", 0, errShortBuffer("channel_type")
	}
	channelType = key[off]
	return
}

func ParseChannelClusterConfigCreatedAtSecondIndexKey(key []byte) (slotID uint32, createdAt uint64, channelID string, channelType uint8, err error) {
	slotID, off, err := parseSlotKeyHeader(key, ScopeSlotSecondIdx, TableChannelClusterConfig, KindSecondIndex)
	if err != nil {
		return 0, 0, "", 0, err
	}
	createdAt, off, err = readUint64(key, off)
	if err != nil {
		return 0, 0, "", 0, err
	}
	channelID, off, err = readString(key, off)
	if err != nil {
		return 0, 0, "", 0, err
	}
	if len(key) <= off {
		return 0, 0, "", 0, errShortBuffer("channel_type")
	}
	channelType = key[off]
	return
}
