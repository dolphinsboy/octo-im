package v3

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSlotSnapshotStreamRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	writer, err := NewSlotSnapshotWriter(&buf, SlotSnapshotHeader{
		SlotID:     23,
		ExportedAt: 1710000000,
	})
	require.NoError(t, err)

	require.NoError(t, writer.WriteRecord([]byte("k1"), []byte("v1")))
	require.NoError(t, writer.WriteRecord([]byte("k2"), nil))

	meta, err := writer.Close()
	require.NoError(t, err)
	require.Equal(t, uint32(23), meta.SlotID)
	require.Equal(t, SlotSnapshotFormatRawKVV1, meta.Format)
	require.Equal(t, int64(1710000000), meta.ExportedAt)
	require.Equal(t, uint64(2), meta.RecordCount)
	require.Equal(t, uint64(buf.Len()), meta.ByteSize)
	require.Len(t, meta.Checksum, 32)

	reader, err := NewSlotSnapshotReader(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	require.Equal(t, SlotSnapshotHeader{
		Format:     SlotSnapshotFormatRawKVV1,
		SlotID:     23,
		ExportedAt: 1710000000,
	}, reader.Header())

	record, err := reader.Next()
	require.NoError(t, err)
	require.Equal(t, SlotSnapshotRecord{Key: []byte("k1"), Value: []byte("v1")}, record)

	record, err = reader.Next()
	require.NoError(t, err)
	require.Equal(t, SlotSnapshotRecord{Key: []byte("k2"), Value: nil}, record)

	_, err = reader.Next()
	require.ErrorIs(t, err, io.EOF)
	require.NoError(t, reader.Verify(meta))
}

func TestSlotSnapshotWriterRejectsEmptyKey(t *testing.T) {
	var buf bytes.Buffer
	writer, err := NewSlotSnapshotWriter(&buf, SlotSnapshotHeader{SlotID: 1})
	require.NoError(t, err)

	err = writer.WriteRecord(nil, []byte("v1"))
	require.Error(t, err)
}

func TestSlotSnapshotReaderDetectsChecksumMismatch(t *testing.T) {
	var buf bytes.Buffer
	writer, err := NewSlotSnapshotWriter(&buf, SlotSnapshotHeader{SlotID: 9, ExportedAt: 123})
	require.NoError(t, err)
	require.NoError(t, writer.WriteRecord([]byte("k1"), []byte("v1")))

	meta, err := writer.Close()
	require.NoError(t, err)

	data := append([]byte(nil), buf.Bytes()...)
	data[len(data)-1]++

	reader, err := NewSlotSnapshotReader(bytes.NewReader(data))
	require.NoError(t, err)
	err = reader.Verify(meta)
	require.Error(t, err)
	require.Contains(t, err.Error(), "checksum")
}

func TestSlotSnapshotReaderRejectsTruncatedValue(t *testing.T) {
	var buf bytes.Buffer
	writer, err := NewSlotSnapshotWriter(&buf, SlotSnapshotHeader{SlotID: 7})
	require.NoError(t, err)
	require.NoError(t, writer.WriteRecord([]byte("k1"), []byte("value")))

	_, err = writer.Close()
	require.NoError(t, err)

	data := append([]byte(nil), buf.Bytes()...)
	data = data[:len(data)-2]

	reader, err := NewSlotSnapshotReader(bytes.NewReader(data))
	require.NoError(t, err)

	_, err = reader.Next()
	require.Error(t, err)
	require.Contains(t, err.Error(), "value")
}
