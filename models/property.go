package models

import "fmt"

// PropertyRange is the OWL-style datatype an Attributes entry may hold.
// Mirrors the range vocabulary mwanachama-backend-api-gateway's
// member_data_property table declares (DSN-1664: text/number/boolean/
// enumeration) — this package does not persist a Range anywhere; it exists
// purely so ValidateAttributes has something to check a value against.
type PropertyRange string

const (
	RangeText        PropertyRange = "text"
	RangeNumber      PropertyRange = "number"
	RangeBoolean     PropertyRange = "boolean"
	RangeEnumeration PropertyRange = "enumeration"
)

// Property declares one entry an Actor's or Group's Attributes map may
// carry — the T-Box, not a value. Name is the key that appears in
// Attributes, mirroring member_data_property's convention of no separate
// id. Built-in properties are returned by DefaultActorProperties and
// DefaultGroupProperties; nothing here stops a caller from validating
// against a longer, organization-declared list instead.
type Property struct {
	Name  string        `json:"name"`
	Label string        `json:"label"`
	Range PropertyRange `json:"range"`
	// Options are the permitted values, for RangeEnumeration only.
	Options []string `json:"options,omitempty"`
	// Required rejects a write whose Attributes omits this property or sets
	// it to the empty string / nil.
	Required bool `json:"required"`
	// Unique rejects a write whose Attributes value for this property
	// already appears on another row of the same model. ValidateAttributes
	// does NOT check this — it needs a database lookup, which the root
	// package's checkUniqueAttributes does before every Create, backed by
	// the partial unique index gormstore.Migrate creates for the same
	// property set.
	Unique bool `json:"unique"`
}

// ValidateAttributes checks attrs against every Required and Range/Options
// constraint in properties. A property absent from properties entirely is
// left alone — this is not a closed-schema check, only a validation of the
// properties the caller declared.
func ValidateAttributes(properties []Property, attrs map[string]any) error {
	for _, p := range properties {
		v, present := attrs[p.Name]
		if !present || isBlank(v) {
			if p.Required {
				return fmt.Errorf("%q is required", p.Name)
			}
			continue
		}
		if err := checkRange(p, v); err != nil {
			return fmt.Errorf("%q: %w", p.Name, err)
		}
	}
	return nil
}

// isBlank reports whether v counts as "not set" for a Required check — nil
// or the empty string. An explicit numeric or boolean zero value (0, false)
// is a real value, not an absence, matching this repo's usual omitempty
// convention.
func isBlank(v any) bool {
	if v == nil {
		return true
	}
	s, ok := v.(string)
	return ok && s == ""
}

func checkRange(p Property, v any) error {
	switch p.Range {
	case RangeText:
		if _, ok := v.(string); !ok {
			return fmt.Errorf("must be text, got %T", v)
		}
	case RangeNumber:
		switch v.(type) {
		case float64, float32, int, int32, int64:
		default:
			return fmt.Errorf("must be a number, got %T", v)
		}
	case RangeBoolean:
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("must be a boolean, got %T", v)
		}
	case RangeEnumeration:
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("must be text (an enumeration option), got %T", v)
		}
		for _, opt := range p.Options {
			if opt == s {
				return nil
			}
		}
		return fmt.Errorf("%q is not one of %v", s, p.Options)
	}
	return nil
}
