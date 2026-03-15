package key

func EncodeDenylistRowKey(slotID uint32, channelID string, channelType uint8, uid string) []byte {
	return encodeChannelMemberRowKey(TableDenylist, slotID, channelID, channelType, uid)
}

func DenylistChannelPrefix(slotID uint32, channelID string, channelType uint8) []byte {
	return channelMemberPrefix(TableDenylist, slotID, channelID, channelType)
}

func DenylistChannelRange(slotID uint32, channelID string, channelType uint8) (lower, upper []byte) {
	return channelMemberRange(TableDenylist, slotID, channelID, channelType)
}

func ParseDenylistRowKey(key []byte) (slotID uint32, channelID string, channelType uint8, uid string, err error) {
	return parseChannelMemberRowKey(TableDenylist, key)
}
