package plates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLintValidPlatePasses(t *testing.T) {
	result := LintPlate(lintValidPlate())
	if result.Status() != "PASS" {
		t.Fatalf("Status() = %s, issues = %#v", result.Status(), result.Issues)
	}
}

func TestLintMissingMetadataFails(t *testing.T) {
	plate := readFixturePlate(t, "missing_description.yml")
	result := LintPlate(plate)
	if result.Status() != "FAIL" {
		t.Fatalf("Status() = %s, issues = %#v", result.Status(), result.Issues)
	}
	if !hasIssue(result, SeverityError, "missing required metadata: description") {
		t.Fatalf("missing description issue not found: %#v", result.Issues)
	}
}

func TestLintUndeclaredVariablesFail(t *testing.T) {
	plate := readFixturePlate(t, "undeclared_reference.yml")
	result := LintPlate(plate)
	if result.Status() != "FAIL" {
		t.Fatalf("Status() = %s, issues = %#v", result.Status(), result.Issues)
	}
	if !hasIssue(result, SeverityError, "template references undeclared ingredient: target") {
		t.Fatalf("undeclared target issue not found: %#v", result.Issues)
	}
}

func TestLintUnusedIngredientsWarn(t *testing.T) {
	plate := readFixturePlate(t, "unused_variable.yml")
	result := LintPlate(plate)
	if result.Status() != "WARN" {
		t.Fatalf("Status() = %s, issues = %#v", result.Status(), result.Issues)
	}
	if !hasIssue(result, SeverityWarn, "ingredient declared but unused: rate") {
		t.Fatalf("unused rate issue not found: %#v", result.Issues)
	}
}

func TestLintDuplicateTagsWarn(t *testing.T) {
	plate := readFixturePlate(t, "duplicate_tags.yml")
	result := LintPlate(plate)
	if result.Status() != "WARN" {
		t.Fatalf("Status() = %s, issues = %#v", result.Status(), result.Issues)
	}
	if !hasIssue(result, SeverityWarn, "duplicate tag: nmap") {
		t.Fatalf("duplicate tag issue not found: %#v", result.Issues)
	}
}

func TestLintDuplicatePlateNamesDetected(t *testing.T) {
	index := NewRackIndex("data/rack", []Plate{
		lintPlateWithName("nmap_scan", "scanning"),
		lintPlateWithName("nmap_scan", "windows"),
	})
	results := LintRack(index)
	for _, result := range results {
		if !hasIssue(result, SeverityWarn, "duplicate plate name exists in multiple categories") {
			t.Fatalf("duplicate name warning missing from %#v", results)
		}
	}
}

func TestRackWideLinting(t *testing.T) {
	index := NewRackIndex("data/rack", []Plate{
		lintValidPlate(),
		readFixturePlate(t, "unused_variable.yml"),
		readFixturePlate(t, "undeclared_reference.yml"),
	})
	results := LintRack(index)
	health := RackHealthFromResults(index, results)
	if health.Passing != 1 || health.Warning != 1 || health.Failing != 1 {
		t.Fatalf("health = %#v", health)
	}
}

func TestHealthSummaryGeneration(t *testing.T) {
	index := NewRackIndex("data/rack", []Plate{
		lintValidPlate(),
		readFixturePlate(t, "unused_variable.yml"),
		readFixturePlate(t, "undeclared_reference.yml"),
	})
	health := RackHealthFromResults(index, LintRack(index))
	if health.TotalPlates != 3 {
		t.Fatalf("TotalPlates = %d", health.TotalPlates)
	}
	if health.UnusedIngredients != 1 {
		t.Fatalf("UnusedIngredients = %d", health.UnusedIngredients)
	}
	if health.UndeclaredVariables != 1 {
		t.Fatalf("UndeclaredVariables = %d", health.UndeclaredVariables)
	}
}

func TestLintWarnsLikelySensitiveVariableNotSecret(t *testing.T) {
	plate := lintValidPlate()
	plate.Ingredients["password"] = Ingredient{Description: "Password", Required: true}
	plate.Template = "tool {{target}} {{password}}"
	result := LintPlate(plate)
	if !hasIssue(result, SeverityWarn, "likely sensitive ingredient not marked secret: password") {
		t.Fatalf("sensitive warning missing: %#v", result.Issues)
	}
}

func readFixturePlate(t *testing.T, name string) Plate {
	t.Helper()
	path := filepath.Join("testdata", "broken", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	var plate Plate
	if err := yaml.Unmarshal(data, &plate); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", path, err)
	}
	if plate.Ingredients == nil {
		plate.Ingredients = map[string]Ingredient{}
	}
	plate.Path = filepath.ToSlash(path)
	return plate
}

func lintValidPlate() Plate {
	return Plate{
		Name:        "nmap_full_tcp",
		Category:    "scanning",
		Description: "Full TCP port scan with output files",
		Tags:        []string{"nmap", "scanning", "tcp"},
		Ingredients: map[string]Ingredient{
			"target": {Description: "Target IP address or hostname", Required: true},
			"rate":   {Description: "Minimum packet rate", Required: false, Default: "5000"},
		},
		Template: "nmap --min-rate {{rate}} {{target}}",
	}
}

func lintPlateWithName(name, category string) Plate {
	plate := lintValidPlate()
	plate.Name = name
	plate.Category = category
	return plate
}

func hasIssue(result LintResult, severity LintSeverity, text string) bool {
	for _, issue := range result.Issues {
		if issue.Severity == severity && strings.Contains(issue.Message, text) {
			return true
		}
	}
	return false
}
