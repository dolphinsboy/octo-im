package key

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSlotPrefixOrdering(t *testing.T) {
	p1 := SlotPrefix(7)
	p2 := SlotPrefix(8)
	require.True(t, bytes.Compare(p1, p2) < 0)
}

func TestPrefixUpperBound(t *testing.T) {
	prefix := []byte{0x01, 0x01, 0x00, 0x00, 0x00, 0x07}
	upper := PrefixUpperBound(prefix)
	require.Equal(t, []byte{0x01, 0x01, 0x00, 0x00, 0x00, 0x08}, upper)
}

func TestSlotRangeCoversAllSlotScopes(t *testing.T) {
	lower, upper := SlotRange(7)

	rowKey := EncodeConversationRowKey(7, "user-1", "group-100", 2)
	secondIndexKey := EncodeConversationUpdatedAtSecondIndexKey(7, "user-1", 100, "group-100", 2)
	auxKey := EncodeMessageEventSeqAuxKey(7, "group-100", 2, "client-1")
	nextSlotKey := EncodeConversationRowKey(8, "user-1", "group-100", 2)

	require.True(t, bytes.Compare(lower, rowKey) <= 0)
	require.True(t, bytes.Compare(rowKey, upper) < 0)
	require.True(t, bytes.Compare(lower, secondIndexKey) <= 0)
	require.True(t, bytes.Compare(secondIndexKey, upper) < 0)
	require.True(t, bytes.Compare(lower, auxKey) <= 0)
	require.True(t, bytes.Compare(auxKey, upper) < 0)
	require.True(t, bytes.Compare(nextSlotKey, upper) >= 0)
}

func TestSlotTableRangesExposeAllScopes(t *testing.T) {
	ranges := SlotTableRanges(9, TableConversation)
	require.Len(t, ranges, 4)
	require.Equal(t, SlotTablePrefix(9, ScopeSlotPrimary, TableConversation), ranges[0].Lower)
	require.Equal(t, SlotTablePrefix(9, ScopeSlotSecondIdx, TableConversation), ranges[2].Lower)
	require.Equal(t, PrefixUpperBound(SlotTablePrefix(9, ScopeSlotAux, TableConversation)), ranges[3].Upper)
}

func TestDecodeSlotKey(t *testing.T) {
	key := EncodeUserRowKey(9, "u-1")
	decoded, err := Decode(key)
	require.NoError(t, err)
	require.Equal(t, Version, decoded.Version)
	require.Equal(t, ScopeSlotPrimary, decoded.Scope)
	require.Equal(t, uint32(9), decoded.SlotID)
	require.Equal(t, TableUser, decoded.Table)
	require.Equal(t, KindRow, decoded.Kind)
}
