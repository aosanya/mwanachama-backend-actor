# Inviting to a group (ACT14, 2026-10-20)

A hidden group (ACT13, [joining-groups.md](joining-groups.md)) is never listed
as joinable and answers a join or a join request by its id as a missing
group, so the ways in are an invitation and, since ACT20, its code
([joining-groups.md](joining-groups.md#finding-a-group-by-its-code-act20-2026-10-21)).
Asked for by the owner as member app decision 56 (amended by 137); the leader's side is drawn in `mwanachama-frontend-member`'s
`mockups/join-group.html` (M04.3, Group settings → Invite someone).
wakala-api WK99 is the consumer.

## The invitation

A new role, `group_invitation`:

| Field | Meaning |
|---|---|
| `id` | Minted. An invitation is read and answered by id. |
| `group_id`, `actor_id` | The pair. Unique together, so there is at most one invitation per actor and group. |
| `state` | `pending`, `accepted` or `declined`. |
| `invited_by` | Who issued it, bound from the caller on the route table. |
| `invited_at` | When it was issued. |
| `expires_at` | When it lapses: the issuer's date, or the issue time plus the invitation lifetime (ACT16). Empty only on a row issued before ACT16, which never lapses. |
| `answered_at` | When the actor accepted or declined. Empty while pending. |
| `last_updated` | The last write. |

### `InviteToGroup(ctx, groupID, actorID, actorCode, invitedBy, expiresAt)`

1. An empty group or issuer is `ErrInvalidInvitation`, and so is an invitee
   named by neither `actorID` nor `actorCode`, or by both (see
   [Inviting by member code](#inviting-by-member-code-act18-2026-10-21)). So is an
   `expires_at` that is not an RFC 3339 time, or is not in the future; a
   valid one is stored in UTC in the module's fixed-width `TimeLayout`, so
   it compares as text against `NowRFC3339()`. An empty one becomes the
   issue time plus the manager's invitation lifetime (below).
2. A missing or deleted group is `ErrGroupNotFound`. An issuer who does not
   lead the group is then `ErrNotGroupLeader` (403, ACT15).
3. A missing actor is `ErrActorNotFound`, whether named by id or by code. The host ensures the actor first
   (WK100's `EnsureActor`).
4. An actor already registered at the group is `ErrAlreadyInGroup` (409);
   there is nothing to invite them to.
5. A first invitation inserts it pending. Inviting the same actor again, in
   any state, updates the one row back to pending with the new issuer,
   issue time and expiry and clears `answered_at`. That is how a lapsed or
   declined invitation is renewed.

Two first invitations racing for the same pair both pass the read; the unique
pair index lets one insert land, and the loser retries once onto the update
path, as `JoinGroup` does.

The group need not be hidden. An invitation to a shown group is harmless and
keeps working if its leaders hide or show the group in between.

### Answering

`AcceptInvitation(ctx, id, actorID)` and `DeclineInvitation(ctx, id, actorID)`.
The answering actor is bound from the caller.

- An invitation addressed to somebody else answers `ErrInvitationNotFound`,
  exactly as a missing one, so nobody can learn who else was invited.
- One already accepted or declined is `ErrInvitationAnswered` (409).
- One past its `expires_at` is `ErrInvitationLapsed` (409), for a decline as
  well as an accept: it is no longer waiting for an answer.
- Accepting checks the group still exists, then moves the invitation out of
  `pending` with a conditional write (still pending, still not lapsed) and
  writes the `registered_at` edge, the `actor_group_assignment` row, in the
  same transaction. A pending join request the actor had left at the group
  is marked approved with an empty `decided_by`, as `JoinGroup` does on an
  open group, so the deciders' list never shows somebody already in.
- Declining writes `declined` and `answered_at` and registers nothing.

**Lapsing writes nothing.** No job sweeps expired invitations and no state
records it; an invitation is lapsed when it is pending and its `expires_at`
has passed, which `GroupInvitation.Lapsed(now)` and `Open(now)` compute.

### How long an invitation lasts (ACT16, 2026-10-20)

Owner, member app decision 81: an invitation's lifetime is a system setting,
starting at 14 days. Until ACT16 an invitation lapsed only when its issuer
set `expires_at`, and one without a date never lapsed.

This module is a library with no environment of its own, so the setting is
the host's to supply when it builds the manager:

```go
mgr, err := mwanachamaactor.NewUserManager(db, s,
	mwanachamaactor.WithInvitationLifetime(lifetime))
```

- No option means `DefaultInvitationLifetime`, 14 days.
- A lifetime of zero or less is refused by `NewUserManager`, so a typo in a
  host's configuration cannot quietly make every invitation lapse on issue.
  There is no setting for "never lapses"; decision 81 asks for none.
- An invitation issued without `expires_at` is stored with the issue time
  plus the lifetime, computed from the same clock reading as `invited_at`,
  so the two always differ by exactly the lifetime.
- An invitation issued with its own `expires_at` keeps it, longer or shorter
  than the lifetime.
- Renewing an invitation (inviting the same actor again) without a date
  restarts the lifetime from the renewal.
- The lifetime is read at issue and written into the row. Changing the
  setting moves no invitation already issued, and rows issued before ACT16
  with an empty `expires_at` still never lapse.

wakala-api reads it from `INVITATION_LIFETIME`; see its
`documentation/2. design/actor-mount.md`.

### Only the group's leaders invite

Decision 56 says a group's leaders issue invitations. `InviteToGroup` records
the issuer from the caller and, since ACT15, refuses one who does not lead the
group with `ErrNotGroupLeader` (403), checked after the group is found and
before the invited actor is looked up. What makes a leader is in
[group-leaders.md](group-leaders.md). The mount still gates
`actor.invitation.create` and `actor.invitation.list_for_group`.

### Inviting by member code (ACT18, 2026-10-21)

Member app decisions 132 and 136: a leader invites someone by their member
code. A leader in the member app cannot see the agency's actor ids, but the
invitee can read their own code out to them. Since ACT21 (decisions
179–181) that code is the actor's `public_code`, `M-` and random groups such
as `M-K7P-4QX-M2T`, described in [public-codes.md](public-codes.md). The
sequential `code`, `AC-<n>`, is internal and is no longer accepted.

- `invite_to_group` takes the invitee in the body as `actor_id` or as
  `actor_code`, exactly one of them. Neither or both is
  `ErrInvalidInvitation` (400).
- A code is matched against `public_code` after trimming surrounding spaces
  and upper-casing, so `" m-k7p-4qx-m2t "` finds `M-K7P-4QX-M2T`. It is
  matched only among the agency's own actors, since each agency's actors are
  their own table set, and a deleted actor's code finds nobody.
- A code naming nobody is `ErrActorNotFound`, the same refusal and the same
  body as an unknown `actor_id`, so a code cannot be told apart from an id by
  how it fails. An internal `AC-<n>` is refused exactly the same way.
- The leader check still runs before the invitee is looked up, so a
  non-leader learns nothing about which codes exist.
- Everything after the lookup is unchanged: the stored invitation carries the
  resolved `actor_id`, and inviting by code renews the same one row an
  invitation by id would.

The invitee reads their own code through `get_my_actor`
(`GET /actors/mine`, `actor.actor.read_mine`), which is `GetActor` with the
actor bound from the caller. It answers the caller's whole actor record,
`public_code` included, and a caller with no actor row yet answers
`ErrActorNotFound` (404) rather than having one minted by a read; the host
ensures the actor as it does elsewhere. The record still carries the
internal `code` as well; keeping it off a member's screen is the host's.

### Reads

- `ListInvitationsForGroup(ctx, groupID, openOnly)`: a group's invitations,
  oldest issued first; `openOnly` keeps those pending and not lapsed.
- `ListInvitationsForActor(ctx, actorID)`: the actor's invitations in every
  state, lapsed ones included. What the member app shows for a lapsed one is
  its choice.
- `GetInvitation(ctx, id)`.

## Route table

| Method | Path | Action | Bound from the session |
|---|---|---|---|
| POST | `/groups/{group_id}/invitations` | `actor.invitation.create` | `invited_by` |
| GET | `/actors/mine` | `actor.actor.read_mine` | `actor_id` |
| GET | `/groups/{group_id}/invitations` | `actor.invitation.list_for_group` | |
| GET | `/invitations/mine` | `actor.invitation.list_mine` | `actor_id` |
| GET | `/invitations/{invitation_id}` | `actor.invitation.read` | |
| POST | `/invitations/{invitation_id}/accept` | `actor.invitation.accept` | `actor_id` |
| POST | `/invitations/{invitation_id}/decline` | `actor.invitation.decline` | `actor_id` |

## Not here

- Withdrawing an invitation before it is answered.
- A notification to the invited actor; raising one is the host's, through
  comm.
- Reading a hidden group's name for an invitee: `GetGroup` reads any group,
  and whether an invitee may is the host's gate.

## Checking a list of invitations (ACT26, first half)

Member app decision 221: an operator uploads a list of first name, last name,
email and group code, previews it, and sends every good row. The preview is
`CheckInvitationBatch(ctx, rows)`. It writes nothing and answers one check per
row, in order, numbered from 1, each with the row as normalised (names and
email trimmed, email lower-cased, group code normalised like any public code)
and one status:

| Status | Meaning |
| --- | --- |
| `missing_name` | first or last name is blank |
| `bad_email` | not a single plain address with a dotted domain |
| `unknown_group` | the code names no live group |
| `duplicate` | an earlier row already names this email for this group |
| `already_member` | an actor holding this email is registered in the group |
| `ready` | none of the above |

The checks run in that order, so a row carries its first problem only. A
duplicate is judged on `(email, group)` after normalisation, which is also the
key the send will be idempotent on. A ready row carries the group's id and,
when an actor already holds the email (the `email` attribute), that actor's id.

Over the wire the check is `POST /invitations/batch/check`, action
`actor.invitation.check_batch`, tool `actor_check_invitation_batch`, with the
list as `rows` in the body. Who may check is the host's gate; the module
answers anyone the gate lets through.

A row may also carry `actor_id`, the invitee the host resolved for its email.
When it does, the check judges `already_member` against that actor rather
than against an `email` attribute lookup.

## Sending a list of invitations (ACT26, second half)

Member app decisions 379 and 380 settled the two questions the send waited
on. **The host gives every row an actor first**: an email with no account
goes through the single invitation's path to get one, and the host fills each
row's `actor_id` before sending. This module never creates an account from an
email. **Only an agency's owners and agents send**, and that is the host's
gate, so the send skips the leader check `InviteToGroup` makes.

`SendInvitationBatch(ctx, rows, invitedBy)` runs `CheckInvitationBatch` again
over the same rows, so a list edited between preview and send is judged as it
is now, then invites each ready row and answers one result per row, in order
and numbered from 1. A result is the row's check plus, when sent, the
invitation:

| Status | Meaning |
| --- | --- |
| `sent` | the row's invitation is pending, with `invitation` set |
| `no_actor` | the row names no actor, or an actor that does not exist |
| `outsider` | the actor holds an outsider role and may not join groups |
| any check status | the row was not ready, and nothing was written for it |

A sent row's invitation is issued exactly as `InviteToGroup` issues one, with
the host's invitation lifetime and `invited_by` from the session. A blank
sender is `ErrInvalidInvitation`.

**Idempotent on (email, group).** Within one send a repeated row is a
`duplicate`. Across sends, a row whose actor already holds a pending,
unlapsed invitation to the group answers that invitation unchanged, same id
and same `invited_at`, rather than issuing or renewing one. An answered or
lapsed invitation is renewed in place, as a second `InviteToGroup` would.

Over the wire the send is `POST /invitations/batch`, action
`actor.invitation.send_batch`, tool `actor_send_invitation_batch`, with the
list as `rows` and the sender bound from the caller. wakala-api's mount and
the gate that admits only owners and agents are WK159.
