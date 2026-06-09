package plates

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Plate struct {
	Name        string                `yaml:"name"`
	Category    string                `yaml:"category"`
	Description string                `yaml:"description"`
	Tags        []string              `yaml:"tags"`
	Ingredients map[string]Ingredient `yaml:"options"`
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

func (p *Plate) UnmarshalYAML(value *yaml.Node) error {
	type rawPlate struct {
		Name        string                `yaml:"name"`
		Category    string                `yaml:"category"`
		Description string                `yaml:"description"`
		Tags        []string              `yaml:"tags"`
		Options     map[string]Ingredient `yaml:"options"`
		Ingredients map[string]Ingredient `yaml:"ingredients"`
		Template    string                `yaml:"template"`
	}
	var raw rawPlate
	if err := value.Decode(&raw); err != nil {
		return err
	}
	p.Name = raw.Name
	p.Category = raw.Category
	p.Description = raw.Description
	p.Tags = raw.Tags
	p.Template = raw.Template
	p.Ingredients = raw.Options
	if p.Ingredients == nil {
		p.Ingredients = raw.Ingredients
	}
	return nil
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
			return errors.New("plate contains an option with an empty name")
		}
		if !templateIdentifierPattern.MatchString(name) {
			return fmt.Errorf("option %q must use letters, numbers, and underscores, starting with a letter or underscore", name)
		}
		if strings.TrimSpace(ingredient.Description) == "" {
			return fmt.Errorf("option %q is missing description", name)
		}
	}
	return nil
}

var templateIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
