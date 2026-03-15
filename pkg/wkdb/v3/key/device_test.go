package key

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeviceRowKeyRoundTrip(t *testing.T) {
	key := EncodeDeviceRowKey(11, "user-1", 2)

	slotID, uid, deviceFlag, err := ParseDeviceRowKey(key)
	require.NoError(t, err)
	require.Equal(t, uint32(11), slotID)
	require.Equal(t, "user-1", uid)
	require.Equal(t, uint64(2), deviceFlag)
}

func TestDeviceUIDRangeContainsRowsForUID(t *testing.T) {
	lower, upper := DeviceUIDRange(11, "user-1")
	rowKey := EncodeDeviceRowKey(11, "user-1", 2)
	otherUID := EncodeDeviceRowKey(11, "user-2", 1)

	require.True(t, bytes.Compare(lower, rowKey) <= 0)
	require.True(t, bytes.Compare(rowKey, upper) < 0)
	require.True(t, bytes.Compare(otherUID, upper) >= 0)
}
