# wkdb v3 API Draft

## Purpose

This document proposes the interface boundaries for `wkdb v3`.

The primary goal is not to finalize every method today.
The goal is to make these boundaries stable:

1. who computes ownership
2. who resolves physical placement
3. who encodes keys
4. who owns snapshot/export/import
5. how business code talks to `wkdb v3`

## Design Principles

### 1. Explicit Slot Routing

`wkdb v3` must not infer authoritative ownership from `uid` or `channelId`.

Instead:

- upper layers compute `slotId`
- `wkdb v3` receives `slotId` explicitly

### 2. Separate Logical Scope from Physical Placement

Logical owner:

- `slotId`

Physical placement:

- `bucketId`

The caller should care about `slotId`.
Only the storage engine should care about `bucketId`.

### 3. Domain APIs on Top of Slot Scope

Business code should not directly manipulate raw bucket DB handles.

Preferred layering:

- routing layer
- slot-scoped domain stores
- low-level KV and snapshot layer

### 4. Snapshot Belongs to Storage Scope

Snapshot/export/import should operate at `slot` scope, not at random business-object scope.

That means:

- `ExportSlotKV`
- `ImportSlotKV`
- `DeleteSlotKV`

belong to the storage layer, not to each domain repo.

## Recommended Layering

### Layer 1: Route Resolution

Owned outside `wkdb`, usually by `store` or a dedicated routing helper.

Responsibilities:

- `uid -> slotId`
- `channelId -> slotId`
- choose whether a request goes to `slot`, `channel`, or `meta`

### Layer 2: wkdb v3 Entry Point

Top-level entry point for storage capabilities.

Responsibilities:

- open/close buckets and meta db
- expose slot/meta/local stores
- expose snapshot and maintenance capabilities

### Layer 3: Slot-Scoped Domain Stores

Typed domain APIs that operate inside one logical slot.

Responsibilities:

- CRUD for slot-owned domains
- searches limited to one slot when possible
- slot-aware index updates

### Layer 4: Low-Level KV and Snapshot

Responsibilities:

- resolve `slotId -> bucket`
- build prefixes
- iterate, export, import, delete by slot prefix
- hide raw Pebble details

## Top-Level Interface Draft

One possible top-level interface:

```go
type DB interface {
    Open() error
    Close() error

    Slots() SlotDB
    Meta() MetaDB
    Local() LocalDB

    Snapshotter() SlotSnapshotter
    Maintenance() Maintenance
}
```

This is intentionally smaller than current `wkdb.DB`.

The current `wkdb.DB` is a flat capability surface.
`v3` should favor grouped capabilities by scope.

## SlotDB Draft

`SlotDB` is the main business entry point for slot-owned state.

```go
type SlotDB interface {
    User(slotId uint32) UserStore
    Device(slotId uint32) DeviceStore
    Conversation(slotId uint32) ConversationStore

    Channel(slotId uint32) ChannelStore
    Subscriber(slotId uint32) SubscriberStore
    Allowlist(slotId uint32) AllowlistStore
    Denylist(slotId uint32) DenylistStore
    ChannelClusterConfig(slotId uint32) ChannelClusterConfigStore
    MessageEvent(slotId uint32) MessageEventStore
}
```

Alternative shape:

```go
type SlotDB interface {
    Scope(slotId uint32) SlotScope
}
```

with:

```go
type SlotScope interface {
    User() UserStore
    Device() DeviceStore
    Conversation() ConversationStore
    Channel() ChannelStore
    ...
}
```

I prefer `Scope(slotId)` because:

- it avoids repeating `slotId` on every method lookup
- it fits the mental model of "operate inside one authoritative slot"

## Preferred SlotScope Draft

```go
type SlotScope interface {
    SlotID() uint32

    Users() UserStore
    Devices() DeviceStore
    Conversations() ConversationStore

    Channels() ChannelStore
    Subscribers() SubscriberStore
    Allowlists() AllowlistStore
    Denylists() DenylistStore
    ChannelClusterConfigs() ChannelClusterConfigStore
    MessageEvents() MessageEventStore

    Raw() SlotRawKV
}
```

`Raw()` exists for:

- migrations
- repair tooling
- snapshot verification

but should not be the main business API.

## MetaDB Draft

Meta-owned state should be explicit.

```go
type MetaDB interface {
    SystemUIDs() SystemUIDStore
    Testers() TesterStore
    Plugins() PluginStore
    PluginUsers() PluginUserStore
}
```

If some plugin data is later decided to be local-only rather than cluster-global, `MetaDB` should be adjusted accordingly.

## LocalDB Draft

Local-only operational state should be isolated.

```go
type LocalDB interface {
    NotifyQueue() NotifyQueueStore
    MigrationState() MigrationStateStore
}
```

This prevents accidental coupling between operational queues and replicated slot state.

## Snapshot Interface Draft

Snapshot should be explicitly slot-scoped.

```go
type SlotSnapshotter interface {
    ExportSlotKV(ctx context.Context, slotId uint32, w io.Writer) (SlotSnapshotMeta, error)
    ImportSlotKV(ctx context.Context, slotId uint32, r io.Reader, meta SlotSnapshotMeta) error
    DeleteSlotKV(ctx context.Context, slotId uint32) error
    VerifySlotKV(ctx context.Context, slotId uint32, r io.Reader, meta SlotSnapshotMeta) error
}
```

Recommended metadata:

```go
type SlotSnapshotMeta struct {
    SlotID       uint32
    Format       uint16
    ExportedAt   int64
    RecordCount  uint64
    ByteSize     uint64
    Checksum     []byte
}
```

`VerifySlotKV` is optional for first implementation but useful for:

- migration tooling
- offline validation
- test coverage

For maintainability, the raw snapshot stream logic should stay separate from the storage-engine adapter.

One practical split is:

```go
type SlotSnapshotScanner interface {
    ScanSlotKV(ctx context.Context, slotId uint32, fn func(key, value []byte) error) error
}

type SlotSnapshotReplacer interface {
    ReplaceSlotKV(ctx context.Context, slotId uint32, fn func(put func(key, value []byte) error) error) error
}
```

Then:

- the generic snapshot codec handles stream framing, checksum, and key validation
- the Pebble adapter only handles slot scan / slot replace mechanics
- unit tests can cover snapshot semantics without opening a real DB

## Maintenance Interface Draft

`Maintenance` should contain operational functions that do not belong to business CRUD.

```go
type Maintenance interface {
    BucketCount() int
    BucketForSlot(slotId uint32) uint32
    ClearSlotCaches(slotId uint32)
    RebuildDerivedState(ctx context.Context, slotId uint32) error
}
```

This keeps maintenance work out of business repos.

## Low-Level Bucket Interfaces

The typed APIs above should be backed by low-level bucket abstractions.

```go
type BucketResolver interface {
    BucketForSlot(slotId uint32) Bucket
}

type Bucket interface {
    ID() uint32
    DB() KVDB
}
```

The rest of the codebase should not need to know whether the bucket wraps:

- Pebble
- a test fake
- or a future alternative engine

The first concrete implementation can stay minimal:

```go
type PebbleBucketManager interface {
    Open() error
    Close() error
    BucketCount() uint32
    BucketForSlot(slotId uint32) uint32
    Bucket(slotId uint32) (PebbleBucket, error)
}
```

where each bucket only needs to expose:

- a `*pebble.DB`
- a `PebbleSlotSnapshotStore`

This keeps bucket lifecycle, slot routing, and snapshot adapter wiring in one place without leaking business semantics into the manager.

## KVDB Draft

Keep the internal low-level KV surface minimal.

```go
type KVDB interface {
    Get(key []byte) ([]byte, error)
    Set(key, value []byte) error
    Delete(key []byte) error
    DeleteRange(start, end []byte) error
    NewBatch() Batch
    NewIter(lower, upper []byte) Iterator
    NewSnapshot() ReadSnapshot
}
```

This should remain internal to `v3`.

Business code should generally not see it.

## Domain Store Drafts

Do not copy the flat `v2` interface style into `v3` unchanged.

The main changes should be:

- slot scoping happens outside the store method
- stores stop owning routing decisions
- writes only expose operations valid inside that scope

### UserStore

```go
type UserStore interface {
    Get(uid string) (User, error)
    Exists(uid string) (bool, error)
    Put(u User) error
    Delete(uid string) error
    Search(req UserSearchReq) ([]User, error)
}
```

### DeviceStore

```go
type DeviceStore interface {
    Get(uid string, deviceFlag uint64) (Device, error)
    ListByUID(uid string) ([]Device, error)
    Put(d Device) error
    Delete(uid string, deviceFlag uint64) error
    Search(req DeviceSearchReq) ([]Device, error)
}
```

### ConversationStore

```go
type ConversationStore interface {
    Get(uid, channelId string, channelType uint8) (Conversation, error)
    ListByUID(uid string) ([]Conversation, error)
    Put(uid string, cs []Conversation) error
    Delete(uid, channelId string, channelType uint8) error
    DeleteBatch(uid string, channels []Channel) error
    UpdateIfSeqGreater(uid, channelId string, channelType uint8, seq uint64) error
    UpdateDeletedAt(uid, channelId string, channelType uint8, seq uint64) error
    Search(req ConversationSearchReq) ([]Conversation, error)
}
```

Important:

- this API only makes sense for `uid slot`
- no `channel slot` mutation path should be introduced here

### ChannelStore

```go
type ChannelStore interface {
    Get(channelId string, channelType uint8) (ChannelInfo, error)
    Exists(channelId string, channelType uint8) (bool, error)
    Put(info ChannelInfo) error
    Delete(channelId string, channelType uint8) error
    Search(req ChannelSearchReq) ([]ChannelInfo, error)
}
```

### MessageEventStore

```go
type MessageEventStore interface {
    GetState(channelId string, channelType uint8, clientMsgNo, eventKey string) (*MessageEventState, error)
    GetStates(channelId string, channelType uint8, clientMsgNo string) ([]MessageEventState, error)
    PutState(state MessageEventState) error
    DeleteStates(channelId string, channelType uint8, clientMsgNo string) error

    GetSeq(channelId string, channelType uint8, clientMsgNo string) (uint64, error)
    SetSeq(channelId string, channelType uint8, clientMsgNo string, seq uint64) error
}
```

Note:

- this draft models current persisted truth more accurately than the v2 projection API
- append/reduce behavior can still live in a higher service layer

## Search API Direction

`v2` has many whole-db search methods.
In `v3`, search needs explicit scope.

Recommended split:

1. slot-local search
2. meta-local search
3. cluster/global aggregation outside `wkdb`

That means:

- `wkdb v3` should not pretend one storage call can transparently search every authoritative domain in one shot
- higher layers should aggregate across slots if they want cluster-wide search

## Compatibility Layer Direction

If a transitional bridge is needed, add an adapter:

```go
type V2Compat struct {
    routes RouteResolver
    db     v3.DB
}
```

This adapter may:

- compute `slotId`
- pick `slot`, `meta`, or `local`
- call the relevant v3 store

This is preferable to polluting `v3` with `v2` routing assumptions.

## What Should Not Be in v3 Root API

The following patterns should be avoided:

### 1. Flat Giant Interface

Do not recreate current `wkdb.DB` as one giant interface with dozens of unrelated methods.

### 2. Hidden Default Scope

Do not expose business methods that silently use:

- default shard
- slot 0
- channel hash
- uid hash

without the caller being aware of the ownership rule.

### 3. Mixed Slot and Channel Storage

Do not mix channel-message storage interfaces into `SlotDB`.

Channel-message storage belongs to a separate replication domain.

## Recommended Next Step

After this API draft is accepted:

1. define `pkg/wkdb/v3/key` package contract
2. define `bucket manager` contract
3. choose `Scope(slotId)` vs `Store(slotId)` style
4. start implementation from one slot-owned domain after conversation routing cleanup
