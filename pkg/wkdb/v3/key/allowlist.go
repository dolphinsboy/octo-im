package key

func EncodeAllowlistRowKey(slotID uint32, channelID string, channelType uint8, uid string) []byte {
	return encodeChannelMemberRowKey(TableAllowlist, slotID, channelID, channelType, uid)
}

func AllowlistChannelPrefix(slotID uint32, channelID string, channelType uint8) []byte {
	return channelMemberPrefix(TableAllowlist, slotID, channelID, channelType)
}

func AllowlistChannelRange(slotID uint32, channelID string, channelType uint8) (lower, upper []byte) {
	return channelMemberRange(TableAllowlist, slotID, channelID, channelType)
}

func ParseAllowlistRowKey(key []byte) (slotID uint32, channelID string, channelType uint8, uid string, err error) {
	return parseChannelMemberRowKey(TableAllowlist, key)
}
