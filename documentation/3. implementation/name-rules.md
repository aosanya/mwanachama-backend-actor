# Name rules

Three names are required and bounded in this module: a group name, an actor display name and a hierarchy name (hub issue MUN678). The check lives in `name_check.go` and runs inside the manager, so every caller gets it, not only the HTTP routes.

## The rules

| Name | Written by | Refused with |
|---|---|---|
| Group name | `CreateGroup`, `EditGroup` | `ErrInvalidGroup` |
| Actor display name | `CreateActor`, `SetActorDisplayName` | `ErrInvalidActor` |
| Hierarchy name | `CreateHierarchy`, `RenameHierarchy` | `ErrInvalidHierarchy` |

1. A name that is empty or holds only white space (including the no-break space and the ideographic space) is refused: "a group name is required".
2. A name of more than 200 characters is refused: "a group name may be at most 200 characters". Characters are counted as runes, so 200 accented Hungarian letters are kept and 201 are refused.
3. The rules of `plaintext.Problem` still apply after the two above: one visible line, no control or invisible characters, at least one letter, digit or symbol that shows.

The routes answer 400 for all three refusals because each sentinel is mapped to 400.

## Where the numbers come from

200 is the limit the group edit route already enforced (`maxGroupName`, now removed from `routes/routes.go` because the manager enforces it) and the limit of an agency name (`throttle.MaxInstanceNameBytes` in the hub). No new number was chosen.

## One exception

An actor display name that is exactly empty is still accepted. A member is minted from a device or a phone number before any name is given, and the existing tests and registration flow depend on it. A display name of only spaces is refused.

## Not covered

First and last names, level names, group type names, role kind names and group name translations keep the plaintext check only, with no length limit and no blank refusal. They are the subject of other hub issues (MUN679 and MUN680 call the same check).

## Proof

`name_length_test.go` drives the six writes in the table with 200, 201, 1000 and 100000 characters and with blank names. `routes/name_length_test.go` posts to `/groups`, `/actors` and `/hierarchies` and expects 400 and 201.
