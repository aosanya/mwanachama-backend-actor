package routes

import (
	"fmt"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/spec"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

type Tool = dispatch.Tool

var carriers = map[string]string{
	"actor":                  "Actor",
	"group":                  "Group",
	"actor_group_assignment": "ActorGroupAssignment",
	"role_kind":              "RoleKind",
	"actor_role_assignment":  "ActorRoleAssignment",
	"hierarchy":              "Hierarchy",
	"level":                  "Level",
	"group_type":             "GroupType",
	"code_sequence":          "CodeSequence",
}

var storeOwned = map[string]bool{
	"Actor.code":                       true,
	"Actor.deleted":                    true,
	"Group.code":                       true,
	"Group.deleted":                    true,
	"ActorGroupAssignment.deleted":     true,
	"RoleKind.code":                    true,
	"RoleKind.retired_at":              true,
	"RoleKind.retired_by":              true,
	"ActorRoleAssignment.active":       true,
	"ActorRoleAssignment.ended_at":     true,
	"ActorRoleAssignment.ended_reason": true,
	"Hierarchy.code":                   true,
	"Level.code":                       true,
	"GroupType.code":                   true,
}

var editFields = map[string]dispatch.FieldDoc{
	"GroupEdit.Name": {
		Description: "The group's new name. Required, and at most 200 characters.",
		Required:    true,
	},
	"GroupEdit.NodeType": {
		Description: "The group type this group is shown as. Empty clears it.",
	},
	"GroupEdit.AnchorLevelOverrideID": {
		Description: "The level that anchors this group's subtree, where it differs from the hierarchy's default. Empty clears it.",
	},
}

func Fields() (map[string]dispatch.FieldDoc, error) {
	b, err := mwanachamaactor.Blueprint()
	if err != nil {
		return nil, err
	}
	out := map[string]dispatch.FieldDoc{}
	for _, role := range b.Roles() {
		carrier, named := carriers[role]
		if !named {
			return nil, fmt.Errorf("actor tools: role %q has no type to describe", role)
		}
		o, ok := b.Object(role)
		if !ok {
			return nil, fmt.Errorf("actor tools: the blueprint declares no %q", role)
		}
		for _, f := range o.Fields {
			key := carrier + "." + f.Name
			out[key] = dispatch.FieldDoc{
				Description: f.Description,
				Required:    f.Required,
				ReadOnly:    f.Primary || f.Type == spec.TypeTimestamp || storeOwned[key],
				Values:      f.Values,
			}
		}
	}
	for key, doc := range editFields {
		out[key] = doc
	}
	return out, nil
}

func BuildTools(um mwanachamaactor.UserManager) ([]Tool, error) {
	return BuildToolsFor(um, Mount{CheckReferences: true})
}

func BuildToolsFor(um mwanachamaactor.UserManager, m Mount) ([]Tool, error) {
	s, err := operations()
	if err != nil {
		return nil, err
	}
	fields, err := Fields()
	if err != nil {
		return nil, err
	}
	return dispatch.Tools(s, dispatch.Deps{
		Manager:   managerFor(um, m),
		Errors:    sentinels,
		Fields:    fields,
		Authorize: m.Authorize,
		Caller:    m.Caller,
	})
}

func Tools(um mwanachamaactor.UserManager) []Tool {
	return ToolsFor(um, Mount{CheckReferences: true})
}

func ToolsFor(um mwanachamaactor.UserManager, m Mount) []Tool {
	out, err := BuildToolsFor(um, m)
	if err != nil {
		panic(fmt.Sprintf("actor tools: %v", err))
	}
	return out
}
