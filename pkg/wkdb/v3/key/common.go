package key

import (
	"encoding/binary"
	"fmt"
)

func appendUint16(dst []byte, v uint16) []byte {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, v)
	return append(dst, buf...)
}

func appendUint32(dst []byte, v uint32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, v)
	return append(dst, buf...)
}

func appendUint64(dst []byte, v uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, v)
	return append(dst, buf...)
}

func appendString(dst []byte, v string) []byte {
	dst = appendUint32(dst, uint32(len(v)))
	return append(dst, v...)
}

func appendBytes(dst []byte, v []byte) []byte {
	dst = appendUint32(dst, uint32(len(v)))
	return append(dst, v...)
}

func readUint16(src []byte, off int) (uint16, int, error) {
	if len(src) < off+2 {
		return 0, off, fmt.Errorf("read uint16: buffer too short")
	}
	return binary.BigEndian.Uint16(src[off : off+2]), off + 2, nil
}

func readUint32(src []byte, off int) (uint32, int, error) {
	if len(src) < off+4 {
		return 0, off, fmt.Errorf("read uint32: buffer too short")
	}
	return binary.BigEndian.Uint32(src[off : off+4]), off + 4, nil
}

func readUint64(src []byte, off int) (uint64, int, error) {
	if len(src) < off+8 {
		return 0, off, fmt.Errorf("read uint64: buffer too short")
	}
	return binary.BigEndian.Uint64(src[off : off+8]), off + 8, nil
}

func readString(src []byte, off int) (string, int, error) {
	size, next, err := readUint32(src, off)
	if err != nil {
		return "", off, err
	}
	end := next + int(size)
	if len(src) < end {
		return "", off, fmt.Errorf("read string: buffer too short")
	}
	return string(src[next:end]), end, nil
}
