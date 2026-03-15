package types

import (
	wkproto "github.com/WuKongIM/WuKongIMGoProto"
)

// Snapshot 快照元数据
type Snapshot struct {
	// LastIncludedIndex 快照包含的最后一条日志索引
	LastIncludedIndex uint64
	// LastIncludedTerm 快照包含的最后一条日志任期
	LastIncludedTerm uint32
	// Config 快照创建时的 Raft 配置
	Config Config
	// Size 快照数据的大小（字节）
	Size uint64
	// CreatedAt 快照创建时间（UnixNano）
	CreatedAt int64
}

func (s *Snapshot) Marshal() ([]byte, error) {
	enc := wkproto.NewEncoder()
	defer enc.End()

	enc.WriteUint64(s.LastIncludedIndex)
	enc.WriteUint32(s.LastIncludedTerm)

	// Config 序列化，复用 Event 中的内联写法
	enc.WriteUint64(s.Config.MigrateFrom)
	enc.WriteUint64(s.Config.MigrateTo)
	enc.WriteUint32(uint32(len(s.Config.Replicas)))
	for _, v := range s.Config.Replicas {
		enc.WriteUint64(v)
	}
	enc.WriteUint32(uint32(len(s.Config.Learners)))
	for _, v := range s.Config.Learners {
		enc.WriteUint64(v)
	}
	enc.WriteUint8(uint8(s.Config.Role))
	enc.WriteUint32(s.Config.Term)
	enc.WriteUint64(s.Config.Version)
	enc.WriteUint64(s.Config.Leader)

	enc.WriteUint64(s.Size)
	enc.WriteInt64(s.CreatedAt)

	return enc.Bytes(), nil
}

func (s *Snapshot) Unmarshal(data []byte) error {
	dec := wkproto.NewDecoder(data)

	var err error
	if s.LastIncludedIndex, err = dec.Uint64(); err != nil {
		return err
	}
	if s.LastIncludedTerm, err = dec.Uint32(); err != nil {
		return err
	}

	// Config 反序列化
	if s.Config.MigrateFrom, err = dec.Uint64(); err != nil {
		return err
	}
	if s.Config.MigrateTo, err = dec.Uint64(); err != nil {
		return err
	}
	replicasLen, err := dec.Uint32()
	if err != nil {
		return err
	}
	for i := 0; i < int(replicasLen); i++ {
		replica, err := dec.Uint64()
		if err != nil {
			return err
		}
		s.Config.Replicas = append(s.Config.Replicas, replica)
	}
	learnersLen, err := dec.Uint32()
	if err != nil {
		return err
	}
	for i := 0; i < int(learnersLen); i++ {
		learner, err := dec.Uint64()
		if err != nil {
			return err
		}
		s.Config.Learners = append(s.Config.Learners, learner)
	}
	role, err := dec.Uint8()
	if err != nil {
		return err
	}
	s.Config.Role = Role(role)
	if s.Config.Term, err = dec.Uint32(); err != nil {
		return err
	}
	if s.Config.Version, err = dec.Uint64(); err != nil {
		return err
	}
	if s.Config.Leader, err = dec.Uint64(); err != nil {
		return err
	}

	if s.Size, err = dec.Uint64(); err != nil {
		return err
	}
	if s.CreatedAt, err = dec.Int64(); err != nil {
		return err
	}

	return nil
}

// IsEmpty 判断快照是否为空
func (s Snapshot) IsEmpty() bool {
	return s.LastIncludedIndex == 0 && s.LastIncludedTerm == 0
}

// SnapshotData 快照数据载体（元数据 + 状态机数据）
type SnapshotData struct {
	Meta Snapshot
	// Data 状态机的序列化数据
	Data []byte
}

func (s *SnapshotData) Marshal() ([]byte, error) {
	enc := wkproto.NewEncoder()
	defer enc.End()

	metaData, err := s.Meta.Marshal()
	if err != nil {
		return nil, err
	}
	enc.WriteUint32(uint32(len(metaData)))
	enc.WriteBytes(metaData)
	enc.WriteUint32(uint32(len(s.Data)))
	enc.WriteBytes(s.Data)

	return enc.Bytes(), nil
}

func (s *SnapshotData) Unmarshal(data []byte) error {
	dec := wkproto.NewDecoder(data)

	metaLen, err := dec.Uint32()
	if err != nil {
		return err
	}
	metaData, err := dec.Bytes(int(metaLen))
	if err != nil {
		return err
	}
	if err := s.Meta.Unmarshal(metaData); err != nil {
		return err
	}

	dataLen, err := dec.Uint32()
	if err != nil {
		return err
	}
	if dataLen > 0 {
		s.Data, err = dec.Bytes(int(dataLen))
		if err != nil {
			return err
		}
	}

	return nil
}
