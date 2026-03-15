package key

func EncodeSubscriberRowKey(slotID uint32, channelID string, channelType uint8, uid string) []byte {
	return encodeChannelMemberRowKey(TableSubscriber, slotID, channelID, channelType, uid)
}

func SubscriberChannelPrefix(slotID uint32, channelID string, channelType uint8) []byte {
	return channelMemberPrefix(TableSubscriber, slotID, channelID, channelType)
}

func SubscriberChannelRange(slotID uint32, channelID string, channelType uint8) (lower, upper []byte) {
	return channelMemberRange(TableSubscriber, slotID, channelID, channelType)
}

func ParseSubscriberRowKey(key []byte) (slotID uint32, channelID string, channelType uint8, uid string, err error) {
	return parseChannelMemberRowKey(TableSubscriber, key)
}
