# wkdb v3 Data Ownership Mapping

## Purpose

This document maps current `wkdb v2` logical domains and physical tables to their intended `wkdb v3` ownership model.

The goal is to answer three questions for every domain:

1. who owns the state
2. where the state should live
3. whether the state belongs to `slot snapshot`

## Ownership Classes

`wkdb v3` should classify data into four groups.

### 1. Slot Authoritative State

This is the main target of `slot snapshot`.

Properties:

- owned by exactly one slot
- replicated by one raft state machine
- encoded under one slot-prefixed key range

### 2. Meta Authoritative State

This is cluster-global state and should not live under any business slot.

Properties:

- replicated by dedicated `meta raft`
- stored in dedicated `meta db`
- excluded from `slot snapshot`

### 3. Channel-Authoritative but Non-Slot State

This state belongs to `channel raft`, not `slot raft`.

Properties:

- not part of `slot snapshot`
- may need a separate `channel snapshot` story later

### 4. Node-Local or Derived State

This state is either:

- local operational state
- cache-like derived state
- rebuildable projection
- or raft/storage metadata

Properties:

- not authoritative business state
- should not define slot snapshot semantics

## Mapping Rules

Before the table-by-table mapping, the intended v3 rules are:

- `user`, `device`, `conversation` are owned by `uid slot`
- `channel info`, `subscriber`, `allowlist`, `denylist`, `channel cluster config`, `message event state`, `message event seq` are owned by `channel slot`
- global plugin/system/tester style data moves to `meta raft`
- channel message storage stays outside `slot snapshot`
- rebuildable caches and derived indexes stay outside authoritative slot state

## Mapping Table

| v2 table / domain | Current meaning | Current placement signal | v3 owner | v3 placement | In slot snapshot | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| `TableUser` | user profile and counters | `uid -> shard` | `uid slot` | slot bucket db | yes | authoritative user state |
| `TableDevice` | user devices | `uid -> shard` | `uid slot` | slot bucket db | yes | authoritative user-attached state |
| `TableConversation` | user conversation rows | `uid -> shard` but writes are not consistently routed | `uid slot` | slot bucket db | yes | must be made `uid slot` only before snapshot is enabled |
| `TableConversationLocalUser` | channel to local-user relation derived from conversations | channel-keyed side index | derived | rebuildable local projection or async maintained index | no | do not treat as authoritative slot state; it crosses channel and uid domains |
| `TableChannelInfo` | channel metadata | `channel -> shard` | `channel slot` | slot bucket db | yes | authoritative channel state |
| `TableSubscriber` | subscriber membership rows | `channel -> shard` | `channel slot` | slot bucket db | yes | authoritative channel state |
| `TableSubscriberChannelRelation` | inverse subscriber/channel relation | channel-side relation index | `channel slot` secondary index | slot bucket db | yes | index must share the same slot prefix as its primary rows |
| `TableAllowlist` | channel allowlist | `channel -> shard` | `channel slot` | slot bucket db | yes | authoritative channel state |
| `TableDenylist` | channel denylist | `channel -> shard` | `channel slot` | slot bucket db | yes | authoritative channel state |
| `TableChannelClusterConfig` | channel distributed config | `channel -> shard` | `channel slot` | slot bucket db | yes | authoritative channel state |
| `TableMessageEventState` | projected event-key state for one message | `channel -> shard` | `channel slot` | slot bucket db | yes | authoritative current projection state |
| `TableMessageEventSeq` | next / latest message event sequence | `channel -> shard` | `channel slot` | slot bucket db | yes | auxiliary but authoritative for event-state progression |
| `TableMessageEvent` | event history indexes | `channel -> shard` | pending decision | likely channel-owned or removed | no in slot v3 | current implementation persists projection state, not full history rows; keep out of slot snapshot planning |
| `TableMessage` | channel message storage | `channel -> shard`, written via channel raft | `channel raft` | separate channel storage track | no | explicitly outside slot snapshot scope |
| `NewChannelLastMessageSeqKey(...)` | channel last message sequence auxiliary key | `channel -> shard` | `channel raft` | separate channel storage track | no | tied to message/channel replication, not slot replication |
| `TableChannelCommon` | channel applied-index style auxiliary state | `channel -> shard` | `channel raft` or channel-local metadata | separate channel storage track | no | do not mix with slot snapshot |
| `TableMessageNotifyQueue` | webhook notify queue | local db queue | node-local | local operational db | no | operational queue, not replicated slot state |
| `TableSystemUid` | system uid set | implicit global data in main db | `meta raft` | meta db | no | remove slot-0 convention |
| `TableTester` | tester registry | implicit global data in main db | `meta raft` | meta db | no | cluster-global control-plane state |
| `TablePlugin` | plugin definitions and config | currently direct DB writes, not slot-routed | pending meta decision | meta db or local admin db | no | if cluster-consistent, move to `meta raft`; otherwise mark local-only explicitly |
| `TablePluginUser` | plugin-to-user binding | currently global-like data | `meta raft` by default | meta db | no | should not stay in slot-local state |
| `TableTotal` | aggregate counters | singleton counters | derived or meta | preferably derived / rebuildable | no | keep out of slot snapshot unless promoted to authoritative state |
| `TableLeaderTermSequence` | raft term-start metadata | shard/log storage metadata | raft storage metadata | raft log storage only | no | not business state; should not live in wkdb v3 business snapshot scope |

## Special Cases

### Conversation

`conversation` is the first domain that must be cleaned up.

Required rule:

- every conversation mutation must be routed by `uid slot`
- no `channel slot` path may mutate conversation rows

Without this rule, `slot snapshot` is not semantically valid.

### ConversationLocalUser

`ConversationLocalUser` is not a clean primary domain.
It is a cross-domain projection used to answer:

- "which local users have a conversation relation with this channel"

Because its primary source of truth is conversation state, it should not be treated as authoritative slot state in `v3`.

Recommended options:

1. rebuild it on demand
2. maintain it asynchronously as local projection
3. if it must become authoritative, redesign it as explicit cross-slot indexed state with separate semantics

Option 1 or 2 is strongly preferred.

### Message and Channel Common State

The channel message path is already separate from slot raft.
That separation should be preserved, not blurred.

So in `v3`:

- `message`
- `channel last message seq`
- `channel applied index`

should remain out of `slot snapshot` scope.

### Plugin State

`plugin` is currently inconsistent from a replication standpoint:

- plugin definitions are written directly through `wkdb`
- plugin user bindings are also stored in `wkdb`
- cluster/store replicated paths for full plugin definition changes are commented out

This means `plugin` needs an explicit architecture decision:

1. cluster-global authoritative state -> move to `meta raft`
2. node-local admin state -> declare it local and exclude from replicated storage

For `v3`, option 1 is cleaner if plugins are expected to be cluster-consistent.

## Recommended Migration Order

1. classify every table into one of the four ownership classes
2. enforce `conversation` as `uid slot` only
3. move global state out of slot conventions into `meta raft`
4. redesign slot-owned keys so all primary rows and indexes share the same slot prefix
5. keep local and derived state explicitly outside authoritative slot snapshot

## Output of This Mapping

After this document is accepted, implementation should be able to proceed with:

1. `v3` key builder design
2. `slotId -> bucketId` physical layout
3. `ExportSlotKV / ImportSlotKV / DeleteSlotKV`
4. conversation routing cleanup

Without the ownership mapping being stable, those steps would remain high risk.
