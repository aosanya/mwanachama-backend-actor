# Inviting to a group (ACT14, 2026-10-20)

A hidden group (ACT13, [joining-groups.md](joining-groups.md)) is never listed
as joinable and answers a join or a join request as a missing group, so the
only way in is an invitation. Asked for by the owner as member app decision
56; the leader's side is drawn in `mwanachama-frontend-member`'s
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
| `expires_at` | When it lapses, as the issuer chose. Empty means it never lapses. |
| `answered_at` | When the actor accepted or declined. Empty while pending. |
| `last_updated` | The last write. |

### `InviteToGroup(ctx, groupID, actorID, invitedBy, expiresAt)`

1. An empty group, actor or issuer is `ErrInvalidInvitation`. So is an
   `expires_at` that is not an RFC 3339 time, or is not in the future; a
   valid one is stored in UTC in the module's fixed-width `TimeLayout`, so
   it compares as text against `NowRFC3339()`.
2. A missing or deleted group is `ErrGroupNotFound`, a missing actor
   `ErrActorNotFound`. The host ensures the actor first (WK100's
   `EnsureActor`).
3. An actor already registered at the group is `ErrAlreadyInGroup` (409);
   there is nothing to invite them to.
4. A first invitation inserts it pending. Inviting the same actor again, in
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

**How long an invitation lasts is the caller's.** No document states a
duration, so the module takes none of its own: the issuer, or the host on its
behalf, supplies `expires_at`, and leaving it out makes an invitation that
never lapses. A product default belongs to the host or the member app, and is
for the owner to set.

### Who may invite is the host's to check

Decision 56 says a group's leaders issue invitations, and this module has role
kinds and seats per group but no definition of which kinds make a leader (the
same open question ACT13 left for deciding join requests). `InviteToGroup`
therefore records the issuer from the caller and checks nothing about it; the
mount gates `actor.invitation.create` and `actor.invitation.list_for_group`.

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
