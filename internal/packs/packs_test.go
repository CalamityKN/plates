package packs

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"plates/internal/plates"
)

func TestExportEntireRack(t *testing.T) {
	root, rack, exported, index := packFixture(t)
	path, manifest, err := Export(rack, exported, index, ExportOptions{Name: "core"})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if len(manifest.Plates) != 3 {
		t.Fatalf("manifest plates = %#v", manifest.Plates)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat %s error = %v", path, err)
	}
	_ = root
}

func TestExportByCategory(t *testing.T) {
	_, rack, exported, index := packFixture(t)
	_, manifest, err := Export(rack, exported, index, ExportOptions{Name: "scans", Category: "scanning"})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if len(manifest.Plates) != 2 {
		t.Fatalf("manifest plates = %#v", manifest.Plates)
	}
}

func TestExportByTag(t *testing.T) {
	_, rack, exported, index := packFixture(t)
	_, manifest, err := Export(rack, exported, index, ExportOptions{Name: "web", Tag: "http"})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if len(manifest.Plates) != 1 || manifest.Plates[0] != "files/http_server" {
		t.Fatalf("manifest plates = %#v", manifest.Plates)
	}
}

func TestExportSinglePlate(t *testing.T) {
	_, rack, exported, index := packFixture(t)
	_, manifest, err := Export(rack, exported, index, ExportOptions{Name: "one", Plate: "scanning/nmap_full_tcp"})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if len(manifest.Plates) != 1 || manifest.Plates[0] != "scanning/nmap_full_tcp" {
		t.Fatalf("manifest plates = %#v", manifest.Plates)
	}
}

func TestPackManifestCreationAndInspect(t *testing.T) {
	_, rack, exported, index := packFixture(t)
	path, manifest, err := Export(rack, exported, index, ExportOptions{Name: "core"})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	inspected, err := Inspect(path)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if inspected.Manifest.Name != manifest.Name || len(inspected.Plates) != 3 {
		t.Fatalf("inspected = %#v", inspected)
	}
}

func TestValidateValidPack(t *testing.T) {
	_, rack, exported, index := packFixture(t)
	path, _, err := Export(rack, exported, index, ExportOptions{Name: "core"})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if _, err := Validate(path); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestRejectMissingManifest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.zip")
	writeZip(t, path, map[string]string{"rack/scanning/test.yml": packPlateYAML("test", "scanning")})
	if _, err := Validate(path); err == nil {
		t.Fatal("Validate() error = nil")
	}
}

func TestRejectPathTraversalEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.zip")
	writeZip(t, path, map[string]string{"pack.yaml": "name: bad\nversion: 1.0.0\nplates: []\n", "rack/../evil.yml": "bad"})
	if _, err := Validate(path); err == nil {
		t.Fatal("Validate() error = nil")
	}
}

func TestImportPackWithoutOverwrite(t *testing.T) {
	_, rack, exported, index := packFixture(t)
	path, _, err := Export(rack, exported, index, ExportOptions{Name: "core", Plate: "files/http_server"})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	destRack := filepath.Join(t.TempDir(), "rack")
	result, err := Import(path, destRack, false)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.Imported != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestImportConflictBehavior(t *testing.T) {
	_, rack, exported, index := packFixture(t)
	path, _, err := Export(rack, exported, index, ExportOptions{Name: "core", Plate: "files/http_server"})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	destRack := filepath.Join(t.TempDir(), "rack")
	if _, err := Import(path, destRack, false); err != nil {
		t.Fatalf("first Import() error = %v", err)
	}
	_, err = Import(path, destRack, false)
	if !errors.Is(err, ErrConflicts) {
		t.Fatalf("second Import() error = %v", err)
	}
}

func TestImportWithForce(t *testing.T) {
	_, rack, exported, index := packFixture(t)
	path, _, err := Export(rack, exported, index, ExportOptions{Name: "core", Plate: "files/http_server"})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	destRack := filepath.Join(t.TempDir(), "rack")
	if _, err := Import(path, destRack, false); err != nil {
		t.Fatalf("first Import() error = %v", err)
	}
	result, err := Import(path, destRack, true)
	if err != nil {
		t.Fatalf("forced Import() error = %v", err)
	}
	if result.Overwritten != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestImportedPlatesLoadThroughRackLoader(t *testing.T) {
	root, rack, exported, index := packFixture(t)
	path, _, err := Export(rack, exported, index, ExportOptions{Name: "core", Plate: "files/http_server"})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	destRoot := t.TempDir()
	destRack := filepath.Join(destRoot, "data", "rack")
	if _, err := Import(path, destRack, false); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	plate, err := plates.NewRackRepository(destRoot, destRack).Load("files/http_server")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if plate.Name != "http_server" {
		t.Fatalf("plate = %#v", plate)
	}
	_ = root
}

func packFixture(t *testing.T) (string, string, string, *plates.RackIndex) {
	t.Helper()
	root := t.TempDir()
	rack := filepath.Join(root, "data", "rack")
	exported := filepath.Join(root, "data", "packs", "exported")
	packPlates := []plates.Plate{
		packPlate("nmap_full_tcp", "scanning", []string{"nmap", "scanning"}),
		packPlate("nmap_services", "scanning", []string{"nmap"}),
		packPlate("http_server", "files", []string{"http", "files"}),
	}
	return root, rack, exported, plates.NewRackIndex("data/rack", packPlates)
}

func packPlate(name, category string, tags []string) plates.Plate {
	return plates.Plate{
		Name:        name,
		Category:    category,
		Description: "Description for " + name,
		Tags:        tags,
		Ingredients: map[string]plates.Ingredient{
			"target": {Description: "Target host", Required: true},
		},
		Template: "echo {{target}}\n",
	}
}

func writeZip(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer file.Close()
	zw := zip.NewWriter(file)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("Create(%s) error = %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func packPlateYAML(name, category string) string {
	return `name: ` + name + `
category: ` + category + `
description: Test plate
ingredients:
  target:
    description: Target host
    required: true
template: |
  echo {{target}}
`
}
