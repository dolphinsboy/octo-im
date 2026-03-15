package key

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelClusterConfigRowKeyRoundTrip(t *testing.T) {
	key := EncodeChannelClusterConfigRowKey(21, "group-100", 2)

	slotID, channelID, channelType, err := ParseChannelClusterConfigRowKey(key)
	require.NoError(t, err)
	require.Equal(t, uint32(21), slotID)
	require.Equal(t, "group-100", channelID)
	require.Equal(t, uint8(2), channelType)
}

func TestMessageEventStateRangeContainsRowsForClientMsgNo(t *testing.T) {
	lower, upper := MessageEventStateRange(7, "group-1", 2, "client-1")
	rowKey := EncodeMessageEventStateRowKey(7, "group-1", 2, "client-1", "main")
	otherClient := EncodeMessageEventStateRowKey(7, "group-1", 2, "client-2", "main")

	require.True(t, bytes.Compare(lower, rowKey) <= 0)
	require.True(t, bytes.Compare(rowKey, upper) < 0)
	require.True(t, bytes.Compare(otherClient, upper) >= 0)
}

func TestMessageEventSeqAuxKeyRoundTrip(t *testing.T) {
	key := EncodeMessageEventSeqAuxKey(9, "group-9", 4, "client-9")

	slotID, channelID, channelType, clientMsgNo, err := ParseMessageEventSeqAuxKey(key)
	require.NoError(t, err)
	require.Equal(t, uint32(9), slotID)
	require.Equal(t, "group-9", channelID)
	require.Equal(t, uint8(4), channelType)
	require.Equal(t, "client-9", clientMsgNo)
}
