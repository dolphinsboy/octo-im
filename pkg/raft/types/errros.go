package types

import "errors"

var (
	ErrLogIndexNotContinuous  = errors.New("log index is not continuous")
	ErrStopped                = errors.New("raft is stopped")
	ErrNotLeader              = errors.New("not leader")
	ErrPaused                 = errors.New("raft is paused")
	ErrProposalDropped        = errors.New("proposal dropped")
	ErrLogCompacted           = errors.New("requested log has been compacted")
	ErrSnapshotNotSupported   = errors.New("snapshot is not supported")
	ErrCompactionNotSupported = errors.New("log compaction is not supported")
)
