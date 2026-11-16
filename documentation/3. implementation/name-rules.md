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

## Order on the group edit

`EditGroup` checks the name before it looks the group up (hub issue ACT74), so `PATCH /groups/{id}` with a name of more than 200 characters answers 400 even when the id is unknown, and only a good name reaches the 404. This is the order the route wrapper had before the check moved into the manager. The edit keeps its original wording, "a name is required" and "a name may be at most 200 characters"; create group says "a group name ...".

## Padding

A name is trimmed first, then counted, then stored, on all six writes: create group, edit group, create actor, rename actor, create hierarchy and rename hierarchy (hub issue ACT75). `checkRequiredName` in `name_check.go` returns the trimmed name and each write stores that value, so `"  Padded  "` is stored as `"Padded"` and a 200 character name with a space on either side is accepted. The edit route no longer trims on its own; the manager does it for every caller. White space is whatever `strings.TrimSpace` removes, which includes the no-break space and the ideographic space.

The duplicate checks still fold spaces and look-alike letters (`plaintext.FoldDrawnName` for groups and hierarchies, `plaintext.FoldLookalike` for display names), so a padded copy of a taken name is refused with 409.

## Where the numbers come from

200 is the limit the group edit route already enforced (`maxGroupName`, now removed from `routes/routes.go` because the manager enforces it) and the same number as the limit of an agency name, `MaxInstanceNameBytes = 200` in the hub's `internal/throttle/limits.go`. The two do not count alike: the agency name limit counts bytes (the hub's refusal reads "an agency's name may be at most 200 bytes"), while these three names count characters (runes). A name of 200 accented Hungarian letters is 200 characters and about 400 bytes, so it is kept here and would be refused as an agency name; a name of 200 characters is within 200 bytes only when every character is one byte. No new number was chosen.

## One exception

An actor display name that is exactly empty is still accepted. A member is minted from a device or a phone number before any name is given, and the existing tests and registration flow depend on it. A display name of only spaces is refused.

## Not covered

First and last names, level names, group type names, role kind names and group name translations keep the plaintext check only, with no length limit and no blank refusal. They are the subject of other hub issues (MUN679 and MUN680 call the same check).

## Proof

`name_length_test.go` drives the six writes in the table with 200, 201, 1000 and 100000 characters and with blank names. `routes/name_length_test.go` posts to `/groups`, `/actors` and `/hierarchies` and expects 400 and 201.
