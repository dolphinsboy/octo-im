package key

import (
	"fmt"
)

const (
	keyClassSlot  = 0x01
	keyClassMeta  = 0x02
	keyClassLocal = 0x03

	slotPrefixSize      = 1 + 1 + 4
	slotScopePrefixSize = slotPrefixSize + 1
	slotTableSize       = slotScopePrefixSize + 2
	slotHeaderSize      = slotTableSize + 1

	metaPrefixSize = 1 + 1 + 1
	metaTableSize  = metaPrefixSize + 2
	metaHeaderSize = metaTableSize + 1
)

func SlotPrefix(slotID uint32) []byte {
	key := make([]byte, 0, slotPrefixSize)
	key = append(key, Version, keyClassSlot)
	key = appendUint32(key, slotID)
	return key
}

func SlotScopePrefix(slotID uint32, scope ScopeType) []byte {
	if !scope.IsSlot() {
		panic(fmt.Sprintf("slot prefix requires slot scope, got: 0x%02x", byte(scope)))
	}
	key := SlotPrefix(slotID)
	return append(key, byte(scope))
}

func SlotTablePrefix(slotID uint32, scope ScopeType, table TableID) []byte {
	key := SlotScopePrefix(slotID, scope)
	return appendUint16(key, uint16(table))
}

func MetaPrefix(scope ScopeType) []byte {
	keyClass := byte(keyClassMeta)
	switch {
	case scope.IsMeta():
	case scope == ScopeLocal:
		keyClass = keyClassLocal
	default:
		panic(fmt.Sprintf("meta/local prefix requires meta or local scope, got: 0x%02x", byte(scope)))
	}
	return []byte{Version, keyClass, byte(scope)}
}

func MetaTablePrefix(scope ScopeType, table TableID) []byte {
	key := MetaPrefix(scope)
	return appendUint16(key, uint16(table))
}

func PrefixUpperBound(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	upper := append([]byte(nil), prefix...)
	for i := len(upper) - 1; i >= 0; i-- {
		if upper[i] == 0xFF {
			continue
		}
		upper[i]++
		return upper[:i+1]
	}
	return nil
}

func SlotRange(slotID uint32) (lower, upper []byte) {
	lower = SlotPrefix(slotID)
	return lower, PrefixUpperBound(lower)
}

func SlotScopeRange(slotID uint32, scope ScopeType) (lower, upper []byte) {
	lower = SlotScopePrefix(slotID, scope)
	return lower, PrefixUpperBound(lower)
}

func SlotTableRanges(slotID uint32, table TableID) []ScopedRange {
	return []ScopedRange{
		newTableScopedRange(slotID, ScopeSlotPrimary, table),
		newTableScopedRange(slotID, ScopeSlotIndex, table),
		newTableScopedRange(slotID, ScopeSlotSecondIdx, table),
		newTableScopedRange(slotID, ScopeSlotAux, table),
	}
}

func SlotTableRange(slotID uint32, scope ScopeType, table TableID) (lower, upper []byte) {
	lower = SlotTablePrefix(slotID, scope, table)
	return lower, PrefixUpperBound(lower)
}

func MetaTableRange(scope ScopeType, table TableID) (lower, upper []byte) {
	lower = MetaTablePrefix(scope, table)
	return lower, PrefixUpperBound(lower)
}

func Decode(key []byte) (DecodedKey, error) {
	if len(key) < 2 {
		return DecodedKey{}, fmt.Errorf("decode key: buffer too short")
	}
	out := DecodedKey{
		Version: key[0],
	}
	off := 2

	switch key[1] {
	case keyClassSlot:
		if len(key) < slotHeaderSize {
			return DecodedKey{}, fmt.Errorf("decode slot key: buffer too short")
		}
		slotID, next, err := readUint32(key, off)
		if err != nil {
			return DecodedKey{}, err
		}
		out.SlotID = slotID
		off = next
		out.Scope = ScopeType(key[off])
		if !out.Scope.IsSlot() {
			return DecodedKey{}, fmt.Errorf("decode slot key: invalid scope 0x%02x", byte(out.Scope))
		}
		off++
	case keyClassMeta:
		if len(key) < metaHeaderSize {
			return DecodedKey{}, fmt.Errorf("decode meta key: buffer too short")
		}
		out.Scope = ScopeType(key[off])
		if !out.Scope.IsMeta() {
			return DecodedKey{}, fmt.Errorf("decode meta key: invalid scope 0x%02x", byte(out.Scope))
		}
		off++
	case keyClassLocal:
		if len(key) < metaHeaderSize {
			return DecodedKey{}, fmt.Errorf("decode local key: buffer too short")
		}
		out.Scope = ScopeType(key[off])
		if out.Scope != ScopeLocal {
			return DecodedKey{}, fmt.Errorf("decode local key: invalid scope 0x%02x", byte(out.Scope))
		}
		off++
	default:
		return DecodedKey{}, fmt.Errorf("decode key: unknown key class 0x%02x", key[1])
	}
	tableID, next, err := readUint16(key, off)
	if err != nil {
		return DecodedKey{}, err
	}
	out.Table = TableID(tableID)
	off = next
	if len(key) <= off {
		return DecodedKey{}, fmt.Errorf("decode key: missing kind byte")
	}
	out.Kind = KeyKind(key[off])
	out.Tail = append([]byte(nil), key[off+1:]...)
	return out, nil
}

func newSlotKey(slotID uint32, scope ScopeType, table TableID, kind KeyKind) []byte {
	key := SlotTablePrefix(slotID, scope, table)
	return append(key, byte(kind))
}

func newMetaKey(scope ScopeType, table TableID, kind KeyKind) []byte {
	key := MetaTablePrefix(scope, table)
	return append(key, byte(kind))
}

func parseSlotKeyHeader(key []byte, scope ScopeType, table TableID, kind KeyKind) (uint32, int, error) {
	decoded, err := Decode(key)
	if err != nil {
		return 0, 0, err
	}
	if decoded.Scope != scope {
		return 0, 0, fmt.Errorf("unexpected scope: got 0x%02x want 0x%02x", byte(decoded.Scope), byte(scope))
	}
	if decoded.Table != table {
		return 0, 0, fmt.Errorf("unexpected table: got 0x%04x want 0x%04x", uint16(decoded.Table), uint16(table))
	}
	if decoded.Kind != kind {
		return 0, 0, fmt.Errorf("unexpected kind: got 0x%02x want 0x%02x", byte(decoded.Kind), byte(kind))
	}
	return decoded.SlotID, slotHeaderSize, nil
}

func parseMetaKeyHeader(key []byte, scope ScopeType, table TableID, kind KeyKind) (int, error) {
	decoded, err := Decode(key)
	if err != nil {
		return 0, err
	}
	if decoded.Scope != scope {
		return 0, fmt.Errorf("unexpected scope: got 0x%02x want 0x%02x", byte(decoded.Scope), byte(scope))
	}
	if decoded.Table != table {
		return 0, fmt.Errorf("unexpected table: got 0x%04x want 0x%04x", uint16(decoded.Table), uint16(table))
	}
	if decoded.Kind != kind {
		return 0, fmt.Errorf("unexpected kind: got 0x%02x want 0x%02x", byte(decoded.Kind), byte(kind))
	}
	return metaHeaderSize, nil
}

func newTableScopedRange(slotID uint32, scope ScopeType, table TableID) ScopedRange {
	lower, upper := SlotTableRange(slotID, scope, table)
	return ScopedRange{
		Scope: scope,
		Lower: lower,
		Upper: upper,
	}
}
