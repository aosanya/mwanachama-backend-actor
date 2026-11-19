package mwanachamaactor_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func TestAttributePropertiesListsPhoneEmailAndDateOfBirthWithLabels(t *testing.T) {
	um := newManager(t)

	got, err := um.AttributeProperties(context.Background())
	if err != nil {
		t.Fatalf("AttributeProperties: %v", err)
	}

	want := []models.Property{
		{Name: "phone", Label: "Phone", Range: models.RangeText, Unique: true, Required: false},
		{Name: "email", Label: "Email", Range: models.RangeText, Unique: true, Required: false},
		{Name: "date_of_birth", Label: "Date of birth", Range: models.RangeDate, Unique: false, Required: false},
	}
	if len(got) != len(want) {
		t.Fatalf("AttributeProperties returned %d entries, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Name != want[i].Name || got[i].Label != want[i].Label ||
			got[i].Range != want[i].Range || got[i].Unique != want[i].Unique ||
			got[i].Required != want[i].Required {
			t.Fatalf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
