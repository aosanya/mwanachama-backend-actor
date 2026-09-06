package gormstore

import (
	"encoding/json"

	"gorm.io/datatypes"
)

// marshalStrings and unmarshalStrings round-trip a []string through
// datatypes.JSON — RoleKind.Capabilities' storage shape, a plain JSON array
// rather than the JSONMap (object) shape ActorRow/GroupRow's Attributes use.
func marshalStrings(ss []string) (datatypes.JSON, error) {
	if ss == nil {
		ss = []string{}
	}
	b, err := json.Marshal(ss)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(b), nil
}

func unmarshalStrings(raw datatypes.JSON) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
