package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	actor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func editGroupTool(t *testing.T, um actor.UserManager) routes.Tool {
	t.Helper()
	tools, err := routes.BuildToolsFor(um, routes.Mount{})
	if err != nil {
		t.Fatalf("BuildTools: %v", err)
	}
	for _, tool := range tools {
		if tool.Name == "actor_edit_group" {
			return tool
		}
	}
	t.Fatal("no actor_edit_group tool")
	return routes.Tool{}
}

func callEditGroup(t *testing.T, tool routes.Tool, args map[string]any) (any, error) {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	return tool.Invoke(context.Background(), raw)
}

func groupToEdit(t *testing.T, um actor.UserManager) actor.Group {
	t.Helper()
	g, err := um.CreateGroup(context.Background(), actor.Group{Name: "Kertváros körzet", NodeType: "Ward"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	return g
}

func versionBefore(t *testing.T, g actor.Group) string {
	t.Helper()
	stamp, err := time.Parse(time.RFC3339Nano, g.LastUpdated)
	if err != nil {
		t.Fatalf("last_updated %q: %v", g.LastUpdated, err)
	}
	return stamp.Add(-time.Minute).Format(time.RFC3339Nano)
}

func TestEditGroupToolDeclaresAnOptionalVersionArgument(t *testing.T) {
	tool := editGroupTool(t, managerOnMemory(t))
	var schema struct {
		Properties           map[string]map[string]any `json:"properties"`
		Required             []string                  `json:"required"`
		AdditionalProperties bool                      `json:"additionalProperties"`
	}
	if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
		t.Fatal(err)
	}
	version, declared := schema.Properties["version"]
	if !declared || version["type"] != "string" {
		t.Fatalf("version is not a declared string argument: %v", schema.Properties)
	}
	if desc, _ := version["description"].(string); desc == "" {
		t.Errorf("version has no description")
	}
	for _, name := range schema.Required {
		if name == "version" {
			t.Errorf("version is required: %v", schema.Required)
		}
	}
	if schema.AdditionalProperties {
		t.Errorf("the schema stopped refusing unknown arguments")
	}
}

func TestEditGroupToolSavesWhenTheVersionIsTheOneRead(t *testing.T) {
	um := managerOnMemory(t)
	tool := editGroupTool(t, um)
	g := groupToEdit(t, um)
	_, err := callEditGroup(t, tool, map[string]any{"group_id": g.ID, "Name": "Újhegy", "NodeType": "Ward", "version": g.LastUpdated})
	if err != nil {
		t.Fatalf("a save on the current version was refused: %v", err)
	}
	got, err := um.GetGroup(context.Background(), g.ID)
	if err != nil || got.Name != "Újhegy" {
		t.Fatalf("group = %+v, %v", got, err)
	}
}

func TestEditGroupToolSavesWithoutAVersion(t *testing.T) {
	um := managerOnMemory(t)
	tool := editGroupTool(t, um)
	g := groupToEdit(t, um)
	if _, err := callEditGroup(t, tool, map[string]any{"group_id": g.ID, "Name": "Újhegy", "NodeType": "Ward"}); err != nil {
		t.Fatalf("a save without a version was refused: %v", err)
	}
}

func TestEditGroupToolRefusesAnOldVersionNamingTheChangedField(t *testing.T) {
	cases := []struct {
		name string
		args map[string]any
		want string
	}{
		{"name", map[string]any{"Name": "Újhegy", "NodeType": "Ward"}, "name changed since you read it"},
		{"node type", map[string]any{"Name": "Kertváros körzet", "NodeType": "County"}, "node_type changed since you read it"},
		{"anchor level", map[string]any{"Name": "Kertváros körzet", "NodeType": "Ward", "AnchorLevelOverrideID": "x"}, "anchor_level_override changed since you read it"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			um := managerOnMemory(t)
			tool := editGroupTool(t, um)
			g := groupToEdit(t, um)
			c.args["group_id"] = g.ID
			c.args["version"] = versionBefore(t, g)
			_, err := callEditGroup(t, tool, c.args)
			if err == nil || err.Error() != c.want {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
			got, getErr := um.GetGroup(context.Background(), g.ID)
			if getErr != nil || got.Name != g.Name || got.NodeType != g.NodeType {
				t.Fatalf("a refused save changed the group: %+v, %v", got, getErr)
			}
		})
	}
}

func TestEditGroupRouteAnswers409ForAnOldVersionAndSavesForTheCurrentOne(t *testing.T) {
	um := managerOnMemory(t)
	srv := serverFor(t, um, routes.Mount{CheckReferences: true})
	g := groupToEdit(t, um)

	status, body := send(t, http.MethodPatch, srv.URL+"/groups/"+g.ID, map[string]any{"name": "Újhegy", "version": versionBefore(t, g)})
	if status != http.StatusConflict || !strings.Contains(string(body), "name changed since you read it") {
		t.Fatalf("old version: %d %s", status, body)
	}

	status, body = send(t, http.MethodPatch, srv.URL+"/groups/"+g.ID, map[string]any{"name": "Újhegy", "version": g.LastUpdated})
	if status != http.StatusOK {
		t.Fatalf("current version: %d %s", status, body)
	}
}
