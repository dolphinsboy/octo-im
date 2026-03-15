package key

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubscriberRowKeyRoundTrip(t *testing.T) {
	key := EncodeSubscriberRowKey(11, "group-1", 2, "user-1")

	slotID, channelID, channelType, uid, err := ParseSubscriberRowKey(key)
	require.NoError(t, err)
	require.Equal(t, uint32(11), slotID)
	require.Equal(t, "group-1", channelID)
	require.Equal(t, uint8(2), channelType)
	require.Equal(t, "user-1", uid)
}

func TestSubscriberChannelRangeContainsRowsForChannel(t *testing.T) {
	lower, upper := SubscriberChannelRange(11, "group-1", 2)
	rowKey := EncodeSubscriberRowKey(11, "group-1", 2, "user-1")
	otherChannel := EncodeSubscriberRowKey(11, "group-2", 2, "user-1")
	otherType := EncodeSubscriberRowKey(11, "group-1", 3, "user-1")

	require.True(t, bytes.Compare(lower, rowKey) <= 0)
	require.True(t, bytes.Compare(rowKey, upper) < 0)
	require.True(t, bytes.Compare(otherChannel, upper) >= 0 || bytes.Compare(otherChannel, lower) < 0)
	require.True(t, bytes.Compare(otherType, upper) >= 0 || bytes.Compare(otherType, lower) < 0)
}

func TestAllowlistAndDenylistRowKeyRoundTrip(t *testing.T) {
	allowKey := EncodeAllowlistRowKey(12, "group-2", 5, "user-a")
	denyKey := EncodeDenylistRowKey(13, "group-3", 6, "user-b")

	slotID, channelID, channelType, uid, err := ParseAllowlistRowKey(allowKey)
	require.NoError(t, err)
	require.Equal(t, uint32(12), slotID)
	require.Equal(t, "group-2", channelID)
	require.Equal(t, uint8(5), channelType)
	require.Equal(t, "user-a", uid)

	slotID, channelID, channelType, uid, err = ParseDenylistRowKey(denyKey)
	require.NoError(t, err)
	require.Equal(t, uint32(13), slotID)
	require.Equal(t, "group-3", channelID)
	require.Equal(t, uint8(6), channelType)
	require.Equal(t, "user-b", uid)
}
