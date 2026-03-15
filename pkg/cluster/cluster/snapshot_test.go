package cluster

import (
	"context"
	"io"
	"testing"

	"github.com/WuKongIM/WuKongIM/pkg/cluster/slot"
	"github.com/WuKongIM/WuKongIM/pkg/cluster/store"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
	"github.com/stretchr/testify/require"
)

type testSlotSnapshotBackend struct {
	snapshotter testSlotSnapshotter
	maintenance testSlotMaintenance
}

func (t testSlotSnapshotBackend) Snapshotter() wkdbv3.SlotSnapshotter {
	return t.snapshotter
}

func (t testSlotSnapshotBackend) Maintenance() wkdbv3.Maintenance {
	return t.maintenance
}

type testSlotSnapshotter struct{}

func (testSlotSnapshotter) ExportSlotKV(ctx context.Context, slotID uint32, w io.Writer) (wkdbv3.SlotSnapshotMeta, error) {
	return wkdbv3.SlotSnapshotMeta{SlotID: slotID, Format: wkdbv3.SlotSnapshotFormatRawKVV1}, nil
}

func (testSlotSnapshotter) ImportSlotKV(ctx context.Context, slotID uint32, r io.Reader, meta wkdbv3.SlotSnapshotMeta) error {
	return nil
}

func (testSlotSnapshotter) DeleteSlotKV(ctx context.Context, slotID uint32) error {
	return nil
}

func (testSlotSnapshotter) VerifySlotKV(ctx context.Context, slotID uint32, r io.Reader, meta wkdbv3.SlotSnapshotMeta) error {
	return nil
}

type testSlotMaintenance struct{}

func (testSlotMaintenance) BucketCount() uint32 {
	return 0
}

func (testSlotMaintenance) BucketForSlot(slotID uint32) uint32 {
	return 0
}

func (testSlotMaintenance) ClearSlotCaches(slotID uint32) {}

func (testSlotMaintenance) RebuildDerivedState(ctx context.Context, slotID uint32) error {
	return nil
}

func TestServerBindSlotSnapshotCallbacksFromBackend(t *testing.T) {
	s := &Server{
		store: store.New(store.NewOptions(
			store.WithSlotSnapshotBackend(store.SlotSnapshotBackend(testSlotSnapshotBackend{
				snapshotter: testSlotSnapshotter{},
				maintenance: testSlotMaintenance{},
			})),
		)),
	}
	slotOpts := slot.NewOptions()

	s.bindSlotSnapshotCallbacks(slotOpts)

	require.NotNil(t, slotOpts.OnCreateSnapshot)
	require.NotNil(t, slotOpts.OnApplySnapshot)
}

func TestServerBindSlotSnapshotCallbacksKeepsExplicitOverrides(t *testing.T) {
	explicitCreate := func(slotID uint32) ([]byte, error) { return []byte("custom"), nil }
	applied := false
	explicitApply := func(slotID uint32, data []byte) error {
		applied = true
		return nil
	}

	s := &Server{
		store: store.New(store.NewOptions(
			store.WithSlotSnapshotBackend(store.SlotSnapshotBackend(testSlotSnapshotBackend{
				snapshotter: testSlotSnapshotter{},
				maintenance: testSlotMaintenance{},
			})),
		)),
	}
	slotOpts := slot.NewOptions(
		slot.WithOnCreateSnapshot(explicitCreate),
		slot.WithOnApplySnapshot(explicitApply),
	)

	s.bindSlotSnapshotCallbacks(slotOpts)

	require.NotNil(t, slotOpts.OnCreateSnapshot)
	require.NotNil(t, slotOpts.OnApplySnapshot)
	require.Equal(t, "custom", mustSnapshotData(t, slotOpts.OnCreateSnapshot))
	require.NoError(t, slotOpts.OnApplySnapshot(1, []byte("custom")))
	require.True(t, applied)
}

func mustSnapshotData(t *testing.T, fn func(slotId uint32) ([]byte, error)) string {
	t.Helper()

	data, err := fn(1)
	require.NoError(t, err)
	return string(data)
}
