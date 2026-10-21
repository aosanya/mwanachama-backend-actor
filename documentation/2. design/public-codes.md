# Member and group codes (ACT21, 2026-10-21)

Member app decisions 179–181. A member and a group each carry a **public
code**, the one a person reads out, shares or types: `M-` for a member, `G-`
for a group, then groups of three characters joined by hyphens, such as
`M-K7P-4QX-M2T`. It supersedes decision 136's `AC-<n>` and `G-<n>` for
everything shown or typed. Those sequential `code`s stay, unchanged, as
internal numbers: never shown to a member and never accepted from one.
wakala-api WK143 passes the new code through the actor mount.

## The code

- `actor` and `group` each gained `public_code`, a string drawn once when
  the row is created and never rewritten. A body naming one is ignored; the
  server draws it.
- The characters are `23456789ABCDEFGHJKMNPQRSTUVWXYZ`: letters and digits
  without the look-alikes `0`/`O` and `1`/`I`/`L`. Each one is drawn with
  `crypto/rand`, so a code cannot be found by counting, for listed and hidden
  groups alike (decision 179).
- Codes are unique per agency. Each agency is its own instance with its own
  actor and group tables, so the check is within those tables, and a unique
  index on `public_code` (leaving out rows not yet backfilled, which carry
  none) is the database's half of it. The prefixes keep a member's code and
  a group's apart.
- **A collision adds one group.** A drawn code that is already held is
  dropped and a new one drawn with one more group, then one more again, up to
  eight draws; eight collisions in a row is an error rather than a silent
  loop. `TestACollidingDrawAddsOneGroup` forces the collisions.

## How many groups (decision 180)

A new role, `code_format`, holds at most one row per agency, id `codes`:

| Field | Meaning |
|---|---|
| `group_count` | How many groups a newly drawn code has before any collision adds one. |
| `last_updated` | When it last changed. |

With no row the count is three. It is read every time a code is drawn, so a
change applies only to codes drawn afterwards; codes already drawn keep their
length.

- `CodeFormat(ctx)` answers the format, three groups when none is set.
- `SetCodeFormat(ctx, groupCount)` writes it. A count below 1 or above 12 is
  `ErrInvalidCodeFormat` (400). The bounds keep a code drawable and short
  enough to read out; the owner's decision names no range.

| Method | Path | Action |
|---|---|---|
| GET | `/code-format` | `actor.code_format.read` |
| PUT | `/code-format` | `actor.code_format.set` |

Decision 180 gives the setting to an agency's owner. Who may call
`actor.code_format.set` is the mount's gate, as for every action here; this
module names no owner.

## Every existing row gets one (decision 181)

`Provision` adds the `public_code` column to an existing actor or group table
(`columnsDeclaredSince`), migrates, and then draws a code for every actor and
group, deleted ones included, whose `public_code` is empty, oldest first.
`upToDate` treats any empty `public_code` as out of date, so a provisioned
database never keeps an uncoded row, and provisioning again draws nothing
new. The internal `code` is left exactly as it was.
`TestProvisionGivesEveryExistingActorAndGroupAPublicCode` drops the column
from a provisioned table, provisions again and checks every row.

## Where the code is accepted

Only `public_code` is matched, after trimming surrounding spaces and
upper-casing:

- inviting by member code (`actor_code`, [invitations.md](invitations.md#inviting-by-member-code-act18-2026-10-21));
- finding and joining a group by its code ([joining-groups.md](joining-groups.md#finding-a-group-by-its-code-act20-2026-10-21)).

An internal `AC-<n>` or `G-<n>` typed into either answers exactly as a code
naming nobody.

## Not here

- Hiding the internal `code` from what the mount returns to a member: the
  actor and group records still carry both fields, and which one a screen
  shows is the host's (wakala-api WK143).
- Redrawing a code on request.
