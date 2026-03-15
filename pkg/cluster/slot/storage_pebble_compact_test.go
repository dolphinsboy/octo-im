package slot

import (
	"os"
	"testing"

	"github.com/WuKongIM/WuKongIM/pkg/raft/types"
	"github.com/stretchr/testify/assert"
)

func newTestPebbleStorage(t *testing.T) (*PebbleShardLogStorage, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "pebble-compact-test-*")
	assert.NoError(t, err)

	s := &Server{opts: &Options{}}
	storage := NewPebbleShardLogStorage(s, tmpDir, 1)
	err = storage.Open()
	assert.NoError(t, err)

	cleanup := func() {
		storage.Close()
		os.RemoveAll(tmpDir)
	}
	return storage, cleanup
}

func makeLogs(start, end uint64, term uint32) []types.Log {
	var logs []types.Log
	for i := start; i <= end; i++ {
		logs = append(logs, types.Log{
			Id:    i,
			Index: i,
			Term:  term,
			Data:  []byte("data"),
		})
	}
	return logs
}

// TestCompactLogTo 验证头部日志清理
func TestCompactLogTo(t *testing.T) {
	storage, cleanup := newTestPebbleStorage(t)
	defer cleanup()

	shardNo := "1"

	// 写入 10 条日志
	err := storage.AppendLogs(shardNo, makeLogs(1, 10, 1), &types.TermStartIndexInfo{Term: 1, Index: 1})
	assert.NoError(t, err)

	// 设置 appliedIndex = 7
	err = storage.SetAppliedIndex(shardNo, 7)
	assert.NoError(t, err)

	// 压缩到 index=5（删除 1-5）
	err = storage.CompactLogTo(shardNo, 5)
	assert.NoError(t, err)

	// 验证 1-5 已被删除
	logs, err := storage.GetLogs(shardNo, 1, 6, 0)
	assert.NoError(t, err)
	assert.Empty(t, logs)

	// 验证 6-10 仍然存在
	logs, err = storage.GetLogs(shardNo, 6, 11, 0)
	assert.NoError(t, err)
	assert.Len(t, logs, 5)
	assert.Equal(t, uint64(6), logs[0].Index)
	assert.Equal(t, uint64(10), logs[4].Index)

	// 验证 LastIndex 仍然是 10
	lastIdx, err := storage.LastIndex(shardNo)
	assert.NoError(t, err)
	assert.Equal(t, uint64(10), lastIdx)
}

// TestCompactLogToSafetyCheck 验证不能压缩超过 appliedIndex
func TestCompactLogToSafetyCheck(t *testing.T) {
	storage, cleanup := newTestPebbleStorage(t)
	defer cleanup()

	shardNo := "1"

	err := storage.AppendLogs(shardNo, makeLogs(1, 10, 1), nil)
	assert.NoError(t, err)

	err = storage.SetAppliedIndex(shardNo, 5)
	assert.NoError(t, err)

	// 尝试压缩到 index=8，超过 appliedIndex=5，应失败
	err = storage.CompactLogTo(shardNo, 8)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot compact beyond applied index")

	// 验证日志未被删除
	logs, err := storage.GetLogs(shardNo, 1, 11, 0)
	assert.NoError(t, err)
	assert.Len(t, logs, 10)
}

// TestCompactLogToZeroIndex 验证 index=0 时不做任何操作
func TestCompactLogToZeroIndex(t *testing.T) {
	storage, cleanup := newTestPebbleStorage(t)
	defer cleanup()

	shardNo := "1"

	err := storage.AppendLogs(shardNo, makeLogs(1, 5, 1), nil)
	assert.NoError(t, err)

	err = storage.CompactLogTo(shardNo, 0)
	assert.NoError(t, err)

	logs, err := storage.GetLogs(shardNo, 1, 6, 0)
	assert.NoError(t, err)
	assert.Len(t, logs, 5)
}

// TestSaveAndGetSnapshot 验证快照存取 round-trip
func TestSaveAndGetSnapshot(t *testing.T) {
	storage, cleanup := newTestPebbleStorage(t)
	defer cleanup()

	shardNo := "1"

	sd := types.SnapshotData{
		Meta: types.Snapshot{
			LastIncludedIndex: 100,
			LastIncludedTerm:  3,
			Config: types.Config{
				Replicas: []uint64{1, 2, 3},
				Role:     types.RoleLeader,
				Term:     3,
				Leader:   1,
			},
			Size:      2048,
			CreatedAt: 1700000000000000000,
		},
		Data: []byte("state machine data"),
	}

	err := storage.SaveSnapshot(shardNo, sd)
	assert.NoError(t, err)

	got, err := storage.GetSnapshot(shardNo)
	assert.NoError(t, err)

	assert.Equal(t, sd.Meta.LastIncludedIndex, got.Meta.LastIncludedIndex)
	assert.Equal(t, sd.Meta.LastIncludedTerm, got.Meta.LastIncludedTerm)
	assert.Equal(t, sd.Meta.Config.Replicas, got.Meta.Config.Replicas)
	assert.Equal(t, sd.Meta.Config.Leader, got.Meta.Config.Leader)
	assert.Equal(t, sd.Meta.Size, got.Meta.Size)
	assert.Equal(t, sd.Meta.CreatedAt, got.Meta.CreatedAt)
	assert.Equal(t, sd.Data, got.Data)
}

// TestGetSnapshotMeta 验证只获取元数据
func TestGetSnapshotMeta(t *testing.T) {
	storage, cleanup := newTestPebbleStorage(t)
	defer cleanup()

	shardNo := "1"

	sd := types.SnapshotData{
		Meta: types.Snapshot{
			LastIncludedIndex: 50,
			LastIncludedTerm:  2,
			Size:              1024,
		},
		Data: []byte("large state machine data"),
	}

	err := storage.SaveSnapshot(shardNo, sd)
	assert.NoError(t, err)

	meta, err := storage.GetSnapshotMeta(shardNo)
	assert.NoError(t, err)
	assert.Equal(t, uint64(50), meta.LastIncludedIndex)
	assert.Equal(t, uint32(2), meta.LastIncludedTerm)
	assert.Equal(t, uint64(1024), meta.Size)
}

// TestGetSnapshotNotFound 验证无快照时返回空值
func TestGetSnapshotNotFound(t *testing.T) {
	storage, cleanup := newTestPebbleStorage(t)
	defer cleanup()

	shardNo := "1"

	sd, err := storage.GetSnapshot(shardNo)
	assert.NoError(t, err)
	assert.True(t, sd.Meta.IsEmpty())
	assert.Empty(t, sd.Data)

	meta, err := storage.GetSnapshotMeta(shardNo)
	assert.NoError(t, err)
	assert.True(t, meta.IsEmpty())
}

// TestCompactLogToWithTermCleanup 验证压缩时清理旧 LeaderTermStartIndex
func TestCompactLogToWithTermCleanup(t *testing.T) {
	storage, cleanup := newTestPebbleStorage(t)
	defer cleanup()

	shardNo := "1"

	// Term 1: 日志 1-5
	err := storage.AppendLogs(shardNo, makeLogs(1, 5, 1), &types.TermStartIndexInfo{Term: 1, Index: 1})
	assert.NoError(t, err)

	// Term 2: 日志 6-10
	err = storage.AppendLogs(shardNo, makeLogs(6, 10, 2), &types.TermStartIndexInfo{Term: 2, Index: 6})
	assert.NoError(t, err)

	// Term 3: 日志 11-15
	err = storage.AppendLogs(shardNo, makeLogs(11, 15, 3), &types.TermStartIndexInfo{Term: 3, Index: 11})
	assert.NoError(t, err)

	err = storage.SetAppliedIndex(shardNo, 12)
	assert.NoError(t, err)

	// 压缩到 index=10
	err = storage.CompactLogTo(shardNo, 10)
	assert.NoError(t, err)

	// 验证日志 1-10 已删除
	logs, err := storage.GetLogs(shardNo, 1, 11, 0)
	assert.NoError(t, err)
	assert.Empty(t, logs)

	// 验证日志 11-15 仍然存在
	logs, err = storage.GetLogs(shardNo, 11, 16, 0)
	assert.NoError(t, err)
	assert.Len(t, logs, 5)

	// 验证 Term 1 和 Term 2 的 LeaderTermStartIndex 已被清理
	idx, err := storage.GetTermStartIndex(shardNo, 1)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), idx)

	idx, err = storage.GetTermStartIndex(shardNo, 2)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), idx)

	// 验证 Term 3 的 LeaderTermStartIndex 仍存在
	idx, err = storage.GetTermStartIndex(shardNo, 3)
	assert.NoError(t, err)
	assert.Equal(t, uint64(11), idx)
}

// TestCompactLogToKeepsSpanningTermStart 验证压缩点后的同任期日志不会丢失 term 起点元数据
func TestCompactLogToKeepsSpanningTermStart(t *testing.T) {
	storage, cleanup := newTestPebbleStorage(t)
	defer cleanup()

	shardNo := "1"

	err := storage.AppendLogs(shardNo, makeLogs(1, 5, 1), &types.TermStartIndexInfo{Term: 1, Index: 1})
	assert.NoError(t, err)
	err = storage.AppendLogs(shardNo, makeLogs(6, 14, 2), &types.TermStartIndexInfo{Term: 2, Index: 6})
	assert.NoError(t, err)
	err = storage.AppendLogs(shardNo, makeLogs(15, 18, 3), &types.TermStartIndexInfo{Term: 3, Index: 15})
	assert.NoError(t, err)
	err = storage.SetAppliedIndex(shardNo, 16)
	assert.NoError(t, err)

	err = storage.CompactLogTo(shardNo, 10)
	assert.NoError(t, err)

	idx, err := storage.GetTermStartIndex(shardNo, 1)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), idx)

	idx, err = storage.GetTermStartIndex(shardNo, 2)
	assert.NoError(t, err)
	assert.Equal(t, uint64(6), idx)

	idx, err = storage.GetTermStartIndex(shardNo, 3)
	assert.NoError(t, err)
	assert.Equal(t, uint64(15), idx)
}

func TestGetStateUsesSnapshotMeta(t *testing.T) {
	storage, cleanup := newTestPebbleStorage(t)
	defer cleanup()

	shardNo := "1"
	sd := types.SnapshotData{
		Meta: types.Snapshot{
			LastIncludedIndex: 50,
			LastIncludedTerm:  4,
		},
	}
	err := storage.SaveSnapshot(shardNo, sd)
	assert.NoError(t, err)

	state, err := storage.GetState(shardNo)
	assert.NoError(t, err)
	assert.Equal(t, uint64(50), state.LastLogIndex)
	assert.Equal(t, uint32(4), state.LastTerm)
	assert.Equal(t, uint64(50), state.LastTermStartIndex)
	assert.Equal(t, uint64(50), state.AppliedIndex)
	assert.Equal(t, uint64(50), state.CompactedIndex)
}

func TestApplySnapshotResetsLocalRaftState(t *testing.T) {
	storage, cleanup := newTestPebbleStorage(t)
	defer cleanup()

	shardNo := "1"
	err := storage.AppendLogs(shardNo, makeLogs(1, 8, 2), &types.TermStartIndexInfo{Term: 2, Index: 1})
	assert.NoError(t, err)
	err = storage.SetAppliedIndex(shardNo, 8)
	assert.NoError(t, err)

	sd := types.SnapshotData{
		Meta: types.Snapshot{
			LastIncludedIndex: 20,
			LastIncludedTerm:  5,
		},
	}
	err = storage.SaveSnapshot(shardNo, sd)
	assert.NoError(t, err)
	err = storage.ApplySnapshot(shardNo, sd)
	assert.NoError(t, err)

	logs, err := storage.GetLogs(shardNo, 1, 100, 0)
	assert.NoError(t, err)
	assert.Empty(t, logs)

	state, err := storage.GetState(shardNo)
	assert.NoError(t, err)
	assert.Equal(t, uint64(20), state.LastLogIndex)
	assert.Equal(t, uint32(5), state.LastTerm)
	assert.Equal(t, uint64(20), state.LastTermStartIndex)
	assert.Equal(t, uint64(20), state.AppliedIndex)
	assert.Equal(t, uint64(20), state.CompactedIndex)

	idx, err := storage.GetTermStartIndex(shardNo, 5)
	assert.NoError(t, err)
	assert.Equal(t, uint64(20), idx)
}
