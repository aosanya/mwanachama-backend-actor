# Group names in several languages

ACT29, member app decision 256 (2026-10-22). A group's name is written once,
in whatever language its author used. An operator may add the same name in
other languages; only a translation somebody has reviewed reaches members.

## The object

`group_name_translation` holds one row per group and language, held unique by
the `group_language` index. It carries the translated `name`, whether it is
`reviewed`, and who reviewed it and when (`reviewed_by`, `reviewed_at`). The
group's own `name` is untouched: it stays the original, and there is no
"original language" field, because a group with no translations needs none.

A language is a lowercase tag such as `en`, `sw` or `pt-br`
(`^[a-z]{2,3}(-[a-z0-9]{2,8})*$`, after trimming and lowercasing). This
module does not know which languages an agency offers; the agency's own list
(decision 255) is the host's to enforce.

## Writing

- `SetGroupNameTranslation(group, language, name)` adds or changes one
  language's name. A **changed** name clears its review, so an edit can never
  reach members unreviewed. Sending the **same** name leaves the review alone,
  so a client resaving a form does not throw away a review.
- `ReviewGroupNameTranslation(group, language, reviewer)` marks it reviewed.
  Over the wire the reviewer is the caller (`from: caller`); a body naming
  someone else is ignored.
- `ListGroupNameTranslations(group)` returns every translation, reviewed or
  not, ordered by language — the operator's view.

## Reading in a language

`GetGroupInLanguage(group, language)` returns the group with its name in that
language when a reviewed translation exists, and the original name otherwise
— an unreviewed translation, a language with none, and no language at all all
read as the original. `GroupNamesInLanguage(language, groups)` is the same
rule for many groups at once, answering only the groups that have a reviewed
translation, so a host listing groups replaces those names and leaves the
rest.

## Addresses

| Method | Path | Action |
| --- | --- | --- |
| GET | `/groups/{group_id}/names` | `actor.group_name.list` |
| PUT | `/groups/{group_id}/names/{language}` | `actor.group_name.translate` |
| POST | `/groups/{group_id}/names/{language}/review` | `actor.group_name.review` |
| GET | `/groups/{group_id}/in/{language}` | `actor.group.read_in_language` |

Not done here: drafting a translation with the chat, and passing a member's
language through the host's group lists. Both are wakala-api WK177 and the
studio's WKS134.
