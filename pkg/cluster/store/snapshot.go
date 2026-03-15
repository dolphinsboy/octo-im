package store

import (
	"bytes"
	"context"
	"fmt"

	rafttypes "github.com/WuKongIM/WuKongIM/pkg/raft/types"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
	wkproto "github.com/WuKongIM/WuKongIMGoProto"
)

var slotSnapshotEnvelopeMagic = [4]byte{'W', 'K', 'S', 'S'}

const slotSnapshotEnvelopeVersion uint16 = 1

// SlotSnapshotBackend exposes the minimal slot snapshot primitives required by cluster store.
// wkdb/v3.DB satisfies this contract, but tests may inject smaller fakes.
type SlotSnapshotBackend interface {
	Snapshotter() wkdbv3.SlotSnapshotter
	Maintenance() wkdbv3.Maintenance
}

type slotSnapshotEnvelope struct {
	Meta wkdbv3.SlotSnapshotMeta
	Data []byte
}

func (s *Store) SupportsSlotSnapshot() bool {
	_, err := s.slotSnapshotBackend()
	return err == nil
}

func (s *Store) CreateSlotSnapshot(slotID uint32) ([]byte, error) {
	backend, err := s.slotSnapshotBackend()
	if err != nil {
		return nil, err
	}

	var raw bytes.Buffer
	meta, err := backend.Snapshotter().ExportSlotKV(context.Background(), slotID, &raw)
	if err != nil {
		return nil, err
	}

	return (&slotSnapshotEnvelope{
		Meta: meta,
		Data: raw.Bytes(),
	}).Marshal()
}

func (s *Store) ApplySlotSnapshot(slotID uint32, data []byte) error {
	backend, err := s.slotSnapshotBackend()
	if err != nil {
		return err
	}

	var envelope slotSnapshotEnvelope
	if err := envelope.Unmarshal(data); err != nil {
		return err
	}
	if envelope.Meta.SlotID != slotID {
		return fmt.Errorf("slot snapshot slot mismatch: got %d want %d", envelope.Meta.SlotID, slotID)
	}

	if err := backend.Snapshotter().ImportSlotKV(context.Background(), slotID, bytes.NewReader(envelope.Data), envelope.Meta); err != nil {
		return err
	}
	backend.Maintenance().ClearSlotCaches(slotID)
	return backend.Maintenance().RebuildDerivedState(context.Background(), slotID)
}

func (s *Store) slotSnapshotBackend() (SlotSnapshotBackend, error) {
	if s.opts.SlotSnapshotBackend != nil {
		if err := validateSlotSnapshotBackend(s.opts.SlotSnapshotBackend); err != nil {
			return nil, err
		}
		return s.opts.SlotSnapshotBackend, nil
	}

	backend, ok := any(s.wdb).(SlotSnapshotBackend)
	if !ok {
		return nil, rafttypes.ErrSnapshotNotSupported
	}
	if err := validateSlotSnapshotBackend(backend); err != nil {
		return nil, err
	}
	return backend, nil
}

func validateSlotSnapshotBackend(backend SlotSnapshotBackend) error {
	if backend == nil {
		return rafttypes.ErrSnapshotNotSupported
	}
	if backend.Snapshotter() == nil {
		return fmt.Errorf("slot snapshot backend requires snapshotter")
	}
	if backend.Maintenance() == nil {
		return fmt.Errorf("slot snapshot backend requires maintenance")
	}
	return nil
}

func (e *slotSnapshotEnvelope) Marshal() ([]byte, error) {
	if e == nil {
		return nil, fmt.Errorf("slot snapshot envelope is nil")
	}
	if uint64(len(e.Meta.Checksum)) > uint64(^uint32(0)) {
		return nil, fmt.Errorf("slot snapshot checksum too large: %d", len(e.Meta.Checksum))
	}
	if uint64(len(e.Data)) > uint64(^uint32(0)) {
		return nil, fmt.Errorf("slot snapshot data too large: %d", len(e.Data))
	}

	enc := wkproto.NewEncoder()
	defer enc.End()

	enc.WriteBytes(slotSnapshotEnvelopeMagic[:])
	enc.WriteUint16(slotSnapshotEnvelopeVersion)
	enc.WriteUint32(e.Meta.SlotID)
	enc.WriteUint16(e.Meta.Format)
	enc.WriteInt64(e.Meta.ExportedAt)
	enc.WriteUint64(e.Meta.RecordCount)
	enc.WriteUint64(e.Meta.ByteSize)
	enc.WriteUint32(uint32(len(e.Meta.Checksum)))
	enc.WriteBytes(e.Meta.Checksum)
	enc.WriteUint32(uint32(len(e.Data)))
	enc.WriteBytes(e.Data)

	return enc.Bytes(), nil
}

func (e *slotSnapshotEnvelope) Unmarshal(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("slot snapshot envelope is empty")
	}

	dec := wkproto.NewDecoder(data)

	magic, err := dec.Bytes(len(slotSnapshotEnvelopeMagic))
	if err != nil {
		return err
	}
	if !bytes.Equal(magic, slotSnapshotEnvelopeMagic[:]) {
		return fmt.Errorf("invalid slot snapshot envelope magic")
	}

	version, err := dec.Uint16()
	if err != nil {
		return err
	}
	if version != slotSnapshotEnvelopeVersion {
		return fmt.Errorf("unsupported slot snapshot envelope version: %d", version)
	}

	slotID, err := dec.Uint32()
	if err != nil {
		return err
	}
	format, err := dec.Uint16()
	if err != nil {
		return err
	}
	exportedAt, err := dec.Int64()
	if err != nil {
		return err
	}
	recordCount, err := dec.Uint64()
	if err != nil {
		return err
	}
	byteSize, err := dec.Uint64()
	if err != nil {
		return err
	}
	checksumLen, err := dec.Uint32()
	if err != nil {
		return err
	}
	checksum, err := dec.Bytes(int(checksumLen))
	if err != nil {
		return err
	}
	rawLen, err := dec.Uint32()
	if err != nil {
		return err
	}
	rawData, err := dec.Bytes(int(rawLen))
	if err != nil {
		return err
	}
	if dec.Len() != 0 {
		return fmt.Errorf("slot snapshot envelope has trailing bytes: %d", dec.Len())
	}

	e.Meta = wkdbv3.SlotSnapshotMeta{
		SlotID:      slotID,
		Format:      format,
		ExportedAt:  exportedAt,
		RecordCount: recordCount,
		ByteSize:    byteSize,
		Checksum:    append([]byte(nil), checksum...),
	}
	e.Data = append([]byte(nil), rawData...)
	return nil
}
