package key

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConversationRowKeyRoundTrip(t *testing.T) {
	key := EncodeConversationRowKey(11, "user-1", "group-100", 2)

	slotID, uid, channelID, channelType, err := ParseConversationRowKey(key)
	require.NoError(t, err)
	require.Equal(t, uint32(11), slotID)
	require.Equal(t, "user-1", uid)
	require.Equal(t, "group-100", channelID)
	require.Equal(t, uint8(2), channelType)
}

func TestConversationKeysAreSlotSeparated(t *testing.T) {
	k1 := EncodeConversationRowKey(11, "user-1", "group-100", 2)
	k2 := EncodeConversationRowKey(12, "user-1", "group-100", 2)
	require.True(t, bytes.Compare(k1, k2) < 0)
}

func TestConversationUIDRangeContainsRowsForUID(t *testing.T) {
	lower, upper := ConversationUIDRange(11, "user-1")
	rowKey := EncodeConversationRowKey(11, "user-1", "group-100", 2)
	otherUID := EncodeConversationRowKey(11, "user-2", "group-100", 2)

	require.True(t, bytes.Compare(lower, rowKey) <= 0)
	require.True(t, bytes.Compare(rowKey, upper) < 0)
	require.True(t, bytes.Compare(otherUID, upper) >= 0)
}

func TestChannelInfoRowKeyRoundTrip(t *testing.T) {
	key := EncodeChannelInfoRowKey(22, "group-100", 2)

	slotID, channelID, channelType, err := ParseChannelInfoRowKey(key)
	require.NoError(t, err)
	require.Equal(t, uint32(22), slotID)
	require.Equal(t, "group-100", channelID)
	require.Equal(t, uint8(2), channelType)
}

func TestMessageEventStateRowKeyRoundTrip(t *testing.T) {
	key := EncodeMessageEventStateRowKey(22, "group-100", 2, "client-1", "main")

	slotID, channelID, channelType, clientMsgNo, eventKey, err := ParseMessageEventStateRowKey(key)
	require.NoError(t, err)
	require.Equal(t, uint32(22), slotID)
	require.Equal(t, "group-100", channelID)
	require.Equal(t, uint8(2), channelType)
	require.Equal(t, "client-1", clientMsgNo)
	require.Equal(t, "main", eventKey)
}
