package key

func EncodeConversationRowKey(slotID uint32, uid, channelID string, channelType uint8) []byte {
	key := newSlotKey(slotID, ScopeSlotPrimary, TableConversation, KindRow)
	key = appendString(key, uid)
	key = appendString(key, channelID)
	return append(key, channelType)
}

func ConversationUIDPrefix(slotID uint32, uid string) []byte {
	key := newSlotKey(slotID, ScopeSlotPrimary, TableConversation, KindRow)
	return appendString(key, uid)
}

func ConversationUIDRange(slotID uint32, uid string) (lower, upper []byte) {
	lower = ConversationUIDPrefix(slotID, uid)
	return lower, PrefixUpperBound(lower)
}

func EncodeConversationUpdatedAtSecondIndexKey(slotID uint32, uid string, updatedAt uint64, channelID string, channelType uint8) []byte {
	key := newSlotKey(slotID, ScopeSlotSecondIdx, TableConversation, KindSecondIndex)
	key = appendString(key, uid)
	key = appendUint64(key, updatedAt)
	key = appendString(key, channelID)
	return append(key, channelType)
}

func ParseConversationRowKey(key []byte) (slotID uint32, uid, channelID string, channelType uint8, err error) {
	slotID, off, err := parseSlotKeyHeader(key, ScopeSlotPrimary, TableConversation, KindRow)
	if err != nil {
		return 0, "", "", 0, err
	}
	uid, off, err = readString(key, off)
	if err != nil {
		return 0, "", "", 0, err
	}
	channelID, off, err = readString(key, off)
	if err != nil {
		return 0, "", "", 0, err
	}
	if len(key) <= off {
		return 0, "", "", 0, errShortBuffer("channel_type")
	}
	channelType = key[off]
	return
}

func ParseConversationUpdatedAtSecondIndexKey(key []byte) (slotID uint32, uid string, updatedAt uint64, channelID string, channelType uint8, err error) {
	slotID, off, err := parseSlotKeyHeader(key, ScopeSlotSecondIdx, TableConversation, KindSecondIndex)
	if err != nil {
		return 0, "", 0, "", 0, err
	}
	uid, off, err = readString(key, off)
	if err != nil {
		return 0, "", 0, "", 0, err
	}
	updatedAt, off, err = readUint64(key, off)
	if err != nil {
		return 0, "", 0, "", 0, err
	}
	channelID, off, err = readString(key, off)
	if err != nil {
		return 0, "", 0, "", 0, err
	}
	if len(key) <= off {
		return 0, "", 0, "", 0, errShortBuffer("channel_type")
	}
	channelType = key[off]
	return
}
