# wkdb v3 Architecture Notes

## Status

This directory contains `wkdb v3` architecture notes, migration status, and
implementation-adjacent design documents.

The parent package `pkg/wkdb/v3` already contains working code, but the overall
system is still in a hybrid `v2 + v3` runtime state. `v3` is not yet a complete
runtime replacement for legacy `pkg/wkdb/v2`.

Related documents:

- `README.md`: overall v3 direction and document index
- `DATA_OWNERSHIP.md`: v2 to v3 ownership and placement mapping
- `KEYSPACE.md`: proposed slot-native keyspace and bucket layout
- `API.md`: proposed `wkdb v3` interfaces and layering
- `MIGRATION_STATUS.md`: current hybrid runtime status and migration matrix
- `LEGACY_REPLACEMENT_PLAN.md`: concrete plan to remove legacy `wkdb`
- `key/README.md`: proposed `v3` key builder package design
- `CONVERSATION_CLEANUP.md`: conversation ownership audit and cleanup plan

Current conclusion:

- Keep `raft slot` as the state-machine partition boundary.
- Do not use `one slot = one Pebble DB` by default.
- Use `bucketed slot db` instead:
  - logical partition = `slot`
  - physical partition = `bucket`
  - one bucket maps to one Pebble DB
- Move global state out of slot state machines into a dedicated `meta raft + meta db`.
- Make snapshot/restore operate on `slot` prefix ranges, not on ad hoc object export rules.

## Why v2 Becomes Hard

The main problem in the current `wkdb` layout is not compaction itself.
The real problem is that the partition boundary of the storage layer does not match the partition boundary of the raft state machine.

In `v2`:

- raft uses `slot` as the replicated state-machine boundary
- `wkdb` physically partitions by `shard`
- `wkdb` still derives storage placement from business keys such as `uid` and `channelId`
- some business domains are not consistently owned by one slot

This creates several issues:

1. One slot's state is scattered across multiple Pebble instances.
2. Slot snapshot cannot be implemented as a simple prefix scan or directory checkpoint.
3. Restore has to reconstruct business objects instead of replaying a native storage range.
4. Some domains are modified from different routing paths, so "who owns this state" is unclear.

The most dangerous example is `conversation`:

- some paths route by `uid`
- some paths route by `channelId`
- but the actual stored rows are keyed under `uid`

That means the same logical domain may be modified by different slots, which makes slot snapshot semantically unsafe.

## v3 Goals

`wkdb v3` should satisfy these properties:

1. A slot owns a closed set of state.
2. All data owned by one slot can be described by one storage key range.
3. Snapshot and restore can be implemented without understanding every business table in detail.
4. Global data is not mixed into slot-local state.
5. Storage routing is explicit and does not depend on hidden business-key hashing inside `wkdb`.
6. Physical DB count remains bounded and operationally reasonable.

## Non-Goals

`wkdb v3` does not try to:

- preserve the current implicit shard-routing model
- optimize for zero-refactor compatibility
- use `256 slots = 256 Pebble DBs`
- solve every online migration detail in the first phase

## Core Direction

`wkdb v3` should be redesigned as:

- slot-native keyspace
- bucketed Pebble layout
- explicit routing from upper layers
- dedicated meta state machine for global data

### 1. Slot-Native Keyspace

Every slot-owned record must be encoded under a keyspace that starts with `slotId`.

Conceptually:

```text
slot/<slotId>/<table>/<business-key...>
```

This changes snapshot from:

- "enumerate business objects that happen to belong to slot X"

to:

- "scan the key range for slot X"

That is the main simplification.

### 2. Bucketed Pebble Layout

Do not create one Pebble DB per slot.

Instead:

- define a fixed `bucketCount`, for example `8` or `16`
- map `slotId -> bucketId`
- open one Pebble DB per bucket

Conceptually:

```text
bucketId = slotId % bucketCount
```

Benefits:

- bounded number of Pebble instances
- one slot still maps to exactly one physical DB
- snapshot/restore stays simple because one slot still occupies one prefix range in one DB

### 3. Meta Raft + Meta DB

Global state should not be written into `slot 0` by convention.

Instead, create a dedicated metadata state machine:

- one meta raft group
- one meta DB

This meta state machine owns truly global records such as:

- system UIDs
- tester
- plugin user bindings
- any future cluster-global configuration that is not slot-local

This removes a large class of special cases from slot snapshot.

### 4. Explicit Routing

`wkdb v3` should not decide ownership by hashing `uid` or `channelId` internally.

Upper layers should compute `slotId` first and pass it into `wkdb`.

That means:

- `store` decides routing
- `wkdb` stores data for a specified slot

This is critical because the replication boundary must be explicit and auditable.

## Data Ownership Rules

These ownership rules should be fixed early and treated as architecture constraints.

### Slot-Owned by `uid slot`

These domains should always belong to the slot derived from `uid`:

- `user`
- `device`
- `conversation`

### Slot-Owned by `channel slot`

These domains should always belong to the slot derived from `channelId`:

- `channel info`
- `subscriber`
- `allowlist`
- `denylist`
- `channel cluster config`
- `message event state`
- `message event sequence`

### Meta-Owned

These domains should move to the dedicated meta state machine:

- `system_uids`
- `tester`
- `plugin_user`
- similar future global tables

## Hard Rule for Conversation

`conversation` must be owned only by `uid slot`.

This means:

- every write that changes a user's conversation must be proposed to that user's slot
- no path may update conversation state through `channel slot`

If this rule is not enforced, slot snapshot is not well-defined.

This is the first semantic cleanup that should happen before any real `slot snapshot` implementation is enabled.

## Keyspace Design

The exact binary encoding can be tuned later, but the structure should be fixed early.

Recommended logical pattern:

```text
slot/<slotId>/user/<uid>
slot/<slotId>/device/<uid>/<deviceFlag>
slot/<slotId>/conversation/<uid>/<channelId>/<channelType>

slot/<slotId>/channel/<channelId>/<channelType>
slot/<slotId>/subscriber/<channelId>/<channelType>/<uid>
slot/<slotId>/allowlist/<channelId>/<channelType>/<uid>
slot/<slotId>/denylist/<channelId>/<channelType>/<uid>
slot/<slotId>/channel_cluster_config/<channelId>/<channelType>
slot/<slotId>/message_event_state/<channelId>/<channelType>/<clientMsgNo>/<eventKey>
slot/<slotId>/message_event_seq/<channelId>/<channelType>/<clientMsgNo>
```

Binary encoding should still use compact table IDs and typed key builders, but the important property is:

- all slot-owned records start with the same slot prefix

### Index Rule

Secondary indexes must also include the same `slotId` prefix.

Otherwise:

- the primary data is slot-local
- but the indexes are not

and snapshot/restore becomes inconsistent.

This rule applies to all future index tables as well.

## Snapshot Model

With slot-native keys, the snapshot model becomes much simpler.

### Export

For one slot:

1. resolve `slotId -> bucketId`
2. open a Pebble read snapshot
3. iterate only the key range under the slot prefix
4. serialize raw KV pairs into a snapshot stream

### Restore

For one slot:

1. block writes for that slot
2. delete the slot prefix range
3. import the KV pairs from snapshot
4. clear related caches
5. resume traffic

### Why Raw KV Snapshot Instead of Object Snapshot

Raw KV snapshot is preferred in `v3` because:

- all primary data and indexes are captured together
- snapshot logic does not need table-specific export code
- schema changes remain localized to key/value codecs
- restore semantics are simpler and easier to verify

This is a better fit once the keyspace is truly slot-native.

## Snapshot File Format

The first version should stay minimal.

Recommended layout:

```text
header
record_count
record_1
record_2
...
record_n
```

Each record:

```text
key_len
value_len
key
value
```

Optional enhancements:

- block checksum
- whole-file checksum
- zstd compression

The first version should avoid section-heavy formats unless there is a clear operational need.

## Cache Model

`wkdb v3` should treat cache as rebuildable state.

Therefore:

- snapshot does not include cache entries
- restore clears related caches
- reads refill caches naturally

This keeps restore deterministic and avoids subtle cache/schema coupling.

## Statistics Model

`TotalDB`-style counters should not be considered slot snapshot state unless they are promoted to first-class replicated business state.

For `v3`, the recommended rule is:

- do not include derived counters in slot snapshot
- either rebuild them asynchronously
- or move truly authoritative counters into dedicated replicated metadata if needed

This avoids mixing derived state with source-of-truth state.

## Proposed Interface Direction

The exact method set can change, but the layering should look like this.

### Store Layer

`store` computes ownership and passes `slotId` explicitly.

Examples:

```go
AddUser(slotId uint32, u User) error
UpdateUser(slotId uint32, u User) error
AddDevice(slotId uint32, d Device) error
AddOrUpdateConversations(slotId uint32, uid string, cs []Conversation) error
SaveChannelClusterConfig(slotId uint32, cfg ChannelClusterConfig) error
```

### wkdb v3 Layer

`wkdb v3` resolves `slotId -> bucket DB` and manages slot-prefixed keys.

Possible shape:

```go
type SlotBucketDB interface {
    ExportSlotKV(ctx context.Context, slotId uint32, w io.Writer) error
    ImportSlotKV(ctx context.Context, slotId uint32, r io.Reader) error
    DeleteSlotKV(ctx context.Context, slotId uint32) error
}
```

The main point is not the exact naming.
The main point is:

- slot-aware storage interface
- explicit routing
- prefix-based snapshot

## Operational Tradeoffs

### Advantages

- slot state is closed and explicit
- snapshot becomes prefix-based and verifiable
- restore becomes deterministic
- bucket count stays small enough for operations
- ownership bugs become much easier to detect

### Costs

- key schema needs a real redesign
- all secondary indexes must be migrated
- old routing assumptions in `wkdb` must be removed
- global state needs to move to a dedicated meta state machine
- migration tooling will be necessary

## Migration Strategy

This should not be a one-shot rewrite.

Recommended phases:

### Phase 1: Ownership Cleanup

- freeze the ownership rules
- make `conversation` strictly `uid slot` owned
- stop writing global data through slot-local conventions

### Phase 2: v3 Keyspace and Bucket Layout

- introduce `pkg/wkdb/v3`
- add slot-prefixed key builders
- add bucket DB manager
- add slot-aware storage interfaces

### Phase 3: Dual-Write or Migration Tooling

Choose one:

- offline migration tool from v2 data to v3
- or a temporary dual-write bridge for controlled rollout

For correctness, offline migration is usually simpler than long-lived dual-write.

### Phase 4: Slot Snapshot Support

- export slot KV by prefix
- import slot KV by prefix
- wire snapshot callbacks to `slot` raft storage

### Phase 5: Enable Compaction

Only after:

- ownership rules are clean
- restore is validated
- restart recovery is verified

should distributed compaction be enabled.

## Open Questions

These points still need decisions before implementation:

1. Whether message-event historical rows should remain projection-only or evolve into append-only event logs.
2. Whether any global counters must become authoritative replicated state.
3. Whether migration should be offline-only for the first production rollout.
4. Whether a dedicated bucket can be reserved for very hot slots in the future.

## Recommended Next Step

The next design/implementation step should be:

1. document and enforce the ownership rules
2. especially fix `conversation` to be `uid slot` only
3. then define `v3` key builders and bucket manager

Without ownership cleanup first, snapshot design will remain unstable.
