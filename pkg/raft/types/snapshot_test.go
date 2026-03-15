package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSnapshotMarshalUnmarshal(t *testing.T) {
	snapshot := Snapshot{
		LastIncludedIndex: 12345,
		LastIncludedTerm:  5,
		Config: Config{
			MigrateFrom: 1,
			MigrateTo:   2,
			Replicas:    []uint64{1, 2, 3},
			Learners:    []uint64{4},
			Role:        RoleLeader,
			Term:        5,
			Version:     10,
			Leader:      1,
		},
		Size:      1024 * 1024,
		CreatedAt: 1700000000000000000,
	}

	encoded, err := snapshot.Marshal()
	assert.NoError(t, err)

	var decoded Snapshot
	err = decoded.Unmarshal(encoded)
	assert.NoError(t, err)

	assert.Equal(t, snapshot.LastIncludedIndex, decoded.LastIncludedIndex)
	assert.Equal(t, snapshot.LastIncludedTerm, decoded.LastIncludedTerm)
	assert.Equal(t, snapshot.Size, decoded.Size)
	assert.Equal(t, snapshot.CreatedAt, decoded.CreatedAt)

	// Config 验证
	assert.Equal(t, snapshot.Config.MigrateFrom, decoded.Config.MigrateFrom)
	assert.Equal(t, snapshot.Config.MigrateTo, decoded.Config.MigrateTo)
	assert.Equal(t, snapshot.Config.Replicas, decoded.Config.Replicas)
	assert.Equal(t, snapshot.Config.Learners, decoded.Config.Learners)
	assert.Equal(t, snapshot.Config.Role, decoded.Config.Role)
	assert.Equal(t, snapshot.Config.Term, decoded.Config.Term)
	assert.Equal(t, snapshot.Config.Version, decoded.Config.Version)
	assert.Equal(t, snapshot.Config.Leader, decoded.Config.Leader)
}

func TestSnapshotDataMarshalUnmarshal(t *testing.T) {
	sd := SnapshotData{
		Meta: Snapshot{
			LastIncludedIndex: 999,
			LastIncludedTerm:  3,
			Config: Config{
				Replicas: []uint64{1, 2, 3},
				Role:     RoleFollower,
				Term:     3,
				Leader:   1,
			},
			Size:      512,
			CreatedAt: 1700000000000000000,
		},
		Data: []byte("state machine snapshot data here"),
	}

	encoded, err := sd.Marshal()
	assert.NoError(t, err)

	var decoded SnapshotData
	err = decoded.Unmarshal(encoded)
	assert.NoError(t, err)

	assert.Equal(t, sd.Meta.LastIncludedIndex, decoded.Meta.LastIncludedIndex)
	assert.Equal(t, sd.Meta.LastIncludedTerm, decoded.Meta.LastIncludedTerm)
	assert.Equal(t, sd.Meta.Size, decoded.Meta.Size)
	assert.Equal(t, sd.Meta.CreatedAt, decoded.Meta.CreatedAt)
	assert.Equal(t, sd.Meta.Config.Replicas, decoded.Meta.Config.Replicas)
	assert.Equal(t, sd.Meta.Config.Leader, decoded.Meta.Config.Leader)
	assert.Equal(t, sd.Data, decoded.Data)
}

func TestSnapshotEmptyConfig(t *testing.T) {
	snapshot := Snapshot{
		LastIncludedIndex: 100,
		LastIncludedTerm:  1,
		Config:            Config{},
		Size:              0,
		CreatedAt:         0,
	}

	encoded, err := snapshot.Marshal()
	assert.NoError(t, err)

	var decoded Snapshot
	err = decoded.Unmarshal(encoded)
	assert.NoError(t, err)

	assert.Equal(t, snapshot.LastIncludedIndex, decoded.LastIncludedIndex)
	assert.Equal(t, snapshot.LastIncludedTerm, decoded.LastIncludedTerm)
	assert.Empty(t, decoded.Config.Replicas)
	assert.Empty(t, decoded.Config.Learners)
	assert.Equal(t, RoleUnknown, decoded.Config.Role)
}

func TestSnapshotDataEmptyData(t *testing.T) {
	sd := SnapshotData{
		Meta: Snapshot{
			LastIncludedIndex: 50,
			LastIncludedTerm:  2,
			Size:              0,
		},
		Data: nil,
	}

	encoded, err := sd.Marshal()
	assert.NoError(t, err)

	var decoded SnapshotData
	err = decoded.Unmarshal(encoded)
	assert.NoError(t, err)

	assert.Equal(t, sd.Meta.LastIncludedIndex, decoded.Meta.LastIncludedIndex)
	assert.Equal(t, sd.Meta.LastIncludedTerm, decoded.Meta.LastIncludedTerm)
	assert.Empty(t, decoded.Data)
}

func TestSnapshotIsEmpty(t *testing.T) {
	assert.True(t, Snapshot{}.IsEmpty())
	assert.False(t, Snapshot{LastIncludedIndex: 1}.IsEmpty())
	assert.False(t, Snapshot{LastIncludedTerm: 1}.IsEmpty())
}
