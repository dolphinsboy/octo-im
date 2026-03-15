package key

func EncodeMessageEventStateRowKey(slotID uint32, channelID string, channelType uint8, clientMsgNo, eventKey string) []byte {
	key := newSlotKey(slotID, ScopeSlotPrimary, TableMessageEventState, KindRow)
	key = appendString(key, channelID)
	key = append(key, channelType)
	key = appendString(key, clientMsgNo)
	return appendString(key, eventKey)
}

func MessageEventStatePrefix(slotID uint32, channelID string, channelType uint8, clientMsgNo string) []byte {
	key := newSlotKey(slotID, ScopeSlotPrimary, TableMessageEventState, KindRow)
	key = appendString(key, channelID)
	key = append(key, channelType)
	return appendString(key, clientMsgNo)
}

func MessageEventStateRange(slotID uint32, channelID string, channelType uint8, clientMsgNo string) (lower, upper []byte) {
	lower = MessageEventStatePrefix(slotID, channelID, channelType, clientMsgNo)
	return lower, PrefixUpperBound(lower)
}

func EncodeMessageEventSeqAuxKey(slotID uint32, channelID string, channelType uint8, clientMsgNo string) []byte {
	key := newSlotKey(slotID, ScopeSlotAux, TableMessageEventSeq, KindAux)
	key = appendString(key, channelID)
	key = append(key, channelType)
	return appendString(key, clientMsgNo)
}

func ParseMessageEventSeqAuxKey(key []byte) (slotID uint32, channelID string, channelType uint8, clientMsgNo string, err error) {
	slotID, off, err := parseSlotKeyHeader(key, ScopeSlotAux, TableMessageEventSeq, KindAux)
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
	clientMsgNo, _, err = readString(key, off+1)
	return
}

func ParseMessageEventStateRowKey(key []byte) (slotID uint32, channelID string, channelType uint8, clientMsgNo, eventKey string, err error) {
	slotID, off, err := parseSlotKeyHeader(key, ScopeSlotPrimary, TableMessageEventState, KindRow)
	if err != nil {
		return 0, "", 0, "", "", err
	}
	channelID, off, err = readString(key, off)
	if err != nil {
		return 0, "", 0, "", "", err
	}
	if len(key) <= off {
		return 0, "", 0, "", "", errShortBuffer("channel_type")
	}
	channelType = key[off]
	off++
	clientMsgNo, off, err = readString(key, off)
	if err != nil {
		return 0, "", 0, "", "", err
	}
	eventKey, _, err = readString(key, off)
	return
}
