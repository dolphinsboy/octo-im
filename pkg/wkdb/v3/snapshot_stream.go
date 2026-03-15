package v3

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
	"io"
)

const (
	// SlotSnapshotFormatRawKVV1 is the first raw-KV slot snapshot stream format.
	SlotSnapshotFormatRawKVV1 uint16 = 1

	slotSnapshotHeaderSize       = 4 + 2 + 4 + 8
	slotSnapshotRecordHeaderSize = 4 + 4
)

var slotSnapshotMagic = [4]byte{'W', 'K', 'S', '3'}

// SlotSnapshotHeader is the fixed stream header written ahead of raw KV records.
type SlotSnapshotHeader struct {
	Format     uint16
	SlotID     uint32
	ExportedAt int64
}

// SlotSnapshotRecord is one raw KV pair inside a slot snapshot stream.
type SlotSnapshotRecord struct {
	Key   []byte
	Value []byte
}

type SlotSnapshotWriter struct {
	w        io.Writer
	checksum hash.Hash
	header   SlotSnapshotHeader
	meta     SlotSnapshotMeta
	closed   bool
}

func NewSlotSnapshotWriter(w io.Writer, header SlotSnapshotHeader) (*SlotSnapshotWriter, error) {
	if w == nil {
		return nil, fmt.Errorf("slot snapshot writer requires io.Writer")
	}
	if header.Format == 0 {
		header.Format = SlotSnapshotFormatRawKVV1
	}
	if header.Format != SlotSnapshotFormatRawKVV1 {
		return nil, fmt.Errorf("unsupported slot snapshot format: %d", header.Format)
	}

	sw := &SlotSnapshotWriter{
		w:        w,
		checksum: sha256.New(),
		header:   header,
		meta: SlotSnapshotMeta{
			SlotID:     header.SlotID,
			Format:     header.Format,
			ExportedAt: header.ExportedAt,
		},
	}
	if err := sw.writeHeader(); err != nil {
		return nil, err
	}
	return sw, nil
}

func (s *SlotSnapshotWriter) Header() SlotSnapshotHeader {
	return s.header
}

func (s *SlotSnapshotWriter) WriteRecord(key, value []byte) error {
	if s.closed {
		return fmt.Errorf("slot snapshot writer already closed")
	}
	if len(key) == 0 {
		return fmt.Errorf("slot snapshot record key cannot be empty")
	}
	if len(key) > int(^uint32(0)) {
		return fmt.Errorf("slot snapshot record key too large: %d", len(key))
	}
	if len(value) > int(^uint32(0)) {
		return fmt.Errorf("slot snapshot record value too large: %d", len(value))
	}

	var header [slotSnapshotRecordHeaderSize]byte
	binary.BigEndian.PutUint32(header[0:4], uint32(len(key)))
	binary.BigEndian.PutUint32(header[4:8], uint32(len(value)))

	if err := s.writeBytes(header[:]); err != nil {
		return err
	}
	if err := s.writeBytes(key); err != nil {
		return err
	}
	if err := s.writeBytes(value); err != nil {
		return err
	}
	s.meta.RecordCount++
	return nil
}

func (s *SlotSnapshotWriter) Close() (SlotSnapshotMeta, error) {
	if s.closed {
		return s.Meta(), nil
	}
	s.closed = true
	return s.Meta(), nil
}

func (s *SlotSnapshotWriter) Meta() SlotSnapshotMeta {
	meta := s.meta
	meta.Checksum = append([]byte(nil), s.checksum.Sum(nil)...)
	return meta
}

func (s *SlotSnapshotWriter) writeHeader() error {
	var header [slotSnapshotHeaderSize]byte
	copy(header[0:4], slotSnapshotMagic[:])
	binary.BigEndian.PutUint16(header[4:6], s.header.Format)
	binary.BigEndian.PutUint32(header[6:10], s.header.SlotID)
	binary.BigEndian.PutUint64(header[10:18], uint64(s.header.ExportedAt))
	return s.writeBytes(header[:])
}

func (s *SlotSnapshotWriter) writeBytes(p []byte) error {
	if len(p) == 0 {
		return nil
	}
	if _, err := s.w.Write(p); err != nil {
		return err
	}
	if _, err := s.checksum.Write(p); err != nil {
		return err
	}
	s.meta.ByteSize += uint64(len(p))
	return nil
}

type SlotSnapshotReader struct {
	r        io.Reader
	checksum hash.Hash
	header   SlotSnapshotHeader
	meta     SlotSnapshotMeta
	done     bool
}

func NewSlotSnapshotReader(r io.Reader) (*SlotSnapshotReader, error) {
	if r == nil {
		return nil, fmt.Errorf("slot snapshot reader requires io.Reader")
	}

	sr := &SlotSnapshotReader{
		r:        r,
		checksum: sha256.New(),
	}

	header, err := sr.readHeader()
	if err != nil {
		return nil, err
	}
	sr.header = header
	sr.meta.SlotID = header.SlotID
	sr.meta.Format = header.Format
	sr.meta.ExportedAt = header.ExportedAt
	return sr, nil
}

func (s *SlotSnapshotReader) Header() SlotSnapshotHeader {
	return s.header
}

func (s *SlotSnapshotReader) Next() (SlotSnapshotRecord, error) {
	if s.done {
		return SlotSnapshotRecord{}, io.EOF
	}

	var header [slotSnapshotRecordHeaderSize]byte
	if _, err := io.ReadFull(s.r, header[:]); err != nil {
		if err == io.EOF {
			s.done = true
			return SlotSnapshotRecord{}, io.EOF
		}
		return SlotSnapshotRecord{}, fmt.Errorf("read slot snapshot record header: %w", err)
	}
	s.ingest(header[:])

	keyLen := binary.BigEndian.Uint32(header[0:4])
	valueLen := binary.BigEndian.Uint32(header[4:8])
	var key []byte
	if keyLen > 0 {
		key = make([]byte, keyLen)
	}
	var value []byte
	if valueLen > 0 {
		value = make([]byte, valueLen)
	}

	if _, err := io.ReadFull(s.r, key); err != nil {
		return SlotSnapshotRecord{}, fmt.Errorf("read slot snapshot record key: %w", err)
	}
	s.ingest(key)

	if _, err := io.ReadFull(s.r, value); err != nil {
		return SlotSnapshotRecord{}, fmt.Errorf("read slot snapshot record value: %w", err)
	}
	s.ingest(value)

	s.meta.RecordCount++
	return SlotSnapshotRecord{
		Key:   key,
		Value: value,
	}, nil
}

func (s *SlotSnapshotReader) Meta() SlotSnapshotMeta {
	meta := s.meta
	meta.Checksum = append([]byte(nil), s.checksum.Sum(nil)...)
	return meta
}

func (s *SlotSnapshotReader) Verify(expected SlotSnapshotMeta) error {
	for {
		_, err := s.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return validateSlotSnapshotMeta(s.Meta(), expected)
}

func (s *SlotSnapshotReader) readHeader() (SlotSnapshotHeader, error) {
	var header [slotSnapshotHeaderSize]byte
	if _, err := io.ReadFull(s.r, header[:]); err != nil {
		return SlotSnapshotHeader{}, fmt.Errorf("read slot snapshot header: %w", err)
	}
	s.ingest(header[:])

	if !bytes.Equal(header[0:4], slotSnapshotMagic[:]) {
		return SlotSnapshotHeader{}, fmt.Errorf("invalid slot snapshot magic")
	}
	format := binary.BigEndian.Uint16(header[4:6])
	if format != SlotSnapshotFormatRawKVV1 {
		return SlotSnapshotHeader{}, fmt.Errorf("unsupported slot snapshot format: %d", format)
	}
	return SlotSnapshotHeader{
		Format:     format,
		SlotID:     binary.BigEndian.Uint32(header[6:10]),
		ExportedAt: int64(binary.BigEndian.Uint64(header[10:18])),
	}, nil
}

func (s *SlotSnapshotReader) ingest(p []byte) {
	if len(p) == 0 {
		return
	}
	_, _ = s.checksum.Write(p)
	s.meta.ByteSize += uint64(len(p))
}

func validateSlotSnapshotMeta(actual, expected SlotSnapshotMeta) error {
	if actual.SlotID != expected.SlotID {
		return fmt.Errorf("slot snapshot slot mismatch: got %d want %d", actual.SlotID, expected.SlotID)
	}
	if actual.Format != expected.Format {
		return fmt.Errorf("slot snapshot format mismatch: got %d want %d", actual.Format, expected.Format)
	}
	if actual.ExportedAt != expected.ExportedAt {
		return fmt.Errorf("slot snapshot exported_at mismatch: got %d want %d", actual.ExportedAt, expected.ExportedAt)
	}
	if actual.RecordCount != expected.RecordCount {
		return fmt.Errorf("slot snapshot record count mismatch: got %d want %d", actual.RecordCount, expected.RecordCount)
	}
	if actual.ByteSize != expected.ByteSize {
		return fmt.Errorf("slot snapshot byte size mismatch: got %d want %d", actual.ByteSize, expected.ByteSize)
	}
	if !bytes.Equal(actual.Checksum, expected.Checksum) {
		return fmt.Errorf("slot snapshot checksum mismatch")
	}
	return nil
}
