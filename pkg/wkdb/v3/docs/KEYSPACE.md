# wkdb v3 Keyspace Design

## Purpose

This document describes the proposed `wkdb v3` keyspace model for slot-native storage.

The design target is:

- one slot owns one closed key range
- one slot maps to one physical bucket DB
- all slot primary rows and slot secondary indexes share the same slot prefix
- `slot snapshot` becomes a prefix export/import operation

## Design Constraints

`wkdb v3` key design must satisfy these invariants.

### Invariant 1: Slot First

Every slot-owned key must begin with the same slot prefix.

This is the main property that makes snapshot simple.

### Invariant 2: Primary and Index Together

For a slot-owned table:

- primary rows
- unique indexes
- non-unique indexes
- authoritative auxiliary records

must all carry the same slot prefix.

Otherwise restore will replay primary data without the matching indexes.

### Invariant 3: Bucket Is Physical, Not Logical

The key should identify the slot.
The bucket is only how the process chooses which Pebble DB to open.

That means:

- `slotId` is part of the logical keyspace
- `bucketId` is not required to be encoded into every key

### Invariant 4: Meta and Local Data Must Not Collide

Global metadata and local operational state should live in separate key classes and preferably separate DBs.

The slot keyspace must only contain slot-authoritative business state.

## Physical Layout

Recommended physical layout:

```text
pkg/wkdb/v3 data root
  buckets/
    bucket-000/
    bucket-001/
    ...
    bucket-015/
  meta/
```

Mapping:

```text
bucketId = slotId % bucketCount
```

Properties:

- bounded number of Pebble instances
- one slot always lands in one bucket DB
- one bucket DB stores many slot prefixes

## Key Envelope

The exact byte layout can still evolve, but the envelope should be structured.

Recommended binary skeleton:

```text
version(1)
keyClass(1)
slotId(4)
scope(1)
tableId(2)
kind(1)
table-local-key(...)
```

Where:

- `version` identifies the key format version
- `keyClass` separates slot, meta, and local key families
- `slotId` identifies the logical owner
- `scope` separates row/index/aux namespaces inside one slot
- `tableId` identifies the table family
- `kind` distinguishes row/index/aux state

This is only a skeleton.
The exact local key payload depends on each table.

## Key Class Values

Suggested top-level class split:

```text
0x01 = slot-owned key
0x02 = meta-owned key
0x03 = local operational key
```

## Scope Values

Suggested scope split:

```text
0x10 = slot primary data
0x11 = slot unique index
0x12 = slot secondary index
0x13 = slot authoritative auxiliary state

0x20 = meta primary data
0x21 = meta unique index
0x22 = meta secondary index

0x30 = local operational state
```

This keeps range scans and on-disk inspection simple.

## Slot Prefix

The canonical slot prefix should be:

```text
version + keyClass(slot) + slotId
```

For example:

```text
[01][01][00 00 00 25]
```

meaning:

- key version 1
- slot key class
- slot 37

All rows, indexes, and authoritative auxiliary records of slot 37 should begin with that prefix.
Per-scope scans then extend it with the `scope` byte.

## Table Families

Suggested table family IDs should remain stable and compact.
They do not need to reuse the old `v2` IDs exactly, but reusing them where reasonable reduces migration confusion.

Example conceptual families:

```text
0x0201 user
0x0301 device
0x0401 subscriber
0x0501 subscriber_channel_relation
0x0601 channel_info
0x0701 denylist
0x0801 allowlist
0x0901 conversation
0x0B01 channel_cluster_config
0x1901 message_event_state
0x1A01 message_event_seq
```

Channel-message and local operational tables should not be mixed into the slot keyspace unless a separate design explicitly moves them there.

## Key Categories

Within one table family, `kind` should distinguish the record role.

Suggested kinds:

```text
0x01 = primary row
0x02 = unique index
0x03 = secondary index
0x04 = auxiliary authoritative state
```

Examples:

- conversation row
- conversation channel index
- subscriber primary membership row
- subscriber inverse relation index
- message-event sequence counter

## Logical Examples

These examples use readable forms, not exact bytes.

### User Slot State

```text
slot/17/user/row/<uid>
slot/17/user/secidx/created_at/<ts>/<uid_pk>
slot/17/device/row/<uid>/<deviceFlag>
slot/17/device/secidx/uid/<uid>/<device_pk>
slot/17/conversation/row/<uid>/<channelId>/<channelType>
slot/17/conversation/secidx/updated_at/<uid>/<ts>/<conversation_pk>
```

### Channel Slot State

```text
slot/42/channel_info/row/<channelId>/<channelType>
slot/42/subscriber/row/<channelId>/<channelType>/<uid>
slot/42/subscriber/secidx/uid/<channelId>/<channelType>/<uid>
slot/42/allowlist/row/<channelId>/<channelType>/<uid>
slot/42/denylist/row/<channelId>/<channelType>/<uid>
slot/42/channel_cluster_config/row/<channelId>/<channelType>
slot/42/message_event_state/row/<channelId>/<channelType>/<clientMsgNo>/<eventKey>
slot/42/message_event_seq/aux/<channelId>/<channelType>/<clientMsgNo>
```

## Range Design

The key layout should support these range operations efficiently.

### Full Slot Export

Prefix:

```text
slot-prefix(slotId)
```

Used by:

- snapshot export
- snapshot delete before restore
- slot-level debugging

### Slot and Table Export

Range set:

```text
slot-table-prefix(slotId, primary, tableId)
slot-table-prefix(slotId, index, tableId)
slot-table-prefix(slotId, second-index, tableId)
slot-table-prefix(slotId, aux, tableId)
```

Used by:

- table-specific verification
- migration tooling
- narrow repair operations

### Slot and Index Export

Prefix:

```text
slot-scope-prefix(slotId, index-scope) + tableId
```

Used by:

- index validation
- rebuild tooling

## Snapshot Implications

With the above layout, `slot snapshot` becomes:

1. find the bucket DB by `slotId`
2. open a Pebble read snapshot
3. scan the slot prefix range
4. emit raw KV records

`slot restore` becomes:

1. resolve bucket DB by `slotId`
2. delete the slot prefix range
3. replay raw KV records

This is much simpler than object-level export because:

- rows and indexes move together
- auxiliary authoritative state moves together
- restore does not need domain-specific rebuild logic

## Meta Keyspace

Meta data should use the same envelope idea, but with `meta` key class and no `slotId`.

Conceptually:

```text
version
keyClass(meta)
scope
tableId
kind
table-local-key
```

Meta families likely include:

- system uid
- tester
- plugin
- plugin user
- future cluster-global settings

Meta state is not part of `slot snapshot`.

## Local Operational Keyspace

Node-local and operational state should use a separate local key class or separate DB.

Examples:

- webhook notify queue
- temporary migration markers
- node-local rebuild progress

This prevents accidental inclusion in replicated snapshot flows.

## What Must Not Reappear in v3

The following `v2` patterns should not reappear:

### 1. Hidden Routing by Business Key

`wkdb` must not hash `uid` or `channelId` to decide ownership internally.

Ownership must be explicit via `slotId`.

### 2. Indexes Without Slot Prefix

A slot-owned table must not have indexes outside the slot range.

### 3. Mixed Channel and Slot Semantics

Channel-message storage and slot business state should not share one implicit key model.

If they both exist, they should be modeled as separate replicated domains.

### 4. Authoritative State Stored as Local Projection

Derived structures like `conversation_local_user` should not silently become authoritative without explicit domain ownership.

## Migration Notes

`v2 -> v3` migration should treat key translation as deterministic rewrite:

1. decode `v2` rows
2. determine owner slot explicitly
3. encode `v3` slot-prefixed rows and indexes
4. write them into the correct bucket DB

This migration is much easier once the ownership mapping document is accepted.

## Recommended Next Step

After this keyspace design is accepted:

1. define `pkg/wkdb/v3/key/` builders
2. define `bucket manager`
3. define `slot prefix` helper APIs
4. begin with one domain, preferably `conversation` after ownership cleanup
