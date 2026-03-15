package slot

import (
	"testing"

	rafttype "github.com/WuKongIM/WuKongIM/pkg/raft/types"
	"github.com/stretchr/testify/require"
)

func TestServerValidateOptionsCompactionRequiresSnapshotCallbacks(t *testing.T) {
	s := &Server{opts: &Options{
		CompactionEnabled: true,
	}}

	err := s.validateOptions()
	require.ErrorIs(t, err, rafttype.ErrSnapshotNotSupported)
}

func TestServerValidateOptionsCompactionWithSnapshotCallbacks(t *testing.T) {
	s := &Server{opts: &Options{
		CompactionEnabled: true,
		OnCreateSnapshot: func(slotId uint32) ([]byte, error) {
			return []byte("snapshot"), nil
		},
		OnApplySnapshot: func(slotId uint32, data []byte) error {
			return nil
		},
	}}

	require.NoError(t, s.validateOptions())
}
