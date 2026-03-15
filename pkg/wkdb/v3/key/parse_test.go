package key

import "testing"

import "github.com/stretchr/testify/require"

func TestParseTableID(t *testing.T) {
	t.Run("supports aliases", func(t *testing.T) {
		testCases := map[string]TableID{
			"user":                   TableUser,
			"channel":                TableChannelInfo,
			"channel_info":           TableChannelInfo,
			"channel-cluster-config": TableChannelClusterConfig,
			"message_event_state":    TableMessageEventState,
			"message-event-seq":      TableMessageEventSeq,
		}
		for input, expected := range testCases {
			got, err := ParseTableID(input)
			require.NoError(t, err)
			require.Equal(t, expected, got)
		}
	})

	t.Run("rejects unknown", func(t *testing.T) {
		_, err := ParseTableID("raw")
		require.ErrorContains(t, err, "unknown table id")
	})
}

func TestParseScopeType(t *testing.T) {
	t.Run("supports aliases", func(t *testing.T) {
		testCases := map[string]ScopeType{
			"primary":           ScopeSlotPrimary,
			"slot_primary":      ScopeSlotPrimary,
			"slot-index":        ScopeSlotIndex,
			"second_index":      ScopeSlotSecondIdx,
			"slot-second-index": ScopeSlotSecondIdx,
			"aux":               ScopeSlotAux,
			"meta_primary":      ScopeMetaPrimary,
			"local":             ScopeLocal,
		}
		for input, expected := range testCases {
			got, err := ParseScopeType(input)
			require.NoError(t, err)
			require.Equal(t, expected, got)
		}
	})

	t.Run("rejects unknown", func(t *testing.T) {
		_, err := ParseScopeType("slot")
		require.ErrorContains(t, err, "unknown scope type")
	})
}
