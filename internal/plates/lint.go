package plates

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

type LintSeverity string

const (
	SeverityError LintSeverity = "ERROR"
	SeverityWarn  LintSeverity = "WARN"
)

type LintIssue struct {
	Severity LintSeverity
	Message  string
}

type LintResult struct {
	Plate  Plate
	Issues []LintIssue
}

type RackHealth struct {
	TotalPlates         int
	Passing             int
	Warning             int
	Failing             int
	DuplicateNames      int
	UnusedIngredients   int
	UndeclaredVariables int
}

func (r LintResult) Status() string {
	if r.HasErrors() {
		return "FAIL"
	}
	if r.HasWarnings() {
		return "WARN"
	}
	return "PASS"
}

func (r LintResult) HasErrors() bool {
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			return true
		}
	}
	return false
}

func (r LintResult) HasWarnings() bool {
	for _, issue := range r.Issues {
		if issue.Severity == SeverityWarn {
			return true
		}
	}
	return false
}

func LintPlate(plate Plate) LintResult {
	result := LintResult{Plate: plate}
	if strings.TrimSpace(plate.Name) == "" {
		result.addError("missing required metadata: name")
	} else if !plateNamePattern.MatchString(plate.Name) {
		result.addError("invalid plate name: " + plate.Name)
	}
	if strings.TrimSpace(plate.Category) == "" {
		result.addError("missing required metadata: category")
	} else if plate.Category != strings.ToLower(plate.Category) {
		result.addWarn("category should be lowercase: " + plate.Category)
	}
	if strings.TrimSpace(plate.Description) == "" {
		result.addError("missing required metadata: description")
	}
	if strings.TrimSpace(plate.Template) == "" {
		result.addError("missing required metadata: template")
	}

	refs := TemplateReferences(plate.Template)
	secretRefs := SecretReferences(plate.Template)
	if strings.TrimSpace(plate.Template) != "" {
		funcs := template.FuncMap{}
		for _, ref := range refs {
			name := ref
			funcs[name] = func() string { return "" }
		}
		funcs["secret"] = func(name string) string { return "" }
		funcs["env"] = func(name string) string { return "" }
		if _, err := template.New(plate.Name).Funcs(funcs).Parse(plate.Template); err != nil {
			result.addError("template parse failed: " + err.Error())
		}
	}

	for name, ingredient := range plate.Ingredients {
		if !templateIdentifierPattern.MatchString(name) {
			result.addError("invalid ingredient name: " + name)
		}
		if strings.TrimSpace(ingredient.Description) == "" {
			result.addWarn("ingredient has empty description: " + name)
		}
		if !contains(refs, name) && !contains(secretRefs, name) {
			result.addWarn("ingredient declared but unused: " + name)
		}
		if likelySensitive(name) && !ingredient.Secret {
			result.addWarn("likely sensitive ingredient not marked secret: " + name)
		}
	}

	for _, ref := range refs {
		if _, ok := plate.Ingredients[ref]; !ok {
			result.addError("template references undeclared ingredient: " + ref)
		}
	}
	for _, ref := range secretRefs {
		if _, ok := plate.Ingredients[ref]; !ok {
			result.addWarn("template references secret without matching ingredient: " + ref)
		}
	}

	for _, tag := range duplicateStrings(plate.Tags) {
		result.addWarn("duplicate tag: " + tag)
	}

	sort.SliceStable(result.Issues, func(i, j int) bool {
		if result.Issues[i].Severity != result.Issues[j].Severity {
			return result.Issues[i].Severity < result.Issues[j].Severity
		}
		return result.Issues[i].Message < result.Issues[j].Message
	})
	return result
}

func LintRack(index *RackIndex) []LintResult {
	results := make([]LintResult, 0, len(index.Plates))
	duplicates := duplicatePlateNames(index.Plates)
	for _, plate := range index.Plates {
		result := LintPlate(plate)
		if keys, ok := duplicates[plate.Name]; ok {
			result.addWarn(fmt.Sprintf("duplicate plate name exists in multiple categories: %s (%s)", plate.Name, strings.Join(keys, ", ")))
		}
		results = append(results, result)
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Plate.Key() < results[j].Plate.Key()
	})
	return results
}

func RackHealthFromResults(index *RackIndex, results []LintResult) RackHealth {
	health := RackHealth{
		TotalPlates:    len(results),
		DuplicateNames: len(duplicatePlateNames(index.Plates)),
	}
	for _, result := range results {
		if result.HasErrors() {
			health.Failing++
		} else if result.HasWarnings() {
			health.Warning++
		} else {
			health.Passing++
		}
		for _, issue := range result.Issues {
			switch {
			case strings.Contains(issue.Message, "ingredient declared but unused"):
				health.UnusedIngredients++
			case strings.Contains(issue.Message, "template references undeclared ingredient"):
				health.UndeclaredVariables++
			}
		}
	}
	return health
}

func TemplateReferences(text string) []string {
	seen := map[string]bool{}
	for _, match := range templateRefPattern.FindAllStringSubmatch(text, -1) {
		seen[match[1]] = true
	}
	refs := make([]string, 0, len(seen))
	for ref := range seen {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	return refs
}

func SecretReferences(text string) []string {
	seen := map[string]bool{}
	for _, match := range secretRefPattern.FindAllStringSubmatch(text, -1) {
		seen[match[1]] = true
	}
	refs := make([]string, 0, len(seen))
	for ref := range seen {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	return refs
}

func (r *LintResult) addError(message string) {
	r.Issues = append(r.Issues, LintIssue{Severity: SeverityError, Message: message})
}

func (r *LintResult) addWarn(message string) {
	r.Issues = append(r.Issues, LintIssue{Severity: SeverityWarn, Message: message})
}

func duplicatePlateNames(plates []Plate) map[string][]string {
	byName := map[string][]string{}
	for _, plate := range plates {
		byName[plate.Name] = append(byName[plate.Name], plate.Key())
	}
	duplicates := map[string][]string{}
	for name, keys := range byName {
		if name == "" || len(keys) < 2 {
			continue
		}
		sort.Strings(keys)
		duplicates[name] = keys
	}
	return duplicates
}

func duplicateStrings(values []string) []string {
	seen := map[string]int{}
	for _, value := range values {
		seen[value]++
	}
	var duplicates []string
	for value, count := range seen {
		if value != "" && count > 1 {
			duplicates = append(duplicates, value)
		}
	}
	sort.Strings(duplicates)
	return duplicates
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

var plateNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
var secretRefPattern = regexp.MustCompile(`{{\s*secret\s+"([^"]+)"\s*}}`)

func likelySensitive(name string) bool {
	normalized := strings.ToLower(name)
	for _, part := range []string{"password", "pass", "token", "api_key", "secret", "hash"} {
		if strings.Contains(normalized, part) {
			return true
		}
	}
	return false
}
