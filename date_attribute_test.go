package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	actor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func TestAResidentCarriesADateOfBirthAsADay(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	a, _ := um.CreateActor(ctx, actor.Actor{DisplayName: "Születési Sára"})
	kept, err := um.SetActorAttributes(ctx, a.ID, "staff-1", models.AttributeCorrection{Attributes: map[string]any{"date_of_birth": "1984-03-09"}})
	if err != nil {
		t.Fatalf("SetActorAttributes: %v", err)
	}
	if kept.Attributes["date_of_birth"] != "1984-03-09" {
		t.Errorf("the day was not kept: %+v", kept.Attributes)
	}
	read, err := um.GetActor(ctx, a.ID)
	if err != nil || read.Attributes["date_of_birth"] != "1984-03-09" {
		t.Errorf("the day is not read back: %+v %v", read.Attributes, err)
	}
}

func TestADateOfBirthThatIsNotADayIsRefused(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	a, _ := um.CreateActor(ctx, actor.Actor{DisplayName: "Rossz Rezső"})
	for _, bad := range []any{"1984-02-30", "09/03/1984", "next spring", 1984, true} {
		if _, err := um.SetActorAttributes(ctx, a.ID, "staff-1", models.AttributeCorrection{Attributes: map[string]any{"date_of_birth": bad}}); !errors.Is(err, actor.ErrInvalidActor) {
			t.Errorf("%v was accepted as a date of birth: %v", bad, err)
		}
	}
}

func TestADateOfBirthInTheFutureIsRefused(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	a, _ := um.CreateActor(ctx, actor.Actor{DisplayName: "Jövő Jolán"})
	tomorrow := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	if _, err := um.SetActorAttributes(ctx, a.ID, "staff-1", models.AttributeCorrection{Attributes: map[string]any{"date_of_birth": tomorrow}}); !errors.Is(err, actor.ErrInvalidActor) {
		t.Errorf("a birth day tomorrow was accepted: %v", err)
	}
	today := time.Now().UTC().Format("2006-01-02")
	if _, err := um.SetActorAttributes(ctx, a.ID, "staff-1", models.AttributeCorrection{Attributes: map[string]any{"date_of_birth": today}}); err != nil {
		t.Errorf("a birth day today was refused: %v", err)
	}
}

func TestADateOfBirthIsOptionalAndTwoPeopleMayShareOne(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	a, err := um.CreateActor(ctx, actor.Actor{DisplayName: "Ikerke Ilka", Attributes: map[string]any{"date_of_birth": "2001-05-05"}})
	if err != nil {
		t.Fatalf("CreateActor with a date of birth: %v", err)
	}
	b, err := um.CreateActor(ctx, actor.Actor{DisplayName: "Ikerke Imre", Attributes: map[string]any{"date_of_birth": "2001-05-05"}})
	if err != nil {
		t.Fatalf("a second person with the same day: %v", err)
	}
	if a.ID == b.ID {
		t.Errorf("one person twice")
	}
	none, err := um.CreateActor(ctx, actor.Actor{DisplayName: "Nincs Nóra"})
	if err != nil {
		t.Fatalf("a person without a date of birth: %v", err)
	}
	cleared, err := um.SetActorAttributes(ctx, a.ID, "staff-1", models.AttributeCorrection{Attributes: map[string]any{"date_of_birth": ""}})
	if err != nil {
		t.Fatalf("clearing the day: %v", err)
	}
	if _, still := cleared.Attributes["date_of_birth"]; still || none.Attributes["date_of_birth"] != nil {
		t.Errorf("an emptied day is still held: %+v", cleared.Attributes)
	}
}

func TestTheAttributePropertiesListTheDateOfBirthAsADate(t *testing.T) {
	um := newManager(t)
	properties, err := um.AttributeProperties(context.Background())
	if err != nil {
		t.Fatalf("AttributeProperties: %v", err)
	}
	for _, property := range properties {
		if property.Name == "date_of_birth" {
			if property.Range != models.RangeDate || property.Label != "Date of birth" || property.Required || property.Unique {
				t.Errorf("date_of_birth = %+v", property)
			}
			return
		}
	}
	t.Errorf("no date_of_birth among %+v", properties)
}
