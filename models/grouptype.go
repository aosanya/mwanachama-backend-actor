package models

// GroupType is a kind of node an organization builds its structure from —
// "Department", "Committee", "Ward Branch", "Faculty", "Service Line". It is
// vocabulary: it declares that the org HAS wards; it creates no ward. The
// nodes themselves are [Group] rows.
//
// Moved here from mwanachama-backend-agency on 2026-09-17 (AGD-017), which
// had briefly carried it after deleting its own Department type. It belongs
// on this side for the same reason Hierarchy/Level did (DSN-1699 gap 1):
// every row that would reference a GroupType — [Group.NodeType] — is already
// stored in this package, and an Agency is a design document that should not
// be the registry for the operational structure's own vocabulary.
//
// [Group.NodeType] is the free-text field a Group carries today. A GroupType
// is what that string is meant to name; nothing enforces the reference yet,
// deliberately — NodeType predates this type and is written by callers
// (including the gateway's GroupEdit) that know nothing about it.
//
// Singular/Plural are what a surface renders the type as; Name is the type's
// own label. LevelID pins the type to one rung, or is "" for a type that may
// sit at any of them. HierarchyID scopes it to one ladder, matching [Level].
type GroupType struct {
	ID string `json:"id"`
	// Code is a stable, human-readable identifier (e.g. "GT-1"), assigned
	// once at creation and never changed afterward — see gormstore's
	// NextCode. Unlike Name, it is never editable.
	Code        string `json:"code"`
	HierarchyID string `json:"hierarchy_id"`
	Name        string `json:"name"`
	Singular    string `json:"singular"`
	Plural      string `json:"plural"`
	LevelID     string `json:"level_id,omitempty"`
}
