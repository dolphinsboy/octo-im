package key

func EncodeChannelInfoRowKey(slotID uint32, channelID string, channelType uint8) []byte {
	key := newSlotKey(slotID, ScopeSlotPrimary, TableChannelInfo, KindRow)
	key = appendString(key, channelID)
	return append(key, channelType)
}

func EncodeChannelInfoCreatedAtSecondIndexKey(slotID uint32, createdAt uint64, channelID string, channelType uint8) []byte {
	key := newSlotKey(slotID, ScopeSlotSecondIdx, TableChannelInfo, KindSecondIndex)
	key = appendUint64(key, createdAt)
	key = appendString(key, channelID)
	return append(key, channelType)
}

func ParseChannelInfoRowKey(key []byte) (slotID uint32, channelID string, channelType uint8, err error) {
	slotID, off, err := parseSlotKeyHeader(key, ScopeSlotPrimary, TableChannelInfo, KindRow)
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

func ParseChannelInfoCreatedAtSecondIndexKey(key []byte) (slotID uint32, createdAt uint64, channelID string, channelType uint8, err error) {
	slotID, off, err := parseSlotKeyHeader(key, ScopeSlotSecondIdx, TableChannelInfo, KindSecondIndex)
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
