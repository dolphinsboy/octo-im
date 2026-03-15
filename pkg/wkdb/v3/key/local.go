package key

func EncodeNotifyQueueRowKey(messageID uint64) []byte {
	key := newMetaKey(ScopeLocal, TableMessageNotifyQueue, KindRow)
	return appendUint64(key, messageID)
}

func ParseNotifyQueueRowKey(key []byte) (messageID uint64, err error) {
	off, err := parseMetaKeyHeader(key, ScopeLocal, TableMessageNotifyQueue, KindRow)
	if err != nil {
		return 0, err
	}
	messageID, _, err = readUint64(key, off)
	return
}
