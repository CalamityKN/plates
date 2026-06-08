package plates

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type Draft struct {
	Name        string
	Category    string
	Description string
	Tags        []string
	Ingredients map[string]Ingredient
	Lines       []string
}

type DraftValidation struct {
	Warnings []string
}

func NewDraft() *Draft {
	return &Draft{Ingredients: map[string]Ingredient{}}
}

func (d *Draft) ToPlate() Plate {
	ingredients := map[string]Ingredient{}
	for name, ingredient := range d.Ingredients {
		ingredients[name] = ingredient
	}
	return Plate{
		Name:        d.Name,
		Category:    d.Category,
		Description: d.Description,
		Tags:        append([]string(nil), d.Tags...),
		Ingredients: ingredients,
		Template:    strings.Join(d.Lines, "\n") + "\n",
	}
}

func (d *Draft) AddLine(line string) {
	d.Lines = append(d.Lines, line)
}

func (d *Draft) InsertLine(number int, line string) error {
	if number < 1 || number > len(d.Lines)+1 {
		return fmt.Errorf("line number must be between 1 and %d", len(d.Lines)+1)
	}
	index := number - 1
	d.Lines = append(d.Lines, "")
	copy(d.Lines[index+1:], d.Lines[index:])
	d.Lines[index] = line
	return nil
}

func (d *Draft) DeleteLine(number int) error {
	if number < 1 || number > len(d.Lines) {
		return fmt.Errorf("line number must be between 1 and %d", len(d.Lines))
	}
	index := number - 1
	d.Lines = append(d.Lines[:index], d.Lines[index+1:]...)
	return nil
}

func (d *Draft) ClearLines() {
	d.Lines = nil
}

func (d *Draft) AddRequiredVar(name, description string) error {
	if err := validateDraftVarName(name); err != nil {
		return err
	}
	d.ensureIngredients()
	d.Ingredients[name] = Ingredient{Description: description, Required: true}
	return nil
}

func (d *Draft) AddSecretVar(name, description string) error {
	if err := validateDraftVarName(name); err != nil {
		return err
	}
	d.ensureIngredients()
	d.Ingredients[name] = Ingredient{Description: description, Required: true, Secret: true}
	return nil
}

func (d *Draft) AddOptionalVar(name, defaultValue, description string) error {
	if err := validateDraftVarName(name); err != nil {
		return err
	}
	d.ensureIngredients()
	d.Ingredients[name] = Ingredient{Description: description, Required: false, Default: defaultValue}
	return nil
}

func (d *Draft) SetVarRequired(name string, required bool) error {
	ingredient, ok := d.Ingredients[name]
	if !ok {
		return fmt.Errorf("variable not found: %s", name)
	}
	ingredient.Required = required
	d.Ingredients[name] = ingredient
	return nil
}

func (d *Draft) SetVarDefault(name, value string) error {
	ingredient, ok := d.Ingredients[name]
	if !ok {
		return fmt.Errorf("variable not found: %s", name)
	}
	ingredient.Default = value
	d.Ingredients[name] = ingredient
	return nil
}

func (d *Draft) RemoveVar(name string) {
	delete(d.Ingredients, name)
}

func (d *Draft) AddTag(tag string) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return
	}
	for _, existing := range d.Tags {
		if existing == tag {
			return
		}
	}
	d.Tags = append(d.Tags, tag)
	sort.Strings(d.Tags)
}

func (d *Draft) RemoveTag(tag string) {
	for i, existing := range d.Tags {
		if existing == tag {
			d.Tags = append(d.Tags[:i], d.Tags[i+1:]...)
			return
		}
	}
}

func (d *Draft) ValidateDraft() (DraftValidation, error) {
	var missing []string
	if strings.TrimSpace(d.Name) == "" {
		missing = append(missing, "name")
	}
	if strings.TrimSpace(d.Category) == "" {
		missing = append(missing, "category")
	}
	if strings.TrimSpace(d.Description) == "" {
		missing = append(missing, "description")
	}
	if len(d.Lines) == 0 {
		missing = append(missing, "template line")
	}
	if len(missing) > 0 {
		return DraftValidation{}, fmt.Errorf("draft is missing required field(s): %s", strings.Join(missing, ", "))
	}
	if err := validateDraftName(d.Name); err != nil {
		return DraftValidation{}, err
	}
	if err := validateDraftCategory(d.Category); err != nil {
		return DraftValidation{}, err
	}
	for name := range d.Ingredients {
		if err := validateDraftVarName(name); err != nil {
			return DraftValidation{}, err
		}
	}

	references := d.TemplateReferences()
	funcs := template.FuncMap{}
	for _, ref := range references {
		name := ref
		funcs[name] = func() string { return "" }
	}
	if _, err := template.New(d.Name).Funcs(funcs).Parse(d.ToPlate().Template); err != nil {
		return DraftValidation{}, err
	}

	var undeclared []string
	for _, ref := range references {
		if _, ok := d.Ingredients[ref]; !ok {
			undeclared = append(undeclared, ref)
		}
	}
	validation := DraftValidation{}
	if len(undeclared) > 0 {
		validation.Warnings = append(validation.Warnings, "template references undeclared ingredients: "+strings.Join(undeclared, ", "))
	}
	return validation, nil
}

func (d *Draft) Save(rackDir string, force bool) (string, error) {
	if _, err := d.ValidateDraft(); err != nil {
		return "", err
	}
	path := filepath.Join(rackDir, filepath.FromSlash(d.Category), d.Name+".yml")
	if _, err := os.Stat(path); err == nil && !force {
		return filepath.ToSlash(path), ErrPlateExists
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	data, err := yaml.Marshal(d.ToPlate())
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(path), os.WriteFile(path, data, 0o644)
}

func (d *Draft) TemplateReferences() []string {
	seen := map[string]bool{}
	for _, match := range templateRefPattern.FindAllStringSubmatch(strings.Join(d.Lines, "\n"), -1) {
		seen[match[1]] = true
	}
	refs := make([]string, 0, len(seen))
	for ref := range seen {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	return refs
}

func (d *Draft) ensureIngredients() {
	if d.Ingredients == nil {
		d.Ingredients = map[string]Ingredient{}
	}
}

var (
	ErrPlateExists       = errors.New("plate already exists")
	draftNamePattern     = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	draftCategoryPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+(/[A-Za-z0-9_-]+)*$`)
	templateRefPattern   = regexp.MustCompile(`{{\s*([A-Za-z_][A-Za-z0-9_]*)\s*}}`)
)

func validateDraftName(name string) error {
	if !draftNamePattern.MatchString(name) {
		return fmt.Errorf("invalid draft name %q; use letters, numbers, underscores, or hyphens", name)
	}
	return nil
}

func validateDraftCategory(category string) error {
	if !draftCategoryPattern.MatchString(category) {
		return fmt.Errorf("invalid category %q; use letters, numbers, underscores, hyphens, and slash-separated subcategories", category)
	}
	return nil
}

func validateDraftVarName(name string) error {
	if !templateIdentifierPattern.MatchString(name) {
		return fmt.Errorf("invalid variable %q; use letters, numbers, and underscores, starting with a letter or underscore", name)
	}
	return nil
}
