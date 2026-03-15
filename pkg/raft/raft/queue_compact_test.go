package raft

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueueCompactTo(t *testing.T) {
	q := newQueue("test", 0, 0)
	assert.Equal(t, uint64(0), q.compactedIndex)

	q.compactTo(100)
	assert.Equal(t, uint64(100), q.compactedIndex)

	q.compactTo(200)
	assert.Equal(t, uint64(200), q.compactedIndex)
}

func TestQueueCompactToNoRegression(t *testing.T) {
	q := newQueue("test", 0, 0)

	q.compactTo(100)
	assert.Equal(t, uint64(100), q.compactedIndex)

	// 较小的值不应回退
	q.compactTo(50)
	assert.Equal(t, uint64(100), q.compactedIndex)

	// 相等的值不应改变
	q.compactTo(100)
	assert.Equal(t, uint64(100), q.compactedIndex)
}

func TestQueueResetFromSnapshot(t *testing.T) {
	q := newQueue("test", 50, 100)

	// 模拟一些状态
	q.compactedIndex = 30

	q.resetFromSnapshot(500)

	assert.Equal(t, uint64(500), q.storedIndex)
	assert.Equal(t, uint64(500), q.lastLogIndex)
	assert.Equal(t, uint64(500), q.committedIndex)
	assert.Equal(t, uint64(500), q.appliedIndex)
	assert.Equal(t, uint64(500), q.compactedIndex)
	assert.Nil(t, q.logs)
	assert.False(t, q.appending)
	assert.False(t, q.applying)
}
