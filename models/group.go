package models

const (
	JoinOpen      = "open"
	JoinConfirmed = "confirmed"
)

func ValidJoinMode(mode string) bool { return mode == JoinOpen || mode == JoinConfirmed }

type Group struct {
	ID                    string `json:"id"`
	Code                  string `json:"code"`
	PublicCode            string `json:"public_code"`
	HierarchyID           string `json:"hierarchy_id"`
	LevelID               string `json:"level_id"`
	ParentID              string `json:"parent_id,omitempty"`
	Name                  string `json:"name"`
	Discoverable          bool   `json:"discoverable"`
	JoinMode              string `json:"join_mode"`
	Hidden                bool   `json:"hidden"`
	AnchorLevelOverrideID string `json:"anchor_level_override,omitempty"`
	NodeType              string `json:"node_type,omitempty"`
	CreatedAt             string `json:"created_at"`
	LastUpdated           string `json:"last_updated"`
	Deleted               bool   `json:"deleted"`

	Attributes map[string]any `json:"attributes,omitempty"`
}

func (g Group) Open() bool { return g.JoinMode == JoinOpen }

type GroupEdit struct {
	Name                  string
	NodeType              string
	AnchorLevelOverrideID string
}

func DefaultGroupProperties() []Property {
	return nil
}
