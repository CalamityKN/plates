package guide

import (
	"strings"
	"testing"
)

func TestListIncludesAvailableTopics(t *testing.T) {
	got := List()
	for _, topic := range []string{"plates", "forge", "variables", "rack", "examples", "safety"} {
		if !strings.Contains(got, topic) {
			t.Fatalf("List() missing topic %q:\n%s", topic, got)
		}
	}
}

func TestShowPlatesIncludesTemplateContent(t *testing.T) {
	got, err := Show("plates")
	if err != nil {
		t.Fatalf("Show(plates) error = %v", err)
	}
	for _, want := range []string{"Plate YAML Guide", "http_server", "{{http_port}}"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Show(plates) missing %q:\n%s", want, got)
		}
	}
}

func TestShowForgeIncludesForgeModeContent(t *testing.T) {
	got, err := Show("forge")
	if err != nil {
		t.Fatalf("Show(forge) error = %v", err)
	}
	for _, want := range []string{"Forge Mode Guide", "save --force", "add_line"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Show(forge) missing %q:\n%s", want, got)
		}
	}
}

func TestShowUnknownTopicReturnsUsefulError(t *testing.T) {
	_, err := Show("unknown")
	if err == nil {
		t.Fatal("Show(unknown) error = nil")
	}
	if !strings.Contains(err.Error(), "unknown guide topic") || !strings.Contains(err.Error(), "guide") {
		t.Fatalf("Show(unknown) error = %v", err)
	}
}
