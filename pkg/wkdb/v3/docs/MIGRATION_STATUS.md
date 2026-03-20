# wkdb v3 Migration Status

## Purpose

This document records the current migration status from legacy `pkg/wkdb/v2` to
`pkg/wkdb/v3`.

It is intentionally about the code that exists today, not only the intended
architecture.

## Current Runtime Model

The system is still running in a hybrid mode.

- startup creates one legacy `wkdb` instance under `data/db`
- startup creates one `wkdb/v3` instance under `data/dbv3`
- both are wrapped by `HybridDB`
- downstream packages still receive one `wkdb.DB`-shaped object

This means the project has not fully switched to `wkdb/v3` yet.

In practical terms:

- slot-owned data is partly served by `wkdb/v3`
- unmigrated capabilities still fall back to legacy `wkdb`
- removing legacy `wkdb` today would break runtime behavior

## What Counts As "Fully Replaced"

`wkdb` should only be considered fully replaced when all of the following are
true:

1. startup no longer creates legacy `wkdb.NewWukongDB(...)`
2. `HybridDB` no longer embeds or depends on a legacy fallback
3. no runtime read or write path depends on `data/db`
4. downstream packages no longer require the flat legacy `wkdb.DB` surface
5. `MetaDB`, `LocalDB`, and channel-message storage all have non-legacy homes

## Capability Matrix

| Capability | Current state | Runtime path today | Notes |
| --- | --- | --- | --- |
| User | migrated | `store.UserDeviceStore -> v3 SlotScope.Users()` for store/runtime reads and apply writes; `store.AdminSearchStore -> v3 SlotScope.Users()` for admin search; `HybridDB -> v3 SlotScope.Users()` for compat reads/writes | store user reads and apply handlers no longer require the full legacy `wkdb.DB` surface; cross-slot admin search no longer lives in `HybridDB`; the old `HybridDB.SearchUser` compat override has been removed |
| Device | migrated | `store.UserDeviceStore -> v3 SlotScope.Devices()` for store/runtime reads and apply writes; `store.AdminSearchStore -> v3 SlotScope.Devices()` for admin search; `HybridDB -> v3 SlotScope.Devices()` for compat reads/writes | store device reads and apply handlers no longer require the full legacy `wkdb.DB` surface; cross-slot admin search no longer lives in `HybridDB`; the old `HybridDB.SearchDevice` compat override has been removed |
| Channel info | migrated | `store.ChannelStateStore -> v3 SlotScope.Channels()` for store/runtime reads and apply writes; `store.AdminSearchStore -> v3 SlotScope.Channels()` for admin search; `HybridDB -> v3 SlotScope.Channels()` for compat reads/writes | `pkg/cluster/store/channel.go` and channel apply handlers no longer depend on the full legacy `wkdb.DB` surface |
| Subscriber | migrated | `store.ChannelStateStore -> v3 SlotScope.Subscribers()` for store/runtime reads and apply writes; `HybridDB -> v3 SlotScope.Subscribers()` for compat reads/writes | slot-owned by `channelId`; store subscriber reads/writes no longer require full `wkdb.DB` |
| Allowlist | migrated | `store.ChannelStateStore -> v3 SlotScope.Allowlists()` for store/runtime reads and apply writes; `HybridDB -> v3 SlotScope.Allowlists()` for compat reads/writes | slot-owned by `channelId`; permission and apply paths in `store` no longer require full `wkdb.DB` |
| Denylist | migrated | `store.ChannelStateStore -> v3 SlotScope.Denylists()` for store/runtime reads and apply writes; `HybridDB -> v3 SlotScope.Denylists()` for compat reads/writes | slot-owned by `channelId`; permission and apply paths in `store` no longer require full `wkdb.DB` |
| Conversation rows | migrated | `store.ConversationStore -> v3 SlotScope.Conversations()` for store/runtime reads and apply writes; `store.AdminSearchStore -> v3 SlotScope.Conversations()` for admin search; `HybridDB -> v3 SlotScope.Conversations()` for compat reads/writes | store conversation reads and apply handlers no longer require the full legacy `wkdb.DB` surface; cross-slot admin search no longer lives in `HybridDB`; the old `HybridDB.SearchConversation` compat override has been removed |
| Conversation ownership routing | fixed | store routes by `uid slot` | `UpdateConversationIfSeqGreaterAsync` is no longer channel-routed |
| Channel cluster config | migrated | `store.ChannelClusterConfigStore -> v3 SlotScope.ChannelClusterConfigs()` for store/runtime reads and apply writes; `store.AdminSearchStore -> v3 SlotScope.ChannelClusterConfigs()` for admin search; `HybridDB -> v3 SlotScope.ChannelClusterConfigs()` for compat reads/writes | store channel-cluster-config reads and apply handlers no longer require the full legacy `wkdb.DB` surface; `pkg/cluster/cluster` now reads channel config through `store` instead of the raw compat DB; cross-slot admin search no longer lives in `HybridDB`; the old `HybridDB.SearchChannelClusterConfig` compat override has been removed |
| Message event state/seq | migrated at storage level | `HybridDB -> v3 SlotScope.MessageEvents()` | v3 stores state and seq, not a full old-style DB surface |
| Message event high-level API | partial | `cluster/store -> MessageEventStore(SlotMessageEventStore) -> v3 SlotScope.MessageEvents()` | store runtime now depends on a narrowed message-event contract and no longer needs raw `HybridDB` ownership for message-event APIs, while old append/list/query semantics are still emulated above v3 |
| Channel message log | boundary extracted, storage not migrated | `cluster/server -> ChannelLogStore(LegacyChannelLogStore) -> legacy wkdb.MessageDB` | `pkg/cluster/channel` now depends on an explicit channel-log contract instead of the full compat DB |
| Channel last message seq | boundary extracted, storage not migrated | `cluster/server -> ChannelLogStore(LegacyChannelLogStore) -> legacy wkdb.MessageDB`; `cluster/store -> MessageQueryStore(LegacyMessageQueryStore) -> legacy wkdb.MessageDB` | remains coupled to channel message storage, but runtime access is now split into distinct legacy-backed channel-log and query adapters |
| Message index helpers | boundary extracted, storage not migrated | `cluster/store -> MessageIndexStore(LegacyMessageIndexStore) -> legacy wkdb.MessageDB` | client-msg-no lookup and user-last-msg-seq helpers are now isolated from the channel-scoped query boundary |
| Message batch helpers | service-layer helper | `cluster/store -> MessageQueryStore(LegacyMessageQueryStore) -> legacy wkdb.MessageDB` | `LoadMsgsBatch` no longer has its own runtime binding and now composes over the channel-scoped query boundary in `store` |
| Message search/count | boundary extracted, storage not migrated | `cluster/store -> MessageSearchStore(LegacyMessageSearchStore) -> legacy wkdb.MessageDB` | admin/search-style message APIs are now isolated from the channel-scoped query boundary |
| Channel common/applied index | not migrated | legacy `wkdb` | not part of slot snapshot scope |
| Leader term sequence | not migrated | legacy `wkdb` | raft metadata, not slot business state |
| System UIDs | migrated to v3 meta storage | `cluster/store -> MetaStore -> HybridDB -> v3 Meta().SystemUIDs()` | store runtime now depends on an explicit meta-store contract instead of the full compat DB |
| Testers | migrated to v3 meta storage | `cluster/store -> MetaStore -> HybridDB -> v3 Meta().Testers()` | storage moved out of legacy tables and store runtime no longer needs whole `wkdb.DB` for tester APIs |
| Plugins | migrated to v3 meta storage | `cluster/store -> MetaStore -> HybridDB -> v3 Meta().Plugins()` | plugin APIs are store-routed and store now uses a dedicated meta-store boundary |
| Plugin users | migrated to v3 meta storage | `cluster/store -> MetaStore -> HybridDB -> v3 Meta().PluginUsers()` | binding storage is now in v3; final ownership/replication boundary is still a separate decision |
| Notify queue | migrated to v3 local storage | `cluster/store -> NotifyQueueStore(LocalNotifyQueueStore) -> v3 Local().NotifyQueue()` | node-local queue storage no longer depends on legacy `wkdb` tables, and notify-queue runtime access is now isolated from the channel/message store boundary |
| Total counters | partial | mixed | admin totals now come from `store.AdminSearchStore` over v3; raw `wkdb.DB` total methods still fall back to legacy behavior |
| ID allocation | partial | `cluster/store -> PrimaryKeyAllocator(SnowflakePrimaryKeyAllocator)` for main runtime; compat/hybrid fallback still exposes legacy `NextPrimaryKey()` | main cluster/store runtime no longer depends on legacy `wkdb` just to allocate business IDs, but compatibility paths still retain the old allocator surface |
| ID allocation helper | partial | `cluster/store -> PrimaryKeyAllocator -> dedicated snowflake allocator in main runtime; legacy NextPrimaryKey()` in compat fallback | store runtime no longer needs the full compat DB surface just to allocate IDs; API-layer message batching no longer groups by old DB shard, but compatibility paths still keep the old allocator behavior |

## Module Matrix

| Module | Status | Notes |
| --- | --- | --- |
| `pkg/cluster/store` | partial | main compat layer; slot-owned plus meta/local compat methods route to v3, ID allocation routes through `PrimaryKeyAllocator`, user/device reads and apply writes route through `UserDeviceStore`, conversation reads and apply writes route through `ConversationStore`, channel/subscriber/allowlist/denylist reads and apply writes route through `ChannelStateStore`, channel-cluster-config reads and apply writes route through `ChannelClusterConfigStore`, meta APIs route through `MetaStore`, meta mutations route through an explicit `MetaCommandProposer`, admin searches route through `AdminSearchStore`, the old `HybridDB` cross-slot search overrides have been removed, conversation-local-user aggregation now prefers service-layer projection, channel-scoped message queries route through `MessageQueryStore`, message index helpers route through `MessageIndexStore`, recent-message batch helpers now compose over `MessageQueryStore` in the store layer, message search/count APIs route through `MessageSearchStore`, notify-queue APIs route through `NotifyQueueStore`, message-event APIs now route through an explicit `SlotMessageEventStore`, hybrid/compat runtime bindings now expand into explicit stores plus explicit lifecycle/snapshot hooks, and the old `Store.DB()` escape hatch is gone |
| `pkg/cluster/cluster` | partial | startup still chooses the hybrid runtime, but `Server` no longer owns or injects a raw compat DB into `store`; it now passes explicit runtime bindings plus dedicated lifecycle/snapshot dependencies instead |
| `pkg/cluster/channel` | partial | depends on a narrowed channel-log storage contract and now receives an explicit legacy-backed channel-log adapter instead of the full compat DB |
| `pkg/cluster/icluster` | partial | no longer exposes `WKDB() wkdb.DB`; remaining store interface is still compatibility-oriented |
| `internal/plugin` | partial | plugin APIs now route through `service.Store` methods instead of direct `wkdb.DB` calls, but the store boundary is still compatibility-shaped |
| `pkg/cluster/slot` | legacy helper dependency | still uses `wkdb.BatchDB/NewBatchDB` |
| `cmd/db` and `pkg/wkdb/v3/inspect` | v3-native | inspection tooling already targets `dbv3` |

## Current Meaning Of v3

Today, `wkdb/v3` is best described as:

- a working slot-native storage engine
- a working slot snapshot backend
- a partial replacement for slot-owned business tables
- not yet a complete replacement for the legacy `wkdb.DB` runtime contract

## Main Gaps Blocking Full Replacement

The major remaining gaps are:

1. channel message storage is still physically tied to legacy `wkdb`, even though runtime access is now split into explicit channel-log and query adapters
2. broad `wkdb.DB` ownership still exists internally for some slot/meta/local compat paths and compat wrappers, even though the old public `Store.DB()` / `icluster.WKDB()` escape hatches are removed, store lifecycle/snapshot wiring is now explicit, and channel-state/message/meta/admin-search reads have been narrowed
3. plugin and other meta APIs are store-routed now, but the store surface is still compatibility-shaped rather than final v3-native service APIs
4. message-event callers still rely on old compat semantics even though the store boundary is now narrowed
5. some legacy-shaped fallback query paths still exist in `pkg/cluster/store` for non-hybrid or degraded cases, even though the main cluster admin search endpoints and conversation-local-user aggregation no longer depend on `HybridDB` compatibility scans at runtime and the old `HybridDB` scan overrides have been removed
6. compatibility-path ID-allocation helper facilities still exist
7. meta storage is implemented, and the old `slot 0` write convention is now isolated behind a dedicated proposer, but the final meta raft / replication boundary is still not cleaned up
8. local storage is implemented, but restore/cutover behavior still needs full operational wiring

## Summary

The project has already migrated the slot-owned core of `wkdb` to `wkdb/v3`.
However, the system is still operationally hybrid.

The remaining work is no longer only "move a few tables". It is now mostly:

- interface boundary cleanup
- meta/local boundary cleanup
- message-domain extraction
- cutover tooling
- removal of remaining compatibility scaffolding and degraded-case fallbacks
