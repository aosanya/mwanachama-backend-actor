package mwanachamaactor

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestACollidingDrawAddsOneGroup(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	um, err := NewUserManager(db, s)
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}

	var asked []int
	drawCode = func(prefix string, groups int) (string, error) {
		asked = append(asked, groups)
		return prefix + strings.Repeat("-ABC", groups), nil
	}
	t.Cleanup(func() { drawCode = drawPublicCode })

	ctx := context.Background()
	first, err := um.CreateActor(ctx, Actor{})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	second, err := um.CreateActor(ctx, Actor{})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	third, err := um.CreateActor(ctx, Actor{})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	if first.PublicCode != "M-ABC-ABC-ABC" || second.PublicCode != "M-ABC-ABC-ABC-ABC" || third.PublicCode != "M-ABC-ABC-ABC-ABC-ABC" {
		t.Errorf("codes = %s, %s, %s; want each collision to add one group", first.PublicCode, second.PublicCode, third.PublicCode)
	}
	if want := []int{3, 3, 4, 3, 4, 5}; !reflect.DeepEqual(asked, want) {
		t.Errorf("draws asked for %v groups, want %v", asked, want)
	}
	group, err := um.CreateGroup(ctx, Group{Name: "Ward"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if group.PublicCode != "G-ABC-ABC-ABC" {
		t.Errorf("group code = %s, want a member code not to collide with a group's", group.PublicCode)
	}
}

func TestADrawUsesOnlyCharactersWithNoLookAlike(t *testing.T) {
	for i := 0; i < 200; i++ {
		code, err := drawPublicCode("M", 3)
		if err != nil {
			t.Fatalf("drawPublicCode: %v", err)
		}
		if strings.ContainsAny(code[2:], "01OILabcdefghijklmnopqrstuvwxyz") {
			t.Fatalf("code %s carries a look-alike", code)
		}
	}
}
