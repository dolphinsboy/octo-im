package key

func encodeChannelMemberRowKey(table TableID, slotID uint32, channelID string, channelType uint8, uid string) []byte {
	key := newSlotKey(slotID, ScopeSlotPrimary, table, KindRow)
	key = appendString(key, channelID)
	key = append(key, channelType)
	return appendString(key, uid)
}

func channelMemberPrefix(table TableID, slotID uint32, channelID string, channelType uint8) []byte {
	key := newSlotKey(slotID, ScopeSlotPrimary, table, KindRow)
	key = appendString(key, channelID)
	return append(key, channelType)
}

func channelMemberRange(table TableID, slotID uint32, channelID string, channelType uint8) (lower, upper []byte) {
	lower = channelMemberPrefix(table, slotID, channelID, channelType)
	return lower, PrefixUpperBound(lower)
}

func parseChannelMemberRowKey(table TableID, key []byte) (slotID uint32, channelID string, channelType uint8, uid string, err error) {
	slotID, off, err := parseSlotKeyHeader(key, ScopeSlotPrimary, table, KindRow)
	if err != nil {
		return 0, "", 0, "", err
	}
	channelID, off, err = readString(key, off)
	if err != nil {
		return 0, "", 0, "", err
	}
	if len(key) <= off {
		return 0, "", 0, "", errShortBuffer("channel_type")
	}
	channelType = key[off]
	uid, _, err = readString(key, off+1)
	return
}
