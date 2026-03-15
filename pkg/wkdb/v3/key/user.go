package key

func EncodeUserRowKey(slotID uint32, uid string) []byte {
	key := newSlotKey(slotID, ScopeSlotPrimary, TableUser, KindRow)
	return appendString(key, uid)
}

func EncodeUserCreatedAtSecondIndexKey(slotID uint32, createdAt uint64, uid string) []byte {
	key := newSlotKey(slotID, ScopeSlotSecondIdx, TableUser, KindSecondIndex)
	key = appendUint64(key, createdAt)
	return appendString(key, uid)
}

func ParseUserRowKey(key []byte) (slotID uint32, uid string, err error) {
	slotID, off, err := parseSlotKeyHeader(key, ScopeSlotPrimary, TableUser, KindRow)
	if err != nil {
		return 0, "", err
	}
	uid, _, err = readString(key, off)
	return
}

func ParseUserCreatedAtSecondIndexKey(key []byte) (slotID uint32, createdAt uint64, uid string, err error) {
	slotID, off, err := parseSlotKeyHeader(key, ScopeSlotSecondIdx, TableUser, KindSecondIndex)
	if err != nil {
		return 0, 0, "", err
	}
	createdAt, off, err = readUint64(key, off)
	if err != nil {
		return 0, 0, "", err
	}
	uid, _, err = readString(key, off)
	return
}
