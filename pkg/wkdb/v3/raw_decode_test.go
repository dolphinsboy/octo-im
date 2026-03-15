package v3

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	"github.com/stretchr/testify/require"
)

func TestDecodeSlotRawValue(t *testing.T) {
	now := time.Unix(1710000000, 0).UTC()

	t.Run("decodes user row", func(t *testing.T) {
		value, err := encodeUserValue(wkdb.User{
			Id:        11,
			Uid:       "user-1",
			PluginNo:  "plugin-a",
			CreatedAt: &now,
			UpdatedAt: &now,
		})
		require.NoError(t, err)

		decoded, err := DecodeSlotRawValue(v3key.DecodedKey{
			Scope: v3key.ScopeSlotPrimary,
			Table: v3key.TableUser,
			Kind:  v3key.KindRow,
		}, value)
		require.NoError(t, err)

		user, ok := decoded.(wkdb.User)
		require.True(t, ok)
		require.Equal(t, "user-1", user.Uid)
		require.Equal(t, "plugin-a", user.PluginNo)
	})

	t.Run("decodes conversation second index", func(t *testing.T) {
		rowKey := v3key.EncodeConversationRowKey(7, "user-1", "group-1", 2)

		decoded, err := DecodeSlotRawValue(v3key.DecodedKey{
			Scope: v3key.ScopeSlotSecondIdx,
			Table: v3key.TableConversation,
			Kind:  v3key.KindSecondIndex,
		}, rowKey)
		require.NoError(t, err)
		require.Equal(t, RawConversationRowRef{
			SlotID:      7,
			UID:         "user-1",
			ChannelID:   "group-1",
			ChannelType: 2,
		}, decoded)
	})

	t.Run("decodes message event seq aux", func(t *testing.T) {
		value := make([]byte, 8)
		binary.BigEndian.PutUint64(value, 9)

		decoded, err := DecodeSlotRawValue(v3key.DecodedKey{
			Scope: v3key.ScopeSlotAux,
			Table: v3key.TableMessageEventSeq,
			Kind:  v3key.KindAux,
		}, value)
		require.NoError(t, err)
		require.Equal(t, uint64(9), decoded)
	})

	t.Run("rejects meta scope", func(t *testing.T) {
		_, err := DecodeSlotRawValue(v3key.DecodedKey{
			Scope: v3key.ScopeMetaPrimary,
			Table: v3key.TableUser,
			Kind:  v3key.KindRow,
		}, nil)
		require.ErrorContains(t, err, "slot scope")
	})
}
