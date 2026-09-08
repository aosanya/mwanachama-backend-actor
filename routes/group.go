// group.go — HTTP routes over the three Group writes group_impl.go backs:
// CreateGroup, EditGroup, MoveGroup. See doc.go for what is and is not in
// scope and why.
package routes

import (
	"errors"
	"net/http"
	"strings"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

// statusFor maps this package's own error sentinels to a status code, the
// same triage the gateway's storeErr already does for the pre-move
// equivalents (chapter.ErrNotFound -> 404, the three structure refusals ->
// 409, domerr.ErrInvalidReference -> 400). The status codes are unchanged
// from before this move; the message text is this package's own vocabulary
// rather than a byte-for-byte copy of the gateway's old chapter.* wording —
// a deliberate, recorded trade-off (see mwanachama-backend-api-gateway's
// CLAUDE.md/todo.md decision record for this move), not an oversight.
func statusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaactor.ErrGroupNotFound):
		return http.StatusNotFound
	case errors.Is(err, mwanachamaactor.ErrRootCannotMove),
		errors.Is(err, mwanachamaactor.ErrParentIsSelf),
		errors.Is(err, mwanachamaactor.ErrParentInSubtree):
		return http.StatusConflict
	case errors.Is(err, mwanachamaactor.ErrParentNotFound),
		errors.Is(err, mwanachamaactor.ErrInvalidGroup),
		errors.Is(err, mwanachamaactor.ErrDuplicateAttribute):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// writeGroupErr answers err on the wire, logging nothing — this package has
// no logger of its own; a 500 here is rare enough (a store failure with no
// sentinel) that the mounting process's own request/error logging
// middleware, wrapped around every route regardless of where it is defined,
// is what a deployment relies on to see it.
func writeGroupErr(w http.ResponseWriter, err error) {
	code := statusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// checkHierarchyRefs runs CreateGroup/EditGroup's shared reference checks:
// hierarchyID (when non-empty) must exist, and levelID/anchorLevelOverrideID
// (each, when non-empty) must exist and belong to hierarchyID. Returns a
// user-facing message and false on the first failure; ok=true and no
// message otherwise.
func checkHierarchyRefs(hc HierarchyChecker, r *http.Request, hierarchyID, levelID, anchorLevelOverrideID string) (msg string, ok bool, err error) {
	ctx := r.Context()
	if hierarchyID != "" {
		exists, err := hc.HierarchyExists(ctx, hierarchyID)
		if err != nil {
			return "", false, err
		}
		if !exists {
			return "invalid reference", false, nil
		}
	}
	for _, id := range []string{levelID, anchorLevelOverrideID} {
		if id == "" {
			continue
		}
		inHierarchy, err := hc.LevelInHierarchy(ctx, id, hierarchyID)
		if err != nil {
			return "", false, err
		}
		if !inHierarchy {
			return "invalid reference", false, nil
		}
	}
	return "", true, nil
}

// CreateGroup handles POST — decode, validate the hierarchy/level/parent
// references, create, encode. Discoverable defaults true when the request
// omits the key, the same seed-before-decode the gateway's old createChapter
// used (DEV-1516): encoding/json leaves an absent key's field untouched, so
// seeding before Decode is the only way an omitted key and an explicit
// `"discoverable":false` read differently afterwards.
func CreateGroup(um mwanachamaactor.UserManager, hc HierarchyChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in := mwanachamaactor.Group{Discoverable: true}
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if in.HierarchyID == "" {
			writeErr(w, http.StatusBadRequest, "hierarchy_id is required")
			return
		}
		if msg, ok, err := checkHierarchyRefs(hc, r, in.HierarchyID, in.LevelID, in.AnchorLevelOverrideID); err != nil {
			writeGroupErr(w, err)
			return
		} else if !ok {
			writeErr(w, http.StatusBadRequest, msg)
			return
		}
		if in.ParentID != "" {
			if _, err := um.GetGroup(r.Context(), in.ParentID); err != nil {
				if errors.Is(err, mwanachamaactor.ErrGroupNotFound) {
					writeErr(w, http.StatusBadRequest, "invalid reference")
					return
				}
				writeGroupErr(w, err)
				return
			}
		}
		out, err := um.CreateGroup(r.Context(), in)
		if err != nil {
			writeGroupErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// groupEditBody is the wire shape for EditGroup — mirrors [mwanachamaactor.GroupEdit]
// field-for-field; a distinct type only so the DisallowUnknownFields readJSON
// already applies keeps rejecting a request that names a field GroupEdit does
// not have, without this package needing json tags on GroupEdit itself.
type groupEditBody struct {
	Name                  string `json:"name"`
	NodeType              string `json:"node_type"`
	AnchorLevelOverrideID string `json:"anchor_level_override"`
}

// EditGroup handles PATCH {id} — decode, validate AnchorLevelOverrideID
// against the group's own HierarchyID when supplied, edit, encode. The
// mounting process's own route pattern must name the id path segment
// "groupID" (e.g. "PATCH /v1/member/chapters/{groupID}") — this package
// owns no path text of its own (see doc.go), only this one path-value key.
//
// Name is required and capped at 200 runes, and NodeType is trimmed —
// ported verbatim from the gateway's old editChapter handler (the door M57
// draws), which enforced both before this move and did so in the HTTP
// layer rather than in group_impl.go's EditGroup, whose own doc still says
// "empty clears" for the three columns it writes. That contract is
// unchanged for NodeType/AnchorLevelOverrideID; Name is the one field this
// handler still refuses to clear, on purpose, matching the door it
// replaces.
func EditGroup(um mwanachamaactor.UserManager, hc HierarchyChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("groupID")
		var body groupEditBody
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		body.Name = strings.TrimSpace(body.Name)
		if body.Name == "" {
			writeErr(w, http.StatusBadRequest, "a name is required")
			return
		}
		if len([]rune(body.Name)) > 200 {
			writeErr(w, http.StatusBadRequest, "a name may be at most 200 characters")
			return
		}
		body.NodeType = strings.TrimSpace(body.NodeType)
		if body.AnchorLevelOverrideID != "" {
			current, err := um.GetGroup(r.Context(), id)
			if err != nil {
				writeGroupErr(w, err)
				return
			}
			if msg, ok, err := checkHierarchyRefs(hc, r, current.HierarchyID, "", body.AnchorLevelOverrideID); err != nil {
				writeGroupErr(w, err)
				return
			} else if !ok {
				writeErr(w, http.StatusBadRequest, msg)
				return
			}
		}
		out, err := um.EditGroup(r.Context(), id, mwanachamaactor.GroupEdit{
			Name:                  body.Name,
			NodeType:              body.NodeType,
			AnchorLevelOverrideID: body.AnchorLevelOverrideID,
		})
		if err != nil {
			writeGroupErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// MoveGroup handles POST {id}/move — decode the new parent, move, encode.
// No HierarchyChecker: MoveGroup re-parents within the same tree and never
// touches HierarchyID/LevelID, so every reference it needs to check
// (self-parent, cycle, missing parent) is already answered inside
// group_impl.go's own MoveGroup. Same "groupID" path-value requirement as
// EditGroup.
//
// An empty parent_id is refused rather than read as "move to root" —
// group_impl.go's MoveGroup would otherwise treat "" as a real, if odd,
// destination (there being exactly one root, and this not being it); ported
// from the gateway's old moveChapter handler, which drew this refusal for
// the same reason: a caller sends this route only when the parent actually
// changed, and "" is never that.
func MoveGroup(um mwanachamaactor.UserManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("groupID")
		var body struct {
			ParentID string `json:"parent_id"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if body.ParentID == "" {
			writeErr(w, http.StatusBadRequest, "a new parent group is required")
			return
		}
		out, err := um.MoveGroup(r.Context(), id, body.ParentID)
		if err != nil {
			writeGroupErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}
