# Legacy wkdb Replacement Plan

## Goal

Remove the runtime dependency on legacy `pkg/wkdb/v2` and make `pkg/wkdb/v3`
the only active storage system for the responsibilities it should own.

This does not mean pushing every old `wkdb.DB` method into `v3`.
It means defining the correct storage boundaries and deleting the hybrid
fallback path.

## Target Boundaries

The intended end state should be split into explicit scopes:

- `SlotDB`
  - user
  - device
  - conversation
  - channel info
  - subscriber
  - allowlist
  - denylist
  - channel cluster config
  - message event state and sequence
- `MetaDB`
  - system UIDs
  - testers
  - plugins
  - plugin users
- `LocalDB`
  - notify queue
  - other node-local operational state
- `ChannelLogStore`
  - channel message log
  - channel last message sequence
  - channel raft-related auxiliary state
- `RaftMetadataStore`
  - leader term sequence and similar raft metadata if it remains separate

The important point is that `channel message log` and `slot state` should not be
forced into the same storage API just to preserve legacy compatibility.

## Work Items

### 1. Stop Using legacy `wkdb.DB` As The Main Injection Contract

Replace broad `wkdb.DB` injection with narrower interfaces.

Primary packages to change:

- `pkg/cluster/cluster`
- `pkg/cluster/store`
- `pkg/cluster/channel`
- `pkg/cluster/icluster`

Exit criteria:

- no central runtime type requires the flat legacy `wkdb.DB` surface

Current progress:

- `pkg/cluster/channel` already consumes a narrowed channel-log interface
- `pkg/cluster/store` now isolates legacy-backed ID allocation behind a
  dedicated `PrimaryKeyAllocator`
- `pkg/cluster/store` now uses a dedicated `UserDeviceStore` for user/device
  reads and apply writes
- `pkg/cluster/store` now uses a dedicated `ConversationStore` for
  conversation reads and apply writes
- `pkg/cluster/store` now uses a dedicated `ChannelStateStore` for
  channel/subscriber/allowlist/denylist reads and apply writes
- `pkg/cluster/store` now uses a dedicated `ChannelClusterConfigStore` for
  channel-cluster-config reads and apply writes
- `pkg/cluster/store` now uses a dedicated `MessageQueryStore` for channel-scoped
  message queries
- `pkg/cluster/store` now uses a dedicated `MessageIndexStore` for client-msg-no
  and user-last-msg-seq lookups
- `pkg/cluster/store` now composes recent-message batch helpers directly over
  `MessageQueryStore`
- `pkg/cluster/store` now uses a dedicated `MessageSearchStore` boundary for
  legacy search/count APIs
- `pkg/cluster/store` now uses a dedicated `NotifyQueueStore` for local notify
  queue operations
- `pkg/cluster/store` now uses a dedicated `MetaStore` for system uid / tester / plugin APIs
- `pkg/cluster/store` now uses a dedicated `AdminSearchStore` for cluster admin search APIs
- hybrid runtime construction and DB open/close lifecycle now sit under
  `pkg/cluster/store`, so `pkg/cluster/cluster.Server` no longer holds a raw
  compat DB field
- `pkg/cluster/cluster` now injects explicit primary-key/user/conversation/
  channel/config/message-query/message-index/message-batch/message-search/
  notify-queue/message-event/meta/admin-search boundaries plus a dedicated
  storage lifecycle into `store` instead of passing `WithDB(...)`
- `pkg/cluster/store` no longer derives lifecycle or slot-snapshot support from
  a hidden raw compat DB fallback; those capabilities are now bound explicitly
  through `WithHybridRuntime(...)`, `WithStorageLifecycle(...)`, and
  `WithSlotSnapshotBackend(...)`
- `pkg/cluster/cluster` now loads channel-cluster-config state through `store`
  instead of reading it from the raw compat DB
- the old `Store.DB()` / `icluster.WKDB()` full-DB escape hatches have been removed

### 2. Implement `MetaDB`

Provide real implementations for:

- `SystemUIDStore`
- `TesterStore`
- `PluginStore`
- `PluginUserStore`

Expected result:

- no slot-0 convention for cluster-global state
- no fallback to legacy plugin/tester/system-uid tables

Exit criteria:

- `PebbleDB.Meta()` is no longer backed by unsupported stores

Current progress:

- store/runtime callers no longer need the full `wkdb.DB` surface for
  system-uid, tester, plugin, and plugin-user APIs
- underlying execution still relies on the compat/hybrid layer
- the old slot-0 proposal convention is now isolated behind an explicit
  meta-command proposer instead of being scattered across business methods

### 3. Implement `LocalDB`

Provide a real implementation for:

- `NotifyQueueStore`

Expected result:

- webhook queue no longer depends on legacy `wkdb`
- local operational state is separated from slot-authoritative state

Exit criteria:

- `PebbleDB.Local()` is no longer backed by unsupported stores

Current progress:

- `pkg/cluster/store` now owns a dedicated `NotifyQueueStore` dependency
- the main cluster/store runtime binds notify-queue operations through an
  explicit local-store wrapper over `wkdb/v3` instead of routing them through
  the broader message-store boundary

### 4. Extract The Message Domain From legacy `wkdb`

Do not treat this as a `SlotDB` migration.

Instead, move the following into a dedicated channel/message storage boundary:

- append/load/truncate channel messages
- last message sequence
- channel-applied or channel-common state if retained
- related query helpers

Primary packages to change:

- `pkg/cluster/channel`
- `pkg/cluster/store/message.go`
- message query APIs in cluster and internal layers

Current progress:

- `pkg/cluster/channel` already consumes `ChannelLogStore`
- `pkg/cluster/store` now also owns a dedicated `MessageQueryStore` dependency
- `pkg/cluster/store` now also owns a dedicated `MessageIndexStore` dependency
- `pkg/cluster/store` now also owns a dedicated `MessageSearchStore`
  dependency
- `pkg/cluster/store` now also owns a dedicated `NotifyQueueStore` dependency
- `pkg/cluster/store` now also owns a dedicated `MessageEventStore` dependency
- legacy runtime bindings now use separate `LegacyChannelLogStore` and
  `LegacyMessageQueryStore` adapters instead of one shared message-store wrapper,
  and message index/search helpers are isolated behind
  `LegacyMessageIndexStore` / `LegacyMessageSearchStore`
- notify-queue APIs have been split back out of `MessageQueryStore` into a dedicated
  `NotifyQueueStore` under `LocalDB`
- message-event runtime wiring now goes through an explicit
  `SlotMessageEventStore` over `wkdb/v3` instead of binding the whole
  `HybridDB` as the event-store dependency
- runtime storage is still backed by legacy message tables, so this is boundary
  extraction, not the final storage cutover

Important constraint:

- keep node-local notify queue ownership under `LocalDB`; do not pull it back
  into the channel-log/message boundary just because legacy `wkdb.DB` exposed it

Exit criteria:

- no runtime message read/write path depends on legacy `wkdb.MessageDB`

### 5. Decide The Final Message Event API

Current state:

- v3 stores message-event state and seq
- compatibility methods still emulate old DB behavior above v3

Decision needed:

- keep a compat wrapper as a stable service boundary
- or change callers to use the v3 state/seq model directly

Current progress:

- store/runtime callers no longer need the full `wkdb.DB` surface for
  message-event operations
- the main cluster/store runtime now binds message-event APIs through an
  explicit slot-backed store over `wkdb/v3`; `HybridDB` remains only as a
  compat surface for legacy callers
- underlying semantics are still provided by the compat layer over v3 state/seq

Exit criteria:

- message-event callers stop depending on the old `MessageEventDB` contract

### 6. Remove Compatibility Scans Masquerading As DB APIs

Historically `HybridDB` performed whole-slot iteration for old-style queries such as:

- `SearchUser`
- `SearchDevice`
- `SearchChannels`
- `SearchConversation`
- `SearchChannelClusterConfig`
- `GetChannelConversationLocalUsers`

These should move to explicit service-layer aggregation or projection logic.

Exit criteria:

- no compatibility method requires full-slot fan-out just to imitate legacy DB shape

Current progress:

- cluster admin endpoints for user, device, channel, conversation, and channel
  cluster-config search now go through an explicit store-level admin-search
  aggregator over `wkdb/v3`
- `GetChannelConversationLocalUsers` now prefers explicit store-level
  conversation aggregation instead of the `HybridDB` compat helper
- the old `HybridDB` compatibility scan/count overrides for user, device,
  channel, conversation, and channel-cluster-config fan-out have been removed
- the result-shaping helpers those scans used are now shared at the store layer,
  so explicit aggregators no longer depend on `HybridDB` internals
- store-level fallback methods still keep legacy-shaped query APIs for
  non-hybrid or degraded cases

### 7. Replace legacy Helper Facilities

Still-owned legacy helpers include:

- `NextPrimaryKey()`
- legacy shard grouping helpers
- `wkdb.BatchDB` helper usage in slot storage

Required outcome:

- dedicated ID allocator where needed
- no business logic depends on old DB shard semantics
- no runtime component depends on legacy `BatchDB`

Current progress:

- `pkg/cluster/store` primary-key allocator boundary now only carries
  `NextPrimaryKey()`; legacy shard-grouping helpers are no longer exposed
  through `store`
- the main `pkg/cluster/store` runtime now binds that boundary to a dedicated
  snowflake allocator instead of sourcing IDs from legacy `wkdb`
- API-layer recent-message batching no longer groups channels by legacy DB
  shard; that concern is now confined to the remaining legacy message-store
  internals

Exit criteria:

- no runtime helper path requires old `pkg/wkdb/v2` internals

### 8. Wire Real Restore Maintenance

Slot snapshot import already exists, but maintenance callbacks are still not
connected to real cache invalidation and rebuild behavior.

Required behavior after restore:

1. block or serialize slot writes as needed
2. replace slot KV
3. clear related caches
4. rebuild derived state
5. resume traffic

Exit criteria:

- restore path is operationally complete, not only storage-complete

### 9. Build Migration And Cutover Tooling

Before removing legacy startup:

- migrate old data into new stores
- validate migrated data
- support staged rollout
- define rollback behavior

Recommended approach:

- start with offline migration for correctness
- add temporary bridges only if rollout pressure requires them

Exit criteria:

- production migration can be rehearsed and validated without legacy fallback

### 10. Remove `HybridDB` And legacy Startup

Only after the previous items are complete:

- delete legacy `wkdb.NewWukongDB(...)` startup path
- delete `HybridDB` fallback behavior
- stop opening `data/db`

Exit criteria:

- runtime opens only v3-owned storage backends

## Recommended Order

1. Freeze the target boundaries:
   `SlotDB + MetaDB + LocalDB + ChannelLogStore`
2. Replace `wkdb.DB` injection with narrower interfaces.
3. Implement `MetaDB` and `LocalDB`.
4. Extract channel message storage out of legacy `wkdb`.
5. Finalize message-event service shape.
6. Remove compatibility scans and legacy helpers.
7. Add migration, verification, and cutover tooling.
8. Delete `HybridDB` fallback and legacy startup.

## Exit Checklist

The legacy runtime can be considered removable only when all items below are true:

- startup no longer calls `wkdb.NewWukongDB(...)`
- `HybridDB` no longer embeds a legacy fallback
- `pkg/cluster/channel` no longer needs legacy message or term APIs
- plugin APIs no longer directly call legacy plugin DB methods
- `MetaDB` and `LocalDB` are real, not unsupported placeholders
- slot snapshot restore rebuilds derived state correctly
- no runtime flow depends on `data/db`

## Notes

- conversation ownership cleanup is already largely in place and should be kept
  as a hard invariant
- channel message storage should remain outside slot snapshot scope
- "fully replace legacy wkdb" should be understood as a runtime cutover task,
  not as a promise to preserve the old monolithic DB interface forever
