# Conversation Ownership Cleanup

## Purpose

This document isolates one specific problem in the current architecture:

- `conversation` does not have one clean authoritative owner

This is the first domain that must be cleaned up before `slot snapshot` can be considered safe.

## Why Conversation Is Special

`conversation` looks like user state:

- rows are stored under `uid`
- reads are user-oriented
- deletion and update APIs are user-oriented

But one write path currently routes by `channelId`, not by `uid`.

That creates a mismatch between:

- replication owner
- storage owner

This is the core reason `conversation` is the first cleanup target.

## Current Storage Truth

In `wkdb v2`, `conversation` rows are stored under `uid`-keyed storage.

Examples in `pkg/wkdb/v2/conversation.go`:

- `AddOrUpdateConversations(...)` groups and writes by `uid`
- `AddOrUpdateConversationsWithUser(uid, ...)` writes through `uid`
- `DeleteConversation(uid, ...)` deletes through `uid`
- `DeleteConversations(uid, ...)` deletes through `uid`
- `UpdateConversationDeletedAtMsgSeq(uid, ...)` updates through `uid`
- `UpdateConversationIfSeqGreaterAsync(uid, ...)` also updates a row addressed by `uid`

So the physical truth is:

- `conversation` is a `uid`-anchored domain

## Current Routing Truth

In `pkg/cluster/store/conversation.go`, most conversation writes route by `uid slot`.

Examples:

- `AddOrUpdateConversations(...)`
- `AddOrUpdateUserConversations(...)`
- `AddConversationsIfNotExist(...)`
- `UpdateConversationDeletedAtMsgSeq(...)`
- `DeleteConversation(...)`
- `DeleteConversations(...)`

But one method is inconsistent:

- `UpdateConversationIfSeqGreaterAsync(...)`

Current logic:

- if personal channel -> route by `uid`
- else -> route by `channelId`

That is the ownership leak.

## Why This Is Wrong

The mutation changes a row that is materially part of the user's conversation state.

That row is stored under `uid`.

Therefore:

- the authoritative owner must be the `uid slot`

Routing some mutations through `channel slot` means:

1. the same conversation domain may be replicated by more than one slot
2. slot snapshot can miss part of authoritative state
3. restore can replay stale or conflicting state
4. ownership is no longer auditable from the API shape

## Current Write Path Audit

### Safe Today

These already align with `uid slot` ownership:

- `AddOrUpdateConversations`
- `AddOrUpdateUserConversations`
- `AddConversationsIfNotExist`
- `UpdateConversationDeletedAtMsgSeq`
- `DeleteConversation`
- `DeleteConversations`

### Unsafe Today

This is the main broken path:

- `UpdateConversationIfSeqGreaterAsync`

Problem:

- replication owner depends on `channelType`
- storage owner still depends on `uid`

This is exactly the kind of split ownership `v3` should eliminate.

## Secondary Derived State

`ConversationLocalUser` also deserves attention.

Current meaning:

- "which local users have conversation relation with this channel"

This structure is derived from conversation rows, but indexed by channel.

So it should not be treated as primary authoritative conversation state.

Recommended rule:

- `conversation` rows remain authoritative
- `ConversationLocalUser` becomes rebuildable / derived / locally maintained projection

Do not let this derived table redefine conversation ownership.

## Target Ownership Rule

The rule for `conversation` in `v3` should be simple:

- all conversation rows belong to `uid slot`
- all conversation mutations must be proposed to `uid slot`
- all conversation indexes must share that same `uid slot` prefix
- no `channel slot` path may directly mutate conversation rows

This rule should have no exceptions.

## API Direction

To make the rule visible in code, conversation APIs should become slot-scoped and uid-centered.

Examples:

```go
type ConversationStore interface {
    Put(uid string, cs []Conversation) error
    Delete(uid, channelId string, channelType uint8) error
    DeleteBatch(uid string, channels []Channel) error
    UpdateIfSeqGreater(uid, channelId string, channelType uint8, seq uint64) error
    UpdateDeletedAt(uid, channelId string, channelType uint8, seq uint64) error
}
```

This makes it harder to accidentally route by channel owner.

## Proposed Cleanup Phases

### Phase 1: Stop Cross-Slot Routing

Change `store.UpdateConversationIfSeqGreaterAsync(...)` so it always routes by:

- `slotId = uid slot`

This is the highest-priority semantic fix.

### Phase 2: Rename Ambiguous APIs

Current names like:

- `AddOrUpdateConversations`

are too generic and hide ownership.

Prefer names that expose the domain boundary more clearly.

Examples:

- `PutUserConversations`
- `UpdateUserConversationIfSeqGreater`
- `DeleteUserConversation`

The exact naming can be tuned later, but the API should stop hiding the `uid-owned` nature of the domain.

### Phase 3: Remove Channel-Side Assumptions

Audit all callers that may still think conversation is channel-owned.

Examples worth reviewing:

- subscriber/channel admin paths that create or delete conversation relations
- recvack path
- message-side read pointer updates

The rule should become:

- a channel-side event may trigger conversation updates
- but the actual mutation must be emitted toward the affected user's slot

### Phase 4: Reclassify ConversationLocalUser

Choose one of these models:

1. rebuildable projection
2. local async projection
3. explicitly authoritative cross-domain index

`v3` should not default to option 3.

Option 1 or 2 is strongly preferred.

## Caller Audit Notes

The current important caller categories are:

### User-Directed APIs

These generally align well with `uid slot`:

- direct user conversation update APIs
- delete/update conversation APIs

### Channel Admin Flows

These can create bulk user conversation mutations.

That is acceptable, but only if:

- they fan out by affected `uid`
- they propose to each target `uid slot`

### RecvAck Flow

`internal/user/handler/event_recvack.go` currently calls:

- `service.Store.UpdateConversationIfSeqGreaterAsync(...)`

This path must remain valid after cleanup, but routing must be:

- by `uid slot`

not by `channel slot`.

## Expected Benefits After Cleanup

Once conversation ownership is cleaned up:

1. one slot owns one closed conversation state set
2. slot snapshot semantics become valid for the conversation domain
3. routing becomes easier to reason about
4. future `v3` keyspace for conversation becomes straightforward

## Minimum Acceptable Fix Before v3 Storage Work

Before starting actual `v3` implementation, at minimum:

1. `UpdateConversationIfSeqGreaterAsync` must stop routing by `channelId`
2. the ownership rule must be documented and enforced in code review
3. `ConversationLocalUser` must be treated as non-authoritative

Without these three points, `conversation` remains structurally unsafe.

## Recommended Immediate Next Step

After this document:

1. implement the routing fix for `UpdateConversationIfSeqGreaterAsync`
2. add tests proving all conversation mutations route to `uid slot`
3. only then start implementing the first `v3` conversation key builders
