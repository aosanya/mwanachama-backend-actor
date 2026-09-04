// assignment.go — HTTP routes over registration_impl.go's plain
// ActorGroupAssignment operations: AssignGroup, Deregister,
// ListGroupsForActor, ListActorsForGroup, HomeCounts. See doc.go for scope.
//
// The gateway's own register/deregister/listMembersForChapter/
// listRegistrationsForMember are NOT replaced by these — DEV-1129's
// self-or-operator check, DEV-1268's discoverable-chapter gate, DEV-1614's
// anchor gate, DEV-1604's membership cap, DSN-1680's agentic-seat gate, and
// DEV-1343/DEV-1344's act-log composition on deregister all depend on
// domains (Role, org_policy, agentic, custody) this package must not
// import. What is here is the same "decode, call UserManager, encode"
// shell Group's three writes already are: safe to mount once a caller has
// already cleared whatever gate the mounting process puts in front of it.
package routes

import (
	"errors"
	"net/http"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

// assignmentStatusFor maps this package's ActorGroupAssignment error
// sentinels to a status code.
func assignmentStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaactor.ErrActorNotFound),
		errors.Is(err, mwanachamaactor.ErrGroupNotFound):
		return http.StatusNotFound
	case errors.Is(err, mwanachamaactor.ErrDuplicateAttribute):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeAssignmentErr(w http.ResponseWriter, err error) {
	code := assignmentStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// assignBody is the wire shape for AssignGroup — every field
// ActorGroupAssignment carries except ActorID, which the path supplies.
// There is no dedicated is_home/joined_at key: neither is a column any
// more (see models.ActorGroupAssignment's doc) — a caller that wants
// either sends it as an ordinary Attributes entry, e.g.
// {"attributes":{"is_home":true}}, the same way it would send any other
// organization-declared property.
type assignBody struct {
	GroupID    string         `json:"group_id"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// AssignGroup handles POST {actorID}/{assignment} — enrol the path actor at
// body.GroupID (upsert on the pair, per AssignGroup's own contract; no
// exclusivity on a caller-declared "is_home" Attributes entry — decision
// 8). The mounting process's own route pattern must name the actor's id
// path segment "actorID". Neither group_id's existence nor whether it is
// "discoverable" is checked here — AssignGroup itself checks neither (see
// registration_impl.go), and the gateway's DEV-1268 discoverable-chapter
// gate is exactly the kind of mounting-process policy this package does
// not reimplement.
func AssignGroup(um mwanachamaactor.UserManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body assignBody
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		in := models.ActorGroupAssignment{
			ActorID:    r.PathValue("actorID"),
			GroupID:    body.GroupID,
			Attributes: body.Attributes,
		}
		out, err := um.AssignGroup(r.Context(), in)
		if err != nil {
			writeAssignmentErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// Deregister handles DELETE {actorID}/{assignment}/{groupID} — removes the
// row if one exists; idempotent, so a caller naming a pair that was never
// assigned still gets 204. No act-log row is written — see
// UserManager.Deregister's doc; a mounting process that needs one composes
// it itself from the pre-delete state, which this package cannot return
// once the row is gone (it can only report found=true/false, discarded
// here in favour of the same 204 either way).
func Deregister(um mwanachamaactor.UserManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _, err := um.Deregister(r.Context(), r.PathValue("actorID"), r.PathValue("groupID"))
		if err != nil {
			writeAssignmentErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ListGroupsForActor handles GET {actorID}/{assignment} — every group the
// path actor is assigned to, created_at order. No visibility fencing: the
// gateway's own listRegistrationsForMember requireMemberVisible gate
// (DEV-1128 — self or co-member) is not reproduced here.
func ListGroupsForActor(um mwanachamaactor.UserManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := um.ListGroupsForActor(r.Context(), r.PathValue("actorID"))
		if err != nil {
			writeAssignmentErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListActorsForGroup handles GET {groupID}/{assignment} — every actor
// assigned to the path group, created_at order. No visibility fencing: the
// gateway's own listMembersForChapter requireChapterMember gate is not
// reproduced here.
func ListActorsForGroup(um mwanachamaactor.UserManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := um.ListActorsForGroup(r.Context(), r.PathValue("groupID"))
		if err != nil {
			writeAssignmentErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// HomeCounts handles GET {assignment}/home-counts — actors-per-group,
// counting only the assignments whose own Attributes["is_home"] is true
// (see UserManager.HomeCounts' doc — there is no dedicated column).
// Groups nobody calls home are absent, not zero.
func HomeCounts(um mwanachamaactor.UserManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := um.HomeCounts(r.Context())
		if err != nil {
			writeAssignmentErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}
