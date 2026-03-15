package v3

import (
	"testing"
	"time"

	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	"github.com/stretchr/testify/require"
)

func TestDescribeSlotRawKey(t *testing.T) {
	t.Run("describes conversation row", func(t *testing.T) {
		key := v3key.EncodeConversationRowKey(7, "user-1", "group-1", 2)
		meta, err := v3key.Decode(key)
		require.NoError(t, err)

		info, err := DescribeSlotRawKey(meta, key)
		require.NoError(t, err)
		require.Equal(t, "user-1", info.UID)
		require.Equal(t, "group-1", info.ChannelID)
		require.NotNil(t, info.ChannelType)
		require.Equal(t, uint8(2), *info.ChannelType)
		require.Equal(t, RawConversationRowKey{
			SlotID:      7,
			UID:         "user-1",
			ChannelID:   "group-1",
			ChannelType: 2,
		}, info.Decoded)
	})

	t.Run("describes user second index", func(t *testing.T) {
		now := time.Unix(1710000000, 0).UTC()
		key := v3key.EncodeUserCreatedAtSecondIndexKey(7, uint64(now.UnixNano()), "user-1")
		meta, err := v3key.Decode(key)
		require.NoError(t, err)

		info, err := DescribeSlotRawKey(meta, key)
		require.NoError(t, err)
		require.Equal(t, "user-1", info.UID)
		require.Equal(t, RawUserCreatedAtSecondIndexKey{
			SlotID:    7,
			CreatedAt: now,
			UID:       "user-1",
		}, info.Decoded)
	})

	t.Run("describes message event aux key", func(t *testing.T) {
		key := v3key.EncodeMessageEventSeqAuxKey(7, "group-1", 2, "client-1")
		meta, err := v3key.Decode(key)
		require.NoError(t, err)

		info, err := DescribeSlotRawKey(meta, key)
		require.NoError(t, err)
		require.Equal(t, "group-1", info.ChannelID)
		require.Equal(t, "client-1", info.ClientMsgNo)
		require.Equal(t, RawMessageEventSeqAuxKey{
			SlotID:      7,
			ChannelID:   "group-1",
			ChannelType: 2,
			ClientMsgNo: "client-1",
		}, info.Decoded)
	})
}
