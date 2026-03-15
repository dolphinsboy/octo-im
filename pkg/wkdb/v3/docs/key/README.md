# wkdb v3 Key Builder Draft

## Purpose

This document defines the intended package contract for `pkg/wkdb/v3/key`.

This file lives under `pkg/wkdb/v3/docs/key/` because all `v3` Markdown
documents are centralized under `docs/`.

This package should be responsible for:

- binary key encoding
- binary key parsing
- slot/meta/local prefix helpers
- range helpers for scans and deletes
- table-family level key builders

This package should not:

- decide routing
- open DBs
- manipulate Pebble directly
- contain business write logic

## Package Goals

`v3/key` should make these operations explicit and reusable:

1. build a slot prefix
2. build a meta prefix
3. build table row keys
4. build table index keys
5. build lower/upper bounds for prefix scans
6. parse keys back into structured parts for tooling and tests

## Design Principles

### 1. Prefix-First APIs

The package should make it easy to ask:

- all keys of one slot
- all keys of one slot and one table
- all keys of one slot and one table index family

These are first-class operations in `v3`.

With the current draft envelope, "all keys of one slot" is represented as a small ordered range set across slot scopes, so the package should expose helpers for that instead of forcing callers to rebuild the scope list themselves.

### 2. Symmetric Encode/Parse

Every structured key builder should ideally have a matching parse helper.

This matters for:

- migration tooling
- verification
- debugging
- offline inspection

### 3. Small, Typed Builders

Prefer many narrow helpers over one giant generic builder.

Good:

```go
EncodeUserRowKey(...)
EncodeConversationRowKey(...)
EncodeChannelInfoRowKey(...)
```

Not ideal:

```go
EncodeKey(namespace, table, kind, rawParts...)
```

Generic builders may still exist internally, but typed wrappers should be the public contract.

## Recommended Package Structure

One possible structure:

```text
pkg/wkdb/v3/key/
  prefix.go
  table.go
  common.go
  user.go
  device.go
  conversation.go
  channel.go
  subscriber.go
  allowlist.go
  denylist.go
  cluster_config.go
  message_event.go
  meta.go
```

This mirrors table/domain boundaries and keeps files readable.

## Core Types

The package should expose a few stable types.

### ScopeType

```go
type ScopeType byte

const (
    ScopeSlotPrimary   ScopeType = 0x10
    ScopeSlotIndex     ScopeType = 0x11
    ScopeSlotSecondIdx ScopeType = 0x12
    ScopeSlotAux       ScopeType = 0x13

    ScopeMetaPrimary   ScopeType = 0x20
    ScopeMetaIndex     ScopeType = 0x21
    ScopeMetaSecondIdx ScopeType = 0x22

    ScopeLocal         ScopeType = 0x30
)
```

### TableID

```go
type TableID uint16
```

Prefer a typed alias over raw `[2]byte` in the public builder package.

Internally it can still be encoded as two bytes.

### KeyKind

```go
type KeyKind byte

const (
    KindRow KeyKind = 0x01
    KindIndex KeyKind = 0x02
    KindSecondIndex KeyKind = 0x03
    KindAux KeyKind = 0x04
)
```

### DecodedKey

Useful for tooling and tests:

```go
type DecodedKey struct {
    Version uint8
    Scope   ScopeType
    SlotID  uint32
    Table   TableID
    Kind    KeyKind
    Tail    []byte
}
```

Not every business path needs `DecodedKey`, but migration and verification tools will.

## Core Prefix Helpers

These should be the first implemented helpers.

```go
func SlotPrefix(slotId uint32) []byte
func SlotScopePrefix(slotId uint32, scope ScopeType) []byte
func SlotTablePrefix(slotId uint32, scope ScopeType, table TableID) []byte
func MetaPrefix(scope ScopeType) []byte
func MetaTablePrefix(scope ScopeType, table TableID) []byte
func PrefixUpperBound(prefix []byte) []byte
```

`PrefixUpperBound` is important because:

- delete by slot
- iterate one slot
- iterate one slot and table

all rely on safe upper bounds.

## Common Encoding Helpers

`common.go` should expose shared binary helpers:

```go
func AppendUint8(dst []byte, v uint8) []byte
func AppendUint16(dst []byte, v uint16) []byte
func AppendUint32(dst []byte, v uint32) []byte
func AppendUint64(dst []byte, v uint64) []byte
func AppendString(dst []byte, s string) []byte
func AppendBytes(dst []byte, b []byte) []byte
```

And parse helpers:

```go
func ReadUint8(src []byte, off int) (uint8, int, error)
func ReadUint16(src []byte, off int) (uint16, int, error)
func ReadUint32(src []byte, off int) (uint32, int, error)
func ReadUint64(src []byte, off int) (uint64, int, error)
func ReadString(src []byte, off int) (string, int, error)
```

The package should use a single consistent string/bytes framing rule.

## Naming Conventions

Recommended naming:

- `SlotPrefix`
- `SlotTablePrefix`
- `EncodeXxxRowKey`
- `EncodeXxxIndexKey`
- `EncodeXxxSecondIndexKey`
- `EncodeXxxAuxKey`
- `ParseXxxRowKey`
- `ParseXxxIndexKey`

Avoid reusing the old mixed `NewXxxKey` style everywhere, because:

- `v3` should distinguish row/index/aux more explicitly
- builder names should tell the caller what semantic record is being encoded

## Domain Builder Drafts

### User

```go
func EncodeUserRowKey(slotId uint32, uid string) []byte
func EncodeUserCreatedAtSecondIndexKey(slotId uint32, createdAt uint64, uid string) []byte
func ParseUserRowKey(key []byte) (slotId uint32, uid string, err error)
```

Rule:

- use the stable business key where possible
- do not force artificial numeric primary keys if natural keys are stable enough

### Device

```go
func EncodeDeviceRowKey(slotId uint32, uid string, deviceFlag uint64) []byte
func EncodeDeviceUIDSecondIndexKey(slotId uint32, uid string, deviceFlag uint64) []byte
func ParseDeviceRowKey(key []byte) (slotId uint32, uid string, deviceFlag uint64, err error)
```

### Conversation

```go
func EncodeConversationRowKey(slotId uint32, uid, channelId string, channelType uint8) []byte
func EncodeConversationUpdatedAtSecondIndexKey(slotId uint32, uid string, updatedAt uint64, channelId string, channelType uint8) []byte
func ParseConversationRowKey(key []byte) (slotId uint32, uid, channelId string, channelType uint8, err error)
```

Conversation is where `v3` must be strict:

- slot is `uid slot`
- all conversation indexes must use that same slot prefix

### Channel Info

```go
func EncodeChannelInfoRowKey(slotId uint32, channelId string, channelType uint8) []byte
func EncodeChannelInfoCreatedAtSecondIndexKey(slotId uint32, createdAt uint64, channelId string, channelType uint8) []byte
func ParseChannelInfoRowKey(key []byte) (slotId uint32, channelId string, channelType uint8, err error)
```

### Subscriber / Allowlist / Denylist

```go
func EncodeSubscriberRowKey(slotId uint32, channelId string, channelType uint8, uid string) []byte
func EncodeSubscriberUIDIndexKey(slotId uint32, channelId string, channelType uint8, uid string) []byte

func EncodeAllowlistRowKey(slotId uint32, channelId string, channelType uint8, uid string) []byte
func EncodeDenylistRowKey(slotId uint32, channelId string, channelType uint8, uid string) []byte
```

### Channel Cluster Config

```go
func EncodeChannelClusterConfigRowKey(slotId uint32, channelId string, channelType uint8) []byte
func EncodeChannelClusterConfigLeaderSecondIndexKey(slotId uint32, leaderId uint64, channelId string, channelType uint8) []byte
```

### Message Event State

```go
func EncodeMessageEventStateRowKey(slotId uint32, channelId string, channelType uint8, clientMsgNo, eventKey string) []byte
func EncodeMessageEventSeqAuxKey(slotId uint32, channelId string, channelType uint8, clientMsgNo string) []byte
func ParseMessageEventStateRowKey(key []byte) (slotId uint32, channelId string, channelType uint8, clientMsgNo, eventKey string, err error)
```

## Raw Decode Helper

The package should include a generic decoder for diagnostics.

```go
func Decode(key []byte) (DecodedKey, error)
```

This is useful for:

- scan dumps
- corruption reports
- migration validation

Even if business code never uses it, operations and tests will.

## Range Helper Draft

Range helpers should be explicit and table-aware.

```go
func SlotRange(slotId uint32) (lower, upper []byte)
func SlotScopeRange(slotId uint32, scope ScopeType) (lower, upper []byte)
func SlotTableRange(slotId uint32, scope ScopeType, table TableID) (lower, upper []byte)
func SlotTableRanges(slotId uint32, table TableID) []ScopedRange
```

The minimum required first version is:

- `SlotRange`
- `SlotScopeRange`
- `SlotTableRange`
- one row-key encoder per initial domain

## Testing Contract

Every builder should have tests for:

1. encode -> parse roundtrip
2. prefix ordering
3. lower/upper bound correctness
4. slot separation
5. table separation

Example expectations:

- all keys for `slot 7` sort before all keys for `slot 8`
- all row/index/aux keys of `slot 7` sort before the chosen upper bound for that slot
- `SlotRange(7)` does not overlap `SlotRange(8)`

## Migration Helper Contract

This package should also support migration tools.

Recommended helper:

```go
type LegacyTranslator interface {
    TranslateV2KeyValue(v2Key, v2Value []byte) (slotId uint32, v3Key, v3Value []byte, ok bool, err error)
}
```

This does not necessarily belong in the runtime package, but the `key` package should be designed so such a translator is easy to implement.

## What Not to Build First

Do not start with:

- a fully generic reflection-based encoder
- dynamic schema registration
- every domain at once

Start with:

1. prefix helpers
2. row key builders
3. index key builders for one or two core domains
4. roundtrip tests

## Recommended First Implementation Slice

If implementation starts after this draft, the first slice should be:

1. `prefix.go`
2. `table.go`
3. `common.go`
4. `conversation.go`
5. `channel.go`

Why these first:

- `conversation` is the main ownership risk
- `channel` is the main channel-slot authoritative domain
- both exercise row and index patterns
