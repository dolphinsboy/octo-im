package raft

import (
	"testing"

	"github.com/WuKongIM/WuKongIM/pkg/raft/types"
	"github.com/stretchr/testify/assert"
)

func newCompactionNode(nodeId uint64, replicas []uint64, minLogCount uint64, retainCount uint64, intervalTick int) *Node {
	opts := NewOptions(
		WithNodeId(nodeId),
		WithReplicas(replicas),
		WithElectionOn(false),
		WithElectionInterval(1000),
		WithAdvance(func() {}),
		WithKey("test"),
		WithCompactionEnabled(true),
		WithCompactionMinLogCount(minLogCount),
		WithCompactionRetainCount(retainCount),
		WithCompactionIntervalTick(intervalTick),
	)
	return NewNode(0, types.RaftState{}, opts)
}

func TestShouldCompact_NotEnabled(t *testing.T) {
	n := newTestNode(1, []uint64{1})
	makeLeader(n, 1)
	n.queue.appliedIndex = 20000
	n.compactionElapsed = 10000
	// CompactionEnabled 默认 false
	assert.False(t, n.shouldCompact())
}

func TestShouldCompact_NotLeader(t *testing.T) {
	n := newCompactionNode(1, []uint64{1, 2}, 10, 5, 1)
	makeFollower(n, 1, 2)
	n.queue.appliedIndex = 100
	n.compactionElapsed = 10
	assert.False(t, n.shouldCompact())
}

func TestShouldCompact_AlreadyCompacting(t *testing.T) {
	n := newCompactionNode(1, []uint64{1}, 10, 5, 1)
	makeLeader(n, 1)
	n.queue.appliedIndex = 100
	n.compactionElapsed = 10
	n.compacting = true
	assert.False(t, n.shouldCompact())
}

func TestShouldCompact_NotEnoughLogs(t *testing.T) {
	n := newCompactionNode(1, []uint64{1}, 100, 10, 1)
	makeLeader(n, 1)
	n.queue.appliedIndex = 50 // 50 < minLogCount(100)
	n.compactionElapsed = 10
	assert.False(t, n.shouldCompact())
}

func TestShouldCompact_IntervalNotReached(t *testing.T) {
	n := newCompactionNode(1, []uint64{1}, 10, 5, 100)
	makeLeader(n, 1)
	n.queue.appliedIndex = 200
	n.compactionElapsed = 50 // 50 < intervalTick(100)
	assert.False(t, n.shouldCompact())
}

func TestShouldCompact_Triggers(t *testing.T) {
	n := newCompactionNode(1, []uint64{1}, 10, 5, 3)
	makeLeader(n, 1)
	n.queue.appliedIndex = 100
	n.compactionElapsed = 3 // >= intervalTick(3)

	assert.True(t, n.shouldCompact())

	// Ready() 应产出 CompactReq
	events := collectEvents(n)
	e, found := findEvent(events, types.CompactReq)
	assert.True(t, found)
	// target = appliedIndex(100) - retainCount(5) = 95
	assert.Equal(t, uint64(95), e.Index)
	// 应标记为压缩中
	assert.True(t, n.compacting)
}

func TestCompactTargetIndex(t *testing.T) {
	n := newCompactionNode(1, []uint64{1}, 10, 20, 1)
	makeLeader(n, 1)

	// appliedIndex = 100, retainCount = 20, compactedIndex = 0
	// target = 100 - 20 = 80
	n.queue.appliedIndex = 100
	assert.Equal(t, uint64(80), n.compactTargetIndex())

	// compactedIndex 已经到 80，不需要再压缩
	n.queue.compactedIndex = 80
	assert.Equal(t, uint64(0), n.compactTargetIndex())

	// appliedIndex 不够 retainCount
	n.queue.compactedIndex = 0
	n.queue.appliedIndex = 10 // 10 <= retainCount(20)
	assert.Equal(t, uint64(0), n.compactTargetIndex())
}

func TestCompactResp_UpdatesQueue(t *testing.T) {
	n := newCompactionNode(1, []uint64{1}, 10, 5, 1)
	makeLeader(n, 1)
	n.compacting = true

	// 模拟 CompactResp 成功
	err := n.Step(types.Event{
		Type:   types.CompactResp,
		Index:  50,
		Reason: types.ReasonOk,
	})
	assert.NoError(t, err)

	assert.False(t, n.compacting)
	assert.Equal(t, uint64(50), n.queue.compactedIndex)
}

func TestCompactResp_Error(t *testing.T) {
	n := newCompactionNode(1, []uint64{1}, 10, 5, 1)
	makeLeader(n, 1)
	n.compacting = true

	// 模拟 CompactResp 失败
	err := n.Step(types.Event{
		Type:   types.CompactResp,
		Reason: types.ReasonError,
	})
	assert.NoError(t, err)

	// 压缩标志应被清除，但 compactedIndex 不变
	assert.False(t, n.compacting)
	assert.Equal(t, uint64(0), n.queue.compactedIndex)
}

func TestCompacting_BlocksSync(t *testing.T) {
	n := newCompactionNode(1, []uint64{1, 2}, 10, 5, 1)
	makeFollower(n, 1, 2)
	n.compacting = true

	// 压缩中调用 sendSyncReq 不应产出 SyncReq 事件
	n.sendSyncReq()
	events := collectEvents(n)
	_, found := findEvent(events, types.SyncReq)
	assert.False(t, found)
}

func TestTickCompaction(t *testing.T) {
	n := newCompactionNode(1, []uint64{1}, 10, 5, 5)
	makeLeader(n, 1)

	assert.Equal(t, 0, n.compactionElapsed)

	// tick 3 次
	tickN(n, 3)
	assert.Equal(t, 3, n.compactionElapsed)

	// 压缩中不应递增
	n.compacting = true
	tickN(n, 2)
	assert.Equal(t, 3, n.compactionElapsed) // 不变
}

// TestFollowerReceivesSnapshotInstall 验证 Follower 完整的快照安装流程（TruncateReq 模式）
func TestFollowerReceivesSnapshotInstall(t *testing.T) {
	n := newTestNode(1, []uint64{1, 2})
	makeFollower(n, 1, 2)

	// 模拟 Follower 有一些日志
	n.queue.lastLogIndex = 50
	n.queue.storedIndex = 50
	n.queue.committedIndex = 40
	n.queue.appliedIndex = 40

	// 阶段1：收到 SyncResp with ReasonInstallSnapshot
	snapshotLogs := []types.Log{{Index: 200, Term: 3, Data: []byte("snapshot-data")}}
	err := n.Step(types.Event{
		Type:        types.SyncResp,
		From:        2,
		To:          1,
		Term:        1,
		Index:       200,
		LastLogTerm: 3,
		Logs:        snapshotLogs,
		Reason:      types.ReasonInstallSnapshot,
	})
	assert.NoError(t, err)

	// 应标记为安装中
	assert.True(t, n.installing)

	// 应产出 InstallSnapshotReq 本地事件（不是直接 resetFromSnapshot）
	events := collectEvents(n)
	e, found := findEvent(events, types.InstallSnapshotReq)
	assert.True(t, found)
	assert.Equal(t, uint64(200), e.Index)
	assert.Equal(t, uint32(3), e.LastLogTerm)
	assert.Len(t, e.Logs, 1)
	assert.Equal(t, []byte("snapshot-data"), e.Logs[0].Data)

	// queue 尚未被重置（等 raft.go 处理后回传 Resp）
	assert.Equal(t, uint64(50), n.queue.lastLogIndex)

	// 阶段2：模拟 raft.go 处理完成，回传 InstallSnapshotResp
	err = n.Step(types.Event{
		Type:        types.InstallSnapshotResp,
		Index:       200,
		LastLogTerm: 3,
		Reason:      types.ReasonOk,
	})
	assert.NoError(t, err)

	// 验证 installing 标志已清除
	assert.False(t, n.installing)

	// 验证 queue 被重置到快照索引
	assert.Equal(t, uint64(200), n.queue.lastLogIndex)
	assert.Equal(t, uint64(200), n.queue.storedIndex)
	assert.Equal(t, uint64(200), n.queue.committedIndex)
	assert.Equal(t, uint64(200), n.queue.appliedIndex)
	assert.Equal(t, uint64(200), n.queue.compactedIndex)

	// 验证 lastTermStartIndex 更新
	assert.Equal(t, uint32(3), n.lastTermStartIndex.Term)
	assert.Equal(t, uint64(200), n.lastTermStartIndex.Index)

	// 验证产出 SyncReq（从快照之后继续同步）
	events = collectEvents(n)
	_, found = findEvent(events, types.SyncReq)
	assert.True(t, found)
}

// TestLearnerReceivesSnapshotInstall 验证 Learner 完整的快照安装流程
func TestLearnerReceivesSnapshotInstall(t *testing.T) {
	n := newTestNode(1, []uint64{2})
	makeLearner(n, 1, 2)

	n.queue.lastLogIndex = 10
	n.queue.storedIndex = 10
	n.queue.committedIndex = 10
	n.queue.appliedIndex = 10

	// 阶段1：收到 SyncResp with ReasonInstallSnapshot
	err := n.Step(types.Event{
		Type:        types.SyncResp,
		From:        2,
		To:          1,
		Term:        1,
		Index:       500,
		LastLogTerm: 5,
		Logs:        []types.Log{{Index: 500, Term: 5, Data: []byte("learner-snapshot")}},
		Reason:      types.ReasonInstallSnapshot,
	})
	assert.NoError(t, err)
	assert.True(t, n.installing)

	// 应产出 InstallSnapshotReq
	events := collectEvents(n)
	_, found := findEvent(events, types.InstallSnapshotReq)
	assert.True(t, found)

	// 阶段2：模拟 InstallSnapshotResp
	err = n.Step(types.Event{
		Type:        types.InstallSnapshotResp,
		Index:       500,
		LastLogTerm: 5,
		Reason:      types.ReasonOk,
	})
	assert.NoError(t, err)

	assert.False(t, n.installing)
	assert.Equal(t, uint64(500), n.queue.lastLogIndex)
	assert.Equal(t, uint64(500), n.queue.appliedIndex)
	assert.Equal(t, uint64(500), n.queue.compactedIndex)
	assert.Equal(t, uint32(5), n.lastTermStartIndex.Term)

	events = collectEvents(n)
	_, found = findEvent(events, types.SyncReq)
	assert.True(t, found)
}

// TestInstalling_BlocksSync 验证快照安装中阻止同步
func TestInstalling_BlocksSync(t *testing.T) {
	n := newTestNode(1, []uint64{1, 2})
	makeFollower(n, 1, 2)
	n.installing = true

	n.sendSyncReq()
	events := collectEvents(n)
	_, found := findEvent(events, types.SyncReq)
	assert.False(t, found)
}
