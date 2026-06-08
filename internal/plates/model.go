package plates

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type Plate struct {
	Name        string                `yaml:"name"`
	Category    string                `yaml:"category"`
	Description string                `yaml:"description"`
	Tags        []string              `yaml:"tags"`
	Ingredients map[string]Ingredient `yaml:"ingredients"`
	Template    string                `yaml:"template"`
	Path        string                `yaml:"-"`
}

type Ingredient struct {
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Default     string `yaml:"default,omitempty"`
	Secret      bool   `yaml:"secret,omitempty"`
}

func (p Plate) Key() string {
	if p.Category == "" || p.Name == "" {
		return p.Name
	}
	return p.Category + "/" + p.Name
}

func (p Plate) Validate() error {
	var missing []string
	if strings.TrimSpace(p.Name) == "" {
		missing = append(missing, "name")
	}
	if strings.TrimSpace(p.Category) == "" {
		missing = append(missing, "category")
	}
	if strings.TrimSpace(p.Description) == "" {
		missing = append(missing, "description")
	}
	if strings.TrimSpace(p.Template) == "" {
		missing = append(missing, "template")
	}
	if len(missing) > 0 {
		return fmt.Errorf("plate is missing required field(s): %s", strings.Join(missing, ", "))
	}
	for name, ingredient := range p.Ingredients {
		if strings.TrimSpace(name) == "" {
			return errors.New("plate contains an ingredient with an empty name")
		}
		if !templateIdentifierPattern.MatchString(name) {
			return fmt.Errorf("ingredient %q must use letters, numbers, and underscores, starting with a letter or underscore", name)
		}
		if strings.TrimSpace(ingredient.Description) == "" {
			return fmt.Errorf("ingredient %q is missing description", name)
		}
	}
	return nil
}

var templateIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
