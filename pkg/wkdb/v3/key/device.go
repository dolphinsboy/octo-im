package key

func EncodeDeviceRowKey(slotID uint32, uid string, deviceFlag uint64) []byte {
	key := newSlotKey(slotID, ScopeSlotPrimary, TableDevice, KindRow)
	key = appendString(key, uid)
	return appendUint64(key, deviceFlag)
}

func DeviceUIDPrefix(slotID uint32, uid string) []byte {
	key := newSlotKey(slotID, ScopeSlotPrimary, TableDevice, KindRow)
	return appendString(key, uid)
}

func DeviceUIDRange(slotID uint32, uid string) (lower, upper []byte) {
	lower = DeviceUIDPrefix(slotID, uid)
	return lower, PrefixUpperBound(lower)
}

func EncodeDeviceCreatedAtSecondIndexKey(slotID uint32, createdAt uint64, uid string, deviceFlag uint64) []byte {
	key := newSlotKey(slotID, ScopeSlotSecondIdx, TableDevice, KindSecondIndex)
	key = appendUint64(key, createdAt)
	key = appendString(key, uid)
	return appendUint64(key, deviceFlag)
}

func ParseDeviceRowKey(key []byte) (slotID uint32, uid string, deviceFlag uint64, err error) {
	slotID, off, err := parseSlotKeyHeader(key, ScopeSlotPrimary, TableDevice, KindRow)
	if err != nil {
		return 0, "", 0, err
	}
	uid, off, err = readString(key, off)
	if err != nil {
		return 0, "", 0, err
	}
	deviceFlag, _, err = readUint64(key, off)
	return
}

func ParseDeviceCreatedAtSecondIndexKey(key []byte) (slotID uint32, createdAt uint64, uid string, deviceFlag uint64, err error) {
	slotID, off, err := parseSlotKeyHeader(key, ScopeSlotSecondIdx, TableDevice, KindSecondIndex)
	if err != nil {
		return 0, 0, "", 0, err
	}
	createdAt, off, err = readUint64(key, off)
	if err != nil {
		return 0, 0, "", 0, err
	}
	uid, off, err = readString(key, off)
	if err != nil {
		return 0, 0, "", 0, err
	}
	deviceFlag, _, err = readUint64(key, off)
	return
}
