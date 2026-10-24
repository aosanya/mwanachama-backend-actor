# Delegations: one actor acting for another (ACT36)

Before this, a role was bound to a group, and nothing in the module said that
one actor may act for another. "This councillor may file for this resident"
was a convention, and the only audit of it was whatever got typed into the
record (municipality template, member app decision 283).

## The record

`delegation` is a role of its own: `delegate_id` (the actor allowed to act),
`subject_id` (the actor acted for), `granted_by`, `granted_at`, an optional
`ends_at`, then `revoked_by` and `revoked_at`.

- **The grantor and the revoker come from the caller.** `granted_by` and
  `revoked_by` are bound with `from: caller`. A body that names either one is
  refused, because the dispatcher does not accept a body field that an
  argument already binds. The body carries a `DelegationGrant`: the delegate,
  the subject and an optional end.
- **Both actors must exist, and they must be different actors.** If either is
  missing the call fails with `ErrActorNotFound`. A delegation to oneself, a
  missing grantor or an `ends_at` that is not RFC 3339 fails with
  `ErrInvalidDelegation` (400).
- **A revocation stamps the record and never deletes it.** That way, what was
  done under a delegation still shows the authority it was done under.
  Revoking twice fails with `ErrDelegationRevoked` (409). The guard sits in
  the UPDATE's WHERE clause, so two concurrent revokes cannot both succeed.
- **`ActsFor(ctx, delegate, subject)`** returns true only when there is an
  unrevoked delegation whose `ends_at` is empty or still in the future. A
  delegation runs one way only, and an actor never acts for themselves
  through this check. It is a Go method only and has no address. The host
  calls it before it accepts a write made on someone else's behalf.
- `GET /actors/{actor_id}/delegations` lists every delegation where the actor
  is the delegate or the subject, oldest first. Revoked and lapsed ones are
  included, because the list is the audit.

## Who may grant: open for the owner

Each address has its own action: `actor.delegation.create`,
`actor.delegation.revoke` and `actor.delegation.list`. The module adds no role
rule of its own, so the host's gate on these actions decides who may grant.
Open questions:

- Who holds `actor.delegation.create`? Staff only, or may the subject grant
  their own delegation?
- Should the subject be told when a delegation naming them is granted or
  revoked?
- Should a write made under a delegation carry the delegation's id on the
  record it writes?
