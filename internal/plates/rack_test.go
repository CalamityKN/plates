package plates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRackRepositoryLoadsValidPlate(t *testing.T) {
	root, rack := testRack(t)
	writePlate(t, rack, "scanning/nmap_full_tcp.yml", validPlateYAML("nmap_full_tcp", "scanning"))

	plate, err := NewRackRepository(root, rack).Load("scanning/nmap_full_tcp")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if plate.Name != "nmap_full_tcp" {
		t.Fatalf("Name = %q", plate.Name)
	}
	if plate.Path != "data/rack/scanning/nmap_full_tcp.yml" {
		t.Fatalf("Path = %q", plate.Path)
	}
}

func TestSecretDemoPlateLoadsAndLints(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	rack := filepath.Join(root, "data", "rack")
	plate, err := NewRackRepository(root, rack).Load("examples/secret_demo")
	if err != nil {
		t.Fatalf("Load(secret_demo) error = %v", err)
	}
	if !plate.Ingredients["password"].Secret {
		t.Fatal("password ingredient is not marked secret")
	}
	result := LintPlate(plate)
	if result.HasErrors() {
		t.Fatalf("secret_demo lint errors = %#v", result.Issues)
	}
}

func TestRackRepositoryRejectsInvalidPlate(t *testing.T) {
	root, rack := testRack(t)
	writePlate(t, rack, "bad.yml", "name: bad\ncategory: misc\n")

	_, err := NewRackRepository(root, rack).Load("bad")
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "description") || !strings.Contains(err.Error(), "template") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestRackRepositoryDiscoversPlatesRecursively(t *testing.T) {
	root, rack := testRack(t)
	writePlate(t, rack, "network/ping.yaml", validPlateYAML("ping", "network"))
	writePlate(t, rack, "web/http/server.yaml", validPlateYAML("server", "web"))
	writePlate(t, rack, "ignored.txt", "not a plate")

	got, err := NewRackRepository(root, rack).List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	want := []string{"network/ping", "web/server"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("List()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestResolveUsesCorrectPrecedence(t *testing.T) {
	plate := Plate{
		Ingredients: map[string]Ingredient{
			"target": {Required: true},
			"rate":   {Default: "5000"},
		},
	}
	got := Resolve(
		plate,
		map[string]string{"target": "pantry", "rate": "1000"},
		map[string]string{"target": "workspace"},
		map[string]string{"target": "session"},
	)
	if got.Values["target"] != "session" {
		t.Fatalf("target = %q", got.Values["target"])
	}
	if got.Values["rate"] != "1000" {
		t.Fatalf("rate = %q", got.Values["rate"])
	}
}

func TestRendererRendersSimpleTemplateVariables(t *testing.T) {
	plate := Plate{
		Name: "nmap",
		Ingredients: map[string]Ingredient{
			"target": {Required: true},
			"rate":   {Default: "5000"},
		},
		Template: "nmap --min-rate {{rate}} {{target}}",
	}
	rendered, err := NewTemplateRenderer().Render(plate, map[string]string{
		"target": "10.129.202.242",
		"rate":   "5000",
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if rendered != "nmap --min-rate 5000 10.129.202.242" {
		t.Fatalf("Render() = %q", rendered)
	}
}

func TestRendererRendersSecretWhenPresent(t *testing.T) {
	plate := Plate{Name: "secret", Template: `tool -p {{secret "password"}}`}
	rendered, err := NewTemplateRenderer().RenderWithContext(plate, nil, RenderContext{Secrets: map[string]string{"password": "SuperSecret123"}})
	if err != nil {
		t.Fatalf("RenderWithContext() error = %v", err)
	}
	if rendered.Text != "tool -p SuperSecret123" {
		t.Fatalf("Text = %q", rendered.Text)
	}
	if len(rendered.SecretValues) != 1 || rendered.SecretValues[0] != "SuperSecret123" {
		t.Fatalf("SecretValues = %#v", rendered.SecretValues)
	}
}

func TestRendererMissingSecretFails(t *testing.T) {
	plate := Plate{Name: "secret", Template: `tool -p {{secret "password"}}`}
	_, err := NewTemplateRenderer().RenderWithContext(plate, nil, RenderContext{Secrets: map[string]string{}})
	if err == nil {
		t.Fatal("RenderWithContext() error = nil")
	}
}

func TestRendererRendersEnvironmentVariable(t *testing.T) {
	plate := Plate{Name: "env", Template: `home {{env "HOME"}}`}
	rendered, err := NewTemplateRenderer().RenderWithContext(plate, nil, RenderContext{
		Env: func(name string) (string, bool) {
			if name == "HOME" {
				return "/tmp/home", true
			}
			return "", false
		},
	})
	if err != nil {
		t.Fatalf("RenderWithContext() error = %v", err)
	}
	if rendered.Text != "home /tmp/home" {
		t.Fatalf("Text = %q", rendered.Text)
	}
}

func TestRendererMissingEnvironmentVariableFails(t *testing.T) {
	plate := Plate{Name: "env", Template: `token {{env "PLATES_TOKEN"}}`}
	_, err := NewTemplateRenderer().RenderWithContext(plate, nil, RenderContext{Env: func(string) (string, bool) { return "", false }})
	if err == nil {
		t.Fatal("RenderWithContext() error = nil")
	}
}

func TestResolveReportsMissingRequiredIngredients(t *testing.T) {
	plate := Plate{
		Ingredients: map[string]Ingredient{
			"target": {Required: true},
			"rate":   {Default: "5000"},
		},
	}
	got := Resolve(plate, nil, nil, nil)
	if len(got.Missing) != 1 || got.Missing[0] != "target" {
		t.Fatalf("Missing = %#v", got.Missing)
	}
}

func TestRackIndexCategoryCounts(t *testing.T) {
	index := NewRackIndex("data/rack", sampleIndexPlates())
	counts := index.Categories()
	if counts["files"] != 1 || counts["scanning"] != 2 {
		t.Fatalf("Categories() = %#v", counts)
	}
}

func TestRackIndexTagCounts(t *testing.T) {
	index := NewRackIndex("data/rack", sampleIndexPlates())
	counts := index.Tags()
	if counts["nmap"] != 2 || counts["files"] != 1 {
		t.Fatalf("Tags() = %#v", counts)
	}
}

func TestRackIndexSearchIsCaseInsensitive(t *testing.T) {
	index := NewRackIndex("data/rack", sampleIndexPlates())
	got := index.Search("NMAP")
	if len(got) != 2 {
		t.Fatalf("Search(NMAP) returned %d results", len(got))
	}
}

func TestRackIndexSearchByTag(t *testing.T) {
	index := NewRackIndex("data/rack", sampleIndexPlates())
	got := index.Search("http")
	if len(got) != 1 || got[0].Key() != "files/http_server" {
		t.Fatalf("Search(http) = %#v", got)
	}
}

func TestRackIndexSearchByIngredientName(t *testing.T) {
	index := NewRackIndex("data/rack", sampleIndexPlates())
	got := index.Search("ports")
	if len(got) != 1 || got[0].Key() != "scanning/nmap_services" {
		t.Fatalf("Search(ports) = %#v", got)
	}
}

func TestRackIndexCategoryFiltering(t *testing.T) {
	index := NewRackIndex("data/rack", sampleIndexPlates())
	got := index.InCategory("scanning")
	if len(got) != 2 {
		t.Fatalf("InCategory(scanning) returned %d results", len(got))
	}
	for _, plate := range got {
		if plate.Category != "scanning" {
			t.Fatalf("InCategory(scanning) included %s", plate.Key())
		}
	}
}

func TestRackIndexStableSorting(t *testing.T) {
	index := NewRackIndex("data/rack", []Plate{
		samplePlate("zeta", "scanning", []string{"last"}, nil),
		samplePlate("http_server", "files", []string{"files"}, nil),
		samplePlate("alpha", "scanning", []string{"first"}, nil),
	})
	got := []string{index.Plates[0].Key(), index.Plates[1].Key(), index.Plates[2].Key()}
	want := []string{"files/http_server", "scanning/alpha", "scanning/zeta"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Plates[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func testRack(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	rack := filepath.Join(root, "data", "rack")
	if err := os.MkdirAll(rack, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	return root, rack
}

func writePlate(t *testing.T, rack, name, content string) {
	t.Helper()
	path := filepath.Join(rack, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func validPlateYAML(name, category string) string {
	return `name: ` + name + `
category: ` + category + `
description: A valid plate
ingredients:
  target:
    description: Target host
    required: true
template: |
  ping {{target}}
`
}

func sampleIndexPlates() []Plate {
	return []Plate{
		samplePlate("nmap_full_tcp", "scanning", []string{"nmap", "scanning", "tcp"}, map[string]Ingredient{
			"target": {Description: "Target IP address or hostname", Required: true},
		}),
		samplePlate("nmap_services", "scanning", []string{"nmap", "services"}, map[string]Ingredient{
			"target": {Description: "Target IP address or hostname", Required: true},
			"ports":  {Description: "Comma-separated list of ports", Required: true},
		}),
		samplePlate("http_server", "files", []string{"python", "http", "files"}, map[string]Ingredient{
			"workdir":   {Description: "Directory to serve files from", Required: true},
			"http_port": {Description: "Local HTTP server port", Default: "8000"},
		}),
	}
}

func samplePlate(name, category string, tags []string, ingredients map[string]Ingredient) Plate {
	if ingredients == nil {
		ingredients = map[string]Ingredient{}
	}
	return Plate{
		Name:        name,
		Category:    category,
		Description: "Description for " + name,
		Tags:        tags,
		Ingredients: ingredients,
		Template:    "echo {{target}}",
	}
}
