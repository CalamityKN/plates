package plates

import (
	"bytes"
	"fmt"
	"os"
	"text/template"
)

type Renderer interface {
	Render(plate Plate, values map[string]string) (string, error)
}

type TemplateRenderer struct{}

type RenderContext struct {
	Secrets map[string]string
	Env     func(string) (string, bool)
}

type RenderOutput struct {
	Text         string
	SecretValues []string
}

func NewTemplateRenderer() *TemplateRenderer {
	return &TemplateRenderer{}
}

func (r *TemplateRenderer) Render(plate Plate, values map[string]string) (string, error) {
	output, err := r.RenderWithContext(plate, values, RenderContext{})
	if err != nil {
		return "", err
	}
	return output.Text, nil
}

func (r *TemplateRenderer) RenderWithContext(plate Plate, values map[string]string, ctx RenderContext) (RenderOutput, error) {
	funcs := template.FuncMap{}
	var usedSecrets []string
	if ctx.Env == nil {
		ctx.Env = os.LookupEnv
	}
	for key := range plate.Ingredients {
		if !templateIdentifierPattern.MatchString(key) {
			continue
		}
		v := values[key]
		funcs[key] = func() string {
			return v
		}
	}
	for key, value := range values {
		if !templateIdentifierPattern.MatchString(key) {
			continue
		}
		v := value
		funcs[key] = func() string {
			return v
		}
	}
	funcs["secret"] = func(name string) (string, error) {
		value, ok := ctx.Secrets[name]
		if !ok {
			return "", fmt.Errorf("Missing secret: %s", name)
		}
		if value != "" {
			usedSecrets = append(usedSecrets, value)
		}
		return value, nil
	}
	funcs["env"] = func(name string) (string, error) {
		value, ok := ctx.Env(name)
		if !ok {
			return "", fmt.Errorf("Missing environment variable: %s", name)
		}
		return value, nil
	}
	tmpl, err := template.New(plate.Name).Funcs(funcs).Parse(plate.Template)
	if err != nil {
		return RenderOutput{}, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, values); err != nil {
		return RenderOutput{}, err
	}
	for name, ingredient := range plate.Ingredients {
		if ingredient.Secret && values[name] != "" {
			usedSecrets = append(usedSecrets, values[name])
		}
	}
	return RenderOutput{Text: buf.String(), SecretValues: usedSecrets}, nil
}
