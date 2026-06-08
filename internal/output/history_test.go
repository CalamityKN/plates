package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestRenderHistoryCreation(t *testing.T) {
	store := testStore(t)
	record, err := store.Add(sampleRecord("nmap_full_tcp", "scanning", "nmap target\n"))
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if record.ID != "1" {
		t.Fatalf("ID = %q", record.ID)
	}
	records, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("len(records) = %d", len(records))
	}
}

func TestHistoryOrdering(t *testing.T) {
	store := testStore(t)
	old := sampleRecord("old", "scanning", "old")
	old.Timestamp = time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	newer := sampleRecord("new", "web", "new")
	newer.Timestamp = time.Date(2026, 6, 8, 11, 0, 0, 0, time.UTC)
	if _, err := store.Add(old); err != nil {
		t.Fatalf("Add(old) error = %v", err)
	}
	if _, err := store.Add(newer); err != nil {
		t.Fatalf("Add(newer) error = %v", err)
	}
	records, err := store.Recent()
	if err != nil {
		t.Fatalf("Recent() error = %v", err)
	}
	if records[0].PlateName != "new" {
		t.Fatalf("records[0].PlateName = %q", records[0].PlateName)
	}
}

func TestHistoryDeletion(t *testing.T) {
	store := testStore(t)
	record, err := store.Add(sampleRecord("nmap_full_tcp", "scanning", "nmap target\n"))
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := store.Delete(record.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	records, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("len(records) = %d", len(records))
	}
}

func TestHistoryClearing(t *testing.T) {
	store := testStore(t)
	if _, err := store.Add(sampleRecord("nmap_full_tcp", "scanning", "nmap target\n")); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	records, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("len(records) = %d", len(records))
	}
}

func TestExportJSON(t *testing.T) {
	store := testStore(t)
	record, _ := store.Add(sampleRecord("nmap_full_tcp", "scanning", "nmap target\n"))
	path, err := store.Export(record, "json")
	if err != nil {
		t.Fatalf("Export(json) error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var decoded RenderRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded.PlateName != "nmap_full_tcp" {
		t.Fatalf("PlateName = %q", decoded.PlateName)
	}
}

func TestExportYAML(t *testing.T) {
	store := testStore(t)
	record, _ := store.Add(sampleRecord("nmap_full_tcp", "scanning", "nmap target\n"))
	path, err := store.Export(record, "yaml")
	if err != nil {
		t.Fatalf("Export(yaml) error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var decoded RenderRecord
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	if decoded.Category != "scanning" {
		t.Fatalf("Category = %q", decoded.Category)
	}
}

func TestExportMarkdown(t *testing.T) {
	store := testStore(t)
	record, _ := store.Add(sampleRecord("nmap_full_tcp", "scanning", "nmap target\n"))
	path, err := store.Export(record, "markdown")
	if err != nil {
		t.Fatalf("Export(markdown) error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "# Rendered Plate") || !strings.Contains(got, "```bash") {
		t.Fatalf("markdown =\n%s", got)
	}
}

func TestSaveOutput(t *testing.T) {
	store := testStore(t)
	record, _ := store.Add(sampleRecord("nmap_full_tcp", "scanning", "nmap target\n"))
	path, err := store.SaveOutput(record, "my_scan.txt")
	if err != nil {
		t.Fatalf("SaveOutput() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "nmap target\n" {
		t.Fatalf("saved output = %q", string(data))
	}
	if filepath.Base(path) != "my_scan.txt" {
		t.Fatalf("path = %q", path)
	}
}

func TestStatisticsGeneration(t *testing.T) {
	store := testStore(t)
	if _, err := store.Add(sampleRecord("nmap_full_tcp", "scanning", "one\n")); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if _, err := store.Add(sampleRecord("nmap_full_tcp", "scanning", "two\n")); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if _, err := store.Add(sampleRecord("http_server", "files", "three\n")); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.TotalRenders != 3 {
		t.Fatalf("TotalRenders = %d", stats.TotalRenders)
	}
	if stats.TopPlates[0].Name != "scanning/nmap_full_tcp" || stats.TopPlates[0].Count != 2 {
		t.Fatalf("TopPlates = %#v", stats.TopPlates)
	}
}

func testStore(t *testing.T) *Store {
	t.Helper()
	store := NewStore(t.TempDir())
	store.now = func() time.Time {
		return time.Date(2026, 6, 8, 15, 35, 22, 0, time.UTC)
	}
	return store
}

func sampleRecord(name, category, text string) RenderRecord {
	return RenderRecord{
		Workspace: "devhub",
		PlateName: name,
		Category:  category,
		Variables: map[string]string{"target": "10.129.202.242"},
		Output:    text,
	}
}
