package plates

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestDraftConvertsToPlate(t *testing.T) {
	draft := NewDraft()
	draft.Name = "nxc_ldap_commands"
	draft.Category = "ldap"
	draft.Description = "Common NetExec LDAP command helper"
	draft.AddTag("nxc")
	draft.AddLine("nxc ldap {{target}}")
	if err := draft.AddRequiredVar("target", "Target IP address or hostname"); err != nil {
		t.Fatalf("AddRequiredVar() error = %v", err)
	}

	plate := draft.ToPlate()
	if plate.Name != draft.Name || plate.Category != draft.Category {
		t.Fatalf("ToPlate() = %#v", plate)
	}
	if plate.Template != "nxc ldap {{target}}\n" {
		t.Fatalf("Template = %q", plate.Template)
	}
}

func TestDraftAddsRequiredVariables(t *testing.T) {
	draft := NewDraft()
	if err := draft.AddRequiredVar("target", "Target host"); err != nil {
		t.Fatalf("AddRequiredVar() error = %v", err)
	}
	ingredient := draft.Ingredients["target"]
	if !ingredient.Required || ingredient.Description != "Target host" {
		t.Fatalf("ingredient = %#v", ingredient)
	}
}

func TestDraftAddsOptionalVariablesWithDefaults(t *testing.T) {
	draft := NewDraft()
	if err := draft.AddOptionalVar("rate", "5000", "Minimum packet rate"); err != nil {
		t.Fatalf("AddOptionalVar() error = %v", err)
	}
	ingredient := draft.Ingredients["rate"]
	if ingredient.Required || ingredient.Default != "5000" {
		t.Fatalf("ingredient = %#v", ingredient)
	}
}

func TestDraftLineEditing(t *testing.T) {
	draft := NewDraft()
	draft.AddLine("line 1")
	draft.AddLine("line 3")
	if err := draft.InsertLine(2, "line 2"); err != nil {
		t.Fatalf("InsertLine() error = %v", err)
	}
	if err := draft.DeleteLine(1); err != nil {
		t.Fatalf("DeleteLine() error = %v", err)
	}
	if strings.Join(draft.Lines, "|") != "line 2|line 3" {
		t.Fatalf("Lines = %#v", draft.Lines)
	}
	draft.ClearLines()
	if len(draft.Lines) != 0 {
		t.Fatalf("Lines = %#v", draft.Lines)
	}
}

func TestDraftValidationRequiresCoreFields(t *testing.T) {
	draft := NewDraft()
	_, err := draft.ValidateDraft()
	if err == nil {
		t.Fatal("ValidateDraft() error = nil, want missing fields")
	}
	for _, field := range []string{"name", "category", "description", "template line"} {
		if !strings.Contains(err.Error(), field) {
			t.Fatalf("ValidateDraft() error %q missing %q", err, field)
		}
	}
}

func TestDraftValidationRejectsInvalidIngredientNames(t *testing.T) {
	draft := validDraft()
	draft.Ingredients["1target"] = Ingredient{Description: "bad", Required: true}
	_, err := draft.ValidateDraft()
	if err == nil {
		t.Fatal("ValidateDraft() error = nil, want invalid ingredient")
	}
}

func TestDraftSaveWritesCorrectRackPath(t *testing.T) {
	root := t.TempDir()
	rack := filepath.Join(root, "data", "rack")
	draft := validDraft()

	path, err := draft.Save(rack, false)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	want := filepath.ToSlash(filepath.Join(rack, "ldap", "nxc_ldap_commands.yml"))
	if path != want {
		t.Fatalf("Save() path = %q, want %q", path, want)
	}
}

func TestDraftSaveRefusesOverwriteUnlessForced(t *testing.T) {
	root := t.TempDir()
	rack := filepath.Join(root, "data", "rack")
	draft := validDraft()
	if _, err := draft.Save(rack, false); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}
	if _, err := draft.Save(rack, false); !errors.Is(err, ErrPlateExists) {
		t.Fatalf("second Save() error = %v, want ErrPlateExists", err)
	}
	if _, err := draft.Save(rack, true); err != nil {
		t.Fatalf("forced Save() error = %v", err)
	}
}

func TestSavedDraftCanBeLoadedByRackLoader(t *testing.T) {
	root := t.TempDir()
	rack := filepath.Join(root, "data", "rack")
	draft := validDraft()
	if _, err := draft.Save(rack, false); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	plate, err := NewRackRepository(root, rack).Load("ldap/nxc_ldap_commands")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if plate.Name != draft.Name {
		t.Fatalf("Name = %q", plate.Name)
	}
}

func validDraft() *Draft {
	draft := NewDraft()
	draft.Name = "nxc_ldap_commands"
	draft.Category = "ldap"
	draft.Description = "Common NetExec LDAP command helper"
	draft.AddLine("# LDAP command helper for {{target}}")
	draft.AddLine("nxc ldap {{target}} -u {{username}} -p {{password}}")
	draft.AddTag("nxc")
	draft.AddTag("ldap")
	_ = draft.AddRequiredVar("target", "Target IP address or hostname")
	_ = draft.AddRequiredVar("username", "Username")
	_ = draft.AddRequiredVar("password", "Password or quoted empty string")
	return draft
}
