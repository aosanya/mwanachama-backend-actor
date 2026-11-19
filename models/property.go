package models

import (
	"fmt"
	"time"
)

type PropertyRange string

const (
	RangeText        PropertyRange = "text"
	RangeNumber      PropertyRange = "number"
	RangeBoolean     PropertyRange = "boolean"
	RangeEnumeration PropertyRange = "enumeration"
	RangeDate        PropertyRange = "date"
)

const dayLayout = "2006-01-02"

type Property struct {
	Name      string        `json:"name"`
	Label     string        `json:"label"`
	Range     PropertyRange `json:"range"`
	Options   []string      `json:"options,omitempty"`
	Required  bool          `json:"required"`
	Unique    bool          `json:"unique"`
	NotFuture bool          `json:"not_future,omitempty"`
}

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
	case RangeDate:
		return checkDay(p, v)
	}
	return nil
}

func checkDay(p Property, v any) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("must be a day written YYYY-MM-DD, got %T", v)
	}
	day, err := time.Parse(dayLayout, s)
	if err != nil {
		return fmt.Errorf("%q is not a day written YYYY-MM-DD", s)
	}
	if p.NotFuture && day.After(time.Now().UTC()) {
		return fmt.Errorf("%q is a day that has not come yet", s)
	}
	return nil
}
