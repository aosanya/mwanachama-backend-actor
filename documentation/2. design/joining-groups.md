# Joining a group (ACT13, 2026-10-20)

A group says how it is joined, and an actor who is not registered at it can
join it or ask to. Asked for by the owner for the member app's Join Group
screen (`mwanachama-frontend-member` MF22, its
`documentation/2. design/join-group.md`, requirements decisions 56–62).
wakala-api WK99 is the consumer.

## The group's policy

`group` gained two declared fields:

| Field | Values | Default | Meaning |
|---|---|---|---|
| `join_mode` | `open`, `confirmed` | `confirmed` | `open` registers a joining actor at once; `confirmed` holds a join request until it is decided. |
| `hidden` | bool | `false` | A hidden group is reached only by invitation (ACT14): it is never listed as joinable, and a join or a request to it answers exactly as a missing group, `ErrGroupNotFound`. |

The stored values are `open` and `confirmed`. The member app words them
"Anyone can join" and "Approval needed"; that wording is the app's, never
stored (decision 60). A new group starts `confirmed` (decision 59):
`CreateGroup` fills an empty `join_mode` with it and refuses any other value
with `ErrInvalidGroup` wrapping `ErrInvalidJoinMode`.

`SetGroupJoinMode` and `SetGroupHidden` change them afterwards. `EditGroup`
and `GroupEdit` are unchanged, because `GroupEdit` writes every field it
carries on every call, and widening it would have made every existing edit
clear the join policy.

`hidden` sits beside the older `discoverable` and does not replace it.
`ListDiscoverableGroups` still reads `discoverable` and nothing else; the
joinable read below ignores `discoverable` and reads `hidden`, because
decision 56 lists every group the member is not in except hidden ones.

### Existing databases

`spec.Migrate` creates tables but does not add a column to one that already
exists (shared's S52). `Provision` therefore adds the two columns itself,
from `columnsDeclaredSince` in `provision.go`, before it migrates, and
`upToDate` treats a missing one as out of date. An existing group reads
back `confirmed` and shown, the safe defaults: nothing is joinable at once
until a group's leaders say so.
`TestProvisionAddsTheJoinColumnsToAnExistingGroupTable` drops both columns
from a provisioned table and provisions again.

## The join request

A new role, `group_join_request`:

| Field | Meaning |
|---|---|
| `id` | Minted. A request is read and decided by id. |
| `group_id`, `actor_id` | The pair. Unique together, so there is at most one request per actor and group. |
| `state` | `pending`, `approved` or `declined`. |
| `appeal` | Set when the actor asks again after a decline (decision 62, as decision 33 for agencies). |
| `message` | What the actor wrote, at most 500 characters (`ErrJoinMessageTooLong`), trimmed. |
| `silent` | Set on a quiet decline (decision 58). |
| `requested_at` | The first ask. An appeal keeps it; asking again after an approval starts afresh. |
| `asked_at` | The last ask: the first one, an appeal, or asking again while pending. A decision leaves it alone, so on a decided request it is still the time the actor last saw while it was pending. A host showing a quiet decline as still pending shows this time (member app decision 113). Added to an existing table by `Provision`, empty on rows written before it. |
| `decided_by`, `decided_at` | Who decided and when. Empty while pending. |
| `last_updated` | The last write. |

### `JoinGroup(ctx, groupID, actorID, message)`

Answers a `JoinOutcome` with `status` one of `joined`, `requested` or
`member`:

1. A missing, deleted or hidden group is `ErrGroupNotFound`; a missing actor
   is `ErrActorNotFound`. The host ensures the actor first (WK100's
   `EnsureActor`).
2. An actor already registered gets `member` and the registration, and
   nothing is written.
3. An open group writes the registration at once and answers `joined`. A
   pending request the actor had left from when the group was confirmed is
   marked `approved` with an empty `decided_by`, meaning the policy approved
   it, so the deciders' pending list never shows somebody already in.
4. A confirmed group writes or updates the one request and answers
   `requested`. A first ask inserts it pending. Asking again while pending
   replaces the message and changes nothing else. Asking again after a
   decline returns it to pending, sets `appeal`, replaces the message and
   clears the decision. Asking again after an approval (the actor has since
   left) starts a fresh pending request that is not an appeal.

Two first asks racing for the same pair both pass the read; the unique pair
index lets one insert land, and the loser retries once onto the update path.

### Deciding

`ApproveJoinRequest(ctx, id, decidedBy)` and
`DeclineJoinRequest(ctx, id, decidedBy, silent)`. Each moves the request out
of `pending` with a conditional write, so a request decided twice, or
decided by two leaders at once, is decided once and the other gets
`ErrJoinRequestDecided` (409). Approval writes the registration in the same
transaction. An empty `decidedBy` is `ErrInvalidJoinRequest`.

`silent` is stored and nothing more: this module raises no notification. The
host raises comm's decline event (CM33) for a decline whose `silent` is
false, and raises nothing for one that is true.

**Only the group's leaders decide** (decision 57). The deciding methods take
the decider as an argument, bound from the caller on the route table, and
since ACT15 refuse one who does not lead the request's group with
`ErrNotGroupLeader` (403), writing nothing. What makes a leader is in
[group-leaders.md](group-leaders.md). The mount still gates
`actor.join_request.approve` and `actor.join_request.decline`.

### Reads

- `ListJoinableGroups(ctx, actorID)`: every group that is not deleted, not
  hidden and has no registration for the actor, oldest first. A group the
  actor has a pending request at stays in the list; the screen shows it as
  Requested by reading the actor's requests.
- `ListJoinRequestsForActor(ctx, actorID)`: the actor's requests in every
  state. A silent decline reads as `declined` with `silent` true; what the
  member app shows for it is the host's choice.
- `ListJoinRequestsForGroup(ctx, groupID, pendingOnly)` and
  `GetJoinRequest(ctx, id)`.

## Route table

| Method | Path | Action | Bound from the session |
|---|---|---|---|
| PUT | `/groups/{group_id}/join-mode` | `actor.group.set_join_mode` | |
| PUT | `/groups/{group_id}/hidden` | `actor.group.set_hidden` | |
| GET | `/groups/joinable` | `actor.group.list_joinable` | `actor_id` |
| POST | `/groups/{group_id}/join` | `actor.group.join` | `actor_id` |
| GET | `/groups/{group_id}/join-requests` | `actor.join_request.list_for_group` | |
| GET | `/join-requests/mine` | `actor.join_request.list_mine` | `actor_id` |
| GET | `/join-requests/{request_id}` | `actor.join_request.read` | |
| POST | `/join-requests/{request_id}/approve` | `actor.join_request.approve` | `decided_by` |
| POST | `/join-requests/{request_id}/decline` | `actor.join_request.decline` | `decided_by` |

The joining actor and the decider come from the caller, never the body, so
no request can join somebody else or decide in somebody else's name.

## Not here

- Invitations to hidden groups: ACT14, [invitations.md](invitations.md).
- Withdrawing a request, and leaving a group beyond the existing
  `Deregister`.
- The private chat a request carries (decision 63) is comm's (CM36).
