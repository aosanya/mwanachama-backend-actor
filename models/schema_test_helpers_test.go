package models_test

import (
	"reflect"
	"testing"
)

// assertField is hand-written, reusable test infra — the one piece
// developer/tools/domain-schema/cmd/gen's generated schema_generated_test.go
// calls into rather than duplicating per generated assertion. Checks a
// struct's field at index i by name, Go type, and json struct tag.
func assertField(t *testing.T, typ reflect.Type, i int, wantName, wantType, wantTag string) {
	t.Helper()
	f := typ.Field(i)
	if f.Name != wantName {
		t.Errorf("field %d: name = %q, want %q", i, f.Name, wantName)
	}
	if f.Type.String() != wantType {
		t.Errorf("field %d (%s): type = %s, want %s", i, f.Name, f.Type.String(), wantType)
	}
	if string(f.Tag) != wantTag {
		t.Errorf("field %d (%s): tag = %q, want %q", i, f.Name, f.Tag, wantTag)
	}
}
