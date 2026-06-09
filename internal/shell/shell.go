package shell

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"plates/internal/config"
	"plates/internal/guide"
	"plates/internal/output"
	"plates/internal/packs"
	"plates/internal/plates"
	"plates/internal/secrets"
	"plates/internal/workspace"

	"github.com/chzyer/readline"
)

type Options struct {
	In  io.Reader
	Out io.Writer
}

type Shell struct {
	paths          config.Paths
	store          config.VariableStore
	configStore    *config.AppConfigStore
	appConfig      config.AppConfig
	discoverer     plates.Discoverer
	loader         plates.Loader
	browser        plates.Browser
	renderer       plates.Renderer
	outputStore    *output.Store
	clipboard      output.Clipboard
	secretStore    *secrets.Store
	session        *workspace.Session
	sessionVars    map[string]string
	plate          *plates.Plate
	forgeMode      bool
	draft          *plates.Draft
	clearSecrets   bool
	lastLiveOutput string
	out            io.Writer
}

func New(opts Options) (*Shell, error) {
	paths, err := config.DefaultPaths()
	if err != nil {
		return nil, err
	}
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	rack := plates.NewRackRepository(paths.RootDir, paths.RackDir)
	return &Shell{
		paths:       paths,
		store:       config.NewYAMLStore(paths),
		configStore: config.NewAppConfigStore(paths.ConfigFile),
		appConfig:   config.DefaultAppConfig(),
		discoverer:  rack,
		loader:      rack,
		browser:     rack,
		renderer:    plates.NewTemplateRenderer(),
		outputStore: output.NewStore(paths.DataDir),
		clipboard:   output.SystemClipboard{},
		secretStore: secrets.NewStore(paths.SecretsFile),
		session:     workspace.NewSession(),
		sessionVars: map[string]string{},
		out:         out,
	}, nil
}

func (s *Shell) Run() error {
	if s.configStore != nil {
		cfg, err := s.configStore.Load()
		if err != nil {
			return err
		}
		s.appConfig = cfg
	}
	s.printStartup()
	rl, err := readline.NewEx(&readline.Config{
		Prompt:       "PLATES > ",
		HistoryFile:  ".plates_history",
		AutoComplete: s.completer(),
	})
	if err != nil {
		return err
	}
	defer rl.Close()

	for {
		rl.SetPrompt(s.prompt())
		line, err := rl.Readline()
		if errors.Is(err, readline.ErrInterrupt) {
			continue
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := s.Execute(line); errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			fmt.Fprintln(s.out, "error:", err)
		}
	}
}

func (s *Shell) Execute(line string) error {
	fields, err := splitCommand(line)
	if err != nil {
		return err
	}
	if len(fields) == 0 {
		return nil
	}
	if s.clearSecrets {
		return s.confirmSecretClear(fields)
	}
	if s.forgeMode {
		return s.executeForge(line, fields)
	}

	switch fields[0] {
	case "exit", "quit":
		return io.EOF
	case "help":
		s.printHelp()
	case "guide":
		return s.printGuide(fields)
	case "init":
		return s.init()
	case "workspace":
		return s.useWorkspace(fields)
	case "set":
		return s.setWorkspace(fields)
	case "setg":
		return s.setGlobal(fields)
	case "use":
		return s.usePlate(fields)
	case "forge":
		return s.startForge(fields)
	case "show":
		return s.show(fields)
	case "list":
		return s.list(fields)
	case "search":
		return s.search(fields)
	case "info", "ll":
		if len(fields) != 1 {
			return fmt.Errorf("usage: %s", fields[0])
		}
		return s.showPlate()
	case "copy":
		return s.copyOutput(fields)
	case "save":
		return s.save(fields)
	case "history":
		if len(fields) != 1 {
			return errors.New("usage: history")
		}
		return s.outputHistory()
	case "export":
		return s.export(fields)
	case "config":
		return s.config(fields)
	case "pack":
		return s.pack(fields)
	case "secret":
		return s.secret(fields)
	case "render", "r", "run":
		return s.renderCurrent()
	case "clear":
		return s.clear(fields)
	case "shell":
		return s.shell(fields)
	default:
		return fmt.Errorf("unknown command %q; run 'help' for available commands", fields[0])
	}
	return nil
}

func (s *Shell) init() error {
	if err := s.store.EnsureBase(); err != nil {
		return err
	}
	if s.outputStore != nil {
		if err := s.outputStore.Ensure(); err != nil {
			return err
		}
	}
	if s.configStore != nil {
		cfg, err := s.configStore.Load()
		if err != nil {
			return err
		}
		s.appConfig = cfg
	}
	if s.secretStore != nil {
		if err := s.secretStore.Ensure(); err != nil {
			return err
		}
	}
	fmt.Fprintln(s.out, "Initialized data directories.")
	return nil
}

func (s *Shell) useWorkspace(fields []string) error {
	if len(fields) != 2 {
		return errors.New("usage: workspace <name>")
	}
	if err := s.store.EnsureWorkspace(fields[1]); err != nil {
		return err
	}
	s.session.Use(fields[1])
	s.sessionVars = map[string]string{}
	fmt.Fprintf(s.out, "Workspace: %s\n", fields[1])
	return nil
}

func (s *Shell) setWorkspace(fields []string) error {
	if len(fields) < 3 {
		return errors.New("usage: set <key> <value>")
	}
	value := strings.Join(fields[2:], " ")
	if err := s.store.SetWorkspaceValue(s.session.Current(), fields[1], value); err != nil {
		return err
	}
	s.sessionVars[fields[1]] = value
	return nil
}

func (s *Shell) setGlobal(fields []string) error {
	if len(fields) < 3 {
		return errors.New("usage: setg <key> <value>")
	}
	return s.store.SetGlobal(fields[1], strings.Join(fields[2:], " "))
}

func (s *Shell) usePlate(fields []string) error {
	if len(fields) != 2 {
		return errors.New("usage: use <plate>")
	}
	if fields[1] == "forge" {
		return s.startForge([]string{"forge"})
	}
	plate, err := s.loader.Load(fields[1])
	if err != nil {
		return err
	}
	s.plate = &plate
	fmt.Fprintf(s.out, "Loaded plate: %s\n", plate.Key())
	return nil
}

func (s *Shell) startForge(fields []string) error {
	if len(fields) != 1 {
		return errors.New("usage: forge")
	}
	s.forgeMode = true
	s.draft = plates.NewDraft()
	s.plate = nil
	fmt.Fprintln(s.out, "Forge mode started.")
	return nil
}

func (s *Shell) show(fields []string) error {
	if len(fields) < 2 {
		return errors.New("usage: show pantry|workspace|options|rack|tags|category <name>")
	}
	switch fields[1] {
	case "pantry":
		if len(fields) != 2 {
			return errors.New("usage: show pantry")
		}
		values, err := s.store.Globals()
		if err != nil {
			return err
		}
		s.printValues(values, s.secretKeysForCurrentPlate())
	case "workspace":
		if len(fields) != 2 {
			return errors.New("usage: show workspace")
		}
		values, err := s.store.WorkspaceValues(s.session.Current())
		if err != nil {
			return err
		}
		s.printValues(values, s.secretKeysForCurrentPlate())
	case "options":
		if len(fields) != 2 {
			return errors.New("usage: show options")
		}
		return s.showOptions()
	case "rack":
		if len(fields) != 2 {
			return errors.New("usage: show rack")
		}
		return s.showRack()
	case "tags":
		if len(fields) != 2 {
			return errors.New("usage: show tags")
		}
		return s.showTags()
	case "category":
		if len(fields) != 3 {
			return errors.New("usage: show category <name>")
		}
		return s.showCategory(fields[2])
	default:
		return errors.New("usage: show pantry|workspace|options|rack|tags|category <name>")
	}
	return nil
}

func (s *Shell) list(fields []string) error {
	if len(fields) != 2 || fields[1] != "plates" {
		return errors.New("usage: list plates")
	}
	index, err := s.browser.Index()
	if err != nil {
		return err
	}
	if len(index.Plates) == 0 {
		fmt.Fprintln(s.out, "No plates found.")
		return nil
	}
	fmt.Fprintln(s.out, "Available Plates:")
	categories := plates.SortedMapKeys(index.Categories())
	for _, category := range categories {
		fmt.Fprintln(s.out)
		fmt.Fprintf(s.out, "%s/\n", category)
		writer := tabwriter.NewWriter(s.out, 0, 0, 2, ' ', 0)
		for _, plate := range index.InCategory(category) {
			fmt.Fprintf(writer, "  %s\t%s\n", plate.Key(), plate.Description)
		}
		writer.Flush()
	}
	return nil
}

func (s *Shell) search(fields []string) error {
	if len(fields) < 3 || fields[1] != "plates" {
		return errors.New("usage: search plates <query>")
	}
	query := strings.Join(fields[2:], " ")
	index, err := s.browser.Index()
	if err != nil {
		return err
	}
	results := index.Search(query)
	if len(results) == 0 {
		fmt.Fprintf(s.out, "No plates matched: %s\n", query)
		return nil
	}
	fmt.Fprintf(s.out, "Search results for: %s\n", query)
	for _, plate := range results {
		fmt.Fprintln(s.out)
		fmt.Fprintln(s.out, plate.Key())
		fmt.Fprintf(s.out, "  %s\n", plate.Description)
		if len(plate.Tags) > 0 {
			fmt.Fprintf(s.out, "  Tags: %s\n", strings.Join(plate.Tags, ", "))
		}
	}
	return nil
}

func (s *Shell) showPlate() error {
	if s.plate == nil {
		return errors.New("no plate loaded; run 'use <plate>' first")
	}
	fmt.Fprintf(s.out, "Name: %s\n", s.plate.Name)
	fmt.Fprintf(s.out, "Category: %s\n", s.plate.Category)
	fmt.Fprintf(s.out, "Description: %s\n", s.plate.Description)
	fmt.Fprintf(s.out, "Tags: %s\n", strings.Join(s.plate.Tags, ", "))
	fmt.Fprintf(s.out, "Path: %s\n", s.plate.Path)
	fmt.Fprintf(s.out, "Options: %d\n", len(s.plate.Ingredients))
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, "Template Preview:")
	for _, line := range previewLines(s.plate.Template, 5) {
		fmt.Fprintf(s.out, "  %s\n", line)
	}
	return nil
}

func (s *Shell) showRack() error {
	index, err := s.browser.Index()
	if err != nil {
		return err
	}
	fmt.Fprintln(s.out, "Rack Summary")
	fmt.Fprintln(s.out)
	fmt.Fprintf(s.out, "Root: %s\n", index.Root)
	fmt.Fprintf(s.out, "Total plates: %d\n", len(index.Plates))
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, "Categories:")
	s.printCounts(index.Categories())
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, "Tags:")
	s.printCounts(index.Tags())
	return nil
}

func (s *Shell) showTags() error {
	index, err := s.browser.Index()
	if err != nil {
		return err
	}
	fmt.Fprintln(s.out, "Tags:")
	fmt.Fprintln(s.out)
	s.printCounts(index.Tags())
	return nil
}

func (s *Shell) showCategory(category string) error {
	index, err := s.browser.Index()
	if err != nil {
		return err
	}
	items := index.InCategory(category)
	if len(items) == 0 {
		fmt.Fprintf(s.out, "No category found: %s\n", category)
		return nil
	}
	fmt.Fprintf(s.out, "Category: %s\n", category)
	for _, plate := range items {
		fmt.Fprintln(s.out)
		fmt.Fprintln(s.out, plate.Key())
		fmt.Fprintf(s.out, "  %s\n", plate.Description)
	}
	return nil
}

func (s *Shell) showOptions() error {
	if s.plate == nil {
		return errors.New("no plate loaded; run 'use <plate>' first")
	}
	resolution, err := s.resolveCurrentPlate()
	if err != nil {
		return err
	}

	required, optional := s.ingredientNames()
	fmt.Fprintln(s.out, "Required:")
	s.printIngredientGroup(required, resolution.Values)
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, "Optional:")
	s.printIngredientGroup(optional, resolution.Values)

	if len(resolution.Missing) > 0 {
		fmt.Fprintln(s.out)
		fmt.Fprintln(s.out, "Missing required options:")
		for _, name := range resolution.Missing {
			fmt.Fprintf(s.out, "  %s\n", name)
		}
	}
	return nil
}

func (s *Shell) renderCurrent() error {
	if s.plate == nil {
		return errors.New("no plate loaded; run 'use <plate>' first")
	}
	resolution, err := s.resolveCurrentPlate()
	if err != nil {
		return err
	}
	secretValues := map[string]string{}
	if s.secretStore != nil {
		secretValues, err = s.secretStore.All()
		if err != nil {
			return err
		}
	}
	missing := s.filterMissingSecrets(resolution.Missing, secretValues)
	if len(missing) > 0 {
		fmt.Fprintln(s.out, "Missing required options:")
		for _, name := range missing {
			fmt.Fprintf(s.out, "  %s\n", name)
		}
		return nil
	}
	renderer, ok := s.renderer.(*plates.TemplateRenderer)
	if !ok {
		rendered, err := s.renderer.Render(*s.plate, resolution.Values)
		if err != nil {
			return err
		}
		record, err := s.storeRender(rendered, resolution.Values, nil)
		if err != nil {
			return err
		}
		s.printRendered(rendered, record.ID)
		return nil
	}
	rendered, err := renderer.RenderWithContext(*s.plate, resolution.Values, plates.RenderContext{Secrets: secretValues})
	if err != nil {
		return err
	}
	record, err := s.storeRender(rendered.Text, resolution.Values, rendered.SecretValues)
	if err != nil {
		return err
	}
	s.lastLiveOutput = strings.TrimRight(rendered.Text, "\n") + "\n"
	s.printRendered(rendered.Text, record.ID)
	return nil
}

func (s *Shell) filterMissingSecrets(missing []string, secretValues map[string]string) []string {
	var filtered []string
	for _, name := range missing {
		ingredient := s.plate.Ingredients[name]
		if ingredient.Secret {
			if value, ok := secretValues[name]; ok && value != "" {
				continue
			}
		}
		filtered = append(filtered, name)
	}
	return filtered
}

func (s *Shell) printRendered(rendered, id string) {
	fmt.Fprintf(s.out, "--- Rendered Plate: %s ---\n", s.plate.Key())
	fmt.Fprintln(s.out, "Render-only output. PLATES did not execute this command.")
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, strings.TrimRight(rendered, "\n"))
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, "--- End ---")
	fmt.Fprintln(s.out)
	fmt.Fprintf(s.out, "Stored as Render #%s\n", id)
}

func (s *Shell) storeRender(rendered string, values map[string]string, secretValues []string) (output.RenderRecord, error) {
	if s.outputStore == nil {
		return output.RenderRecord{ID: "0"}, nil
	}
	copied := map[string]string{}
	for key, value := range values {
		if s.plate != nil && s.plate.Ingredients[key].Secret {
			value = secrets.Mask()
		}
		copied[key] = value
	}
	raw := strings.TrimRight(rendered, "\n") + "\n"
	redacted := output.Redact(raw, secretValues)
	record := output.RenderRecord{
		Workspace:       s.session.Current(),
		PlateName:       s.plate.Name,
		Category:        s.plate.Category,
		Variables:       copied,
		Output:          redacted,
		ContainsSecrets: len(secretValues) > 0,
	}
	if len(secretValues) > 0 && s.currentConfig().StoreSecretOutputs {
		record.RawOutput = raw
	}
	return s.outputStore.Add(record)
}

func (s *Shell) clear(fields []string) error {
	if len(fields) != 2 || fields[1] != "plate" {
		return errors.New("usage: clear plate")
	}
	s.plate = nil
	fmt.Fprintln(s.out, "Plate cleared.")
	return nil
}

func (s *Shell) shell(fields []string) error {
	if len(fields) != 2 || fields[1] != "clear" {
		return errors.New("usage: shell clear")
	}
	fmt.Fprint(s.out, "\033[H\033[2J")
	return nil
}

func (s *Shell) copyOutput(fields []string) error {
	if len(fields) != 1 {
		return errors.New("usage: copy")
	}
	record, err := s.latestOutput()
	if err != nil {
		return err
	}
	if s.clipboard == nil {
		return errors.New("clipboard is not configured")
	}
	text := record.Output
	if s.lastLiveOutput != "" {
		text = s.lastLiveOutput
	}
	if err := s.clipboard.Copy(text); err != nil {
		return err
	}
	fmt.Fprintln(s.out, "Copied rendered output to clipboard.")
	return nil
}

func (s *Shell) save(fields []string) error {
	if len(fields) < 2 || fields[1] != "output" || len(fields) > 3 {
		return errors.New("usage: save output [filename]")
	}
	record, err := s.latestOutput()
	if err != nil {
		return err
	}
	filename := ""
	if len(fields) == 3 {
		filename = fields[2]
	}
	path, err := s.outputStore.SaveOutput(record, filename)
	if err != nil {
		return err
	}
	fmt.Fprintln(s.out, "Saved:")
	fmt.Fprintln(s.out, s.displayPath(path))
	return nil
}

func (s *Shell) export(fields []string) error {
	if len(fields) != 2 {
		return errors.New("usage: export markdown/json/yaml")
	}
	switch fields[1] {
	case "markdown", "json", "yaml":
	default:
		return errors.New("usage: export markdown/json/yaml")
	}
	record, err := s.latestOutput()
	if err != nil {
		return err
	}
	path, err := s.outputStore.Export(record, fields[1])
	if err != nil {
		return err
	}
	fmt.Fprintln(s.out, "Exported:")
	fmt.Fprintln(s.out, s.displayPath(path))
	return nil
}

func (s *Shell) secret(fields []string) error {
	if len(fields) < 2 {
		return errors.New("usage: secret set|get|reveal|list|delete|clear")
	}
	if s.secretStore == nil {
		return errors.New("secret store is not configured")
	}
	switch fields[1] {
	case "set":
		if len(fields) < 4 {
			return errors.New("usage: secret set <key> <value>")
		}
		if err := s.secretStore.Set(fields[2], strings.Join(fields[3:], " ")); err != nil {
			return err
		}
		fmt.Fprintf(s.out, "Secret stored: %s\n", fields[2])
	case "get":
		if len(fields) != 3 {
			return errors.New("usage: secret get <key>")
		}
		if _, ok, err := s.secretStore.Get(fields[2]); err != nil {
			return err
		} else if !ok {
			return fmt.Errorf("secret not found: %s", fields[2])
		}
		fmt.Fprintf(s.out, "%s = %s\n", fields[2], secrets.Mask())
	case "reveal":
		if len(fields) != 3 {
			return errors.New("usage: secret reveal <key>")
		}
		value, ok, err := s.secretStore.Get(fields[2])
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("secret not found: %s", fields[2])
		}
		fmt.Fprintf(s.out, "%s = %s\n", fields[2], value)
	case "list":
		if len(fields) != 2 {
			return errors.New("usage: secret list")
		}
		keys, err := s.secretStore.Keys()
		if err != nil {
			return err
		}
		fmt.Fprintln(s.out, "Secrets")
		fmt.Fprintln(s.out)
		if len(keys) == 0 {
			fmt.Fprintln(s.out, "(empty)")
			return nil
		}
		writer := tabwriter.NewWriter(s.out, 0, 0, 2, ' ', 0)
		for _, key := range keys {
			fmt.Fprintf(writer, "%s\t%s\n", key, secrets.Mask())
		}
		writer.Flush()
	case "delete":
		if len(fields) != 3 {
			return errors.New("usage: secret delete <key>")
		}
		if err := s.secretStore.Delete(fields[2]); err != nil {
			return err
		}
		fmt.Fprintf(s.out, "Secret deleted: %s\n", fields[2])
	case "clear":
		if len(fields) == 3 && fields[2] == "--force" {
			if err := s.secretStore.Clear(); err != nil {
				return err
			}
			fmt.Fprintln(s.out, "Secrets cleared.")
			return nil
		}
		if len(fields) != 2 {
			return errors.New("usage: secret clear [--force]")
		}
		s.clearSecrets = true
		fmt.Fprintln(s.out, "Type YES to continue:")
	default:
		return errors.New("usage: secret set|get|reveal|list|delete|clear")
	}
	return nil
}

func (s *Shell) confirmSecretClear(fields []string) error {
	s.clearSecrets = false
	if len(fields) == 1 && fields[0] == "YES" {
		if err := s.secretStore.Clear(); err != nil {
			return err
		}
		fmt.Fprintln(s.out, "Secrets cleared.")
		return nil
	}
	fmt.Fprintln(s.out, "Secret clear canceled.")
	return nil
}

func (s *Shell) secretKeysForCurrentPlate() map[string]bool {
	keys := map[string]bool{}
	if s.plate == nil {
		return keys
	}
	for name, ingredient := range s.plate.Ingredients {
		if ingredient.Secret {
			keys[name] = true
		}
	}
	return keys
}

func (s *Shell) outputHistory() error {
	records, err := s.outputStore.Recent()
	if err != nil {
		return err
	}
	fmt.Fprintln(s.out, "Recent Output History")
	fmt.Fprintln(s.out)
	if len(records) == 0 {
		fmt.Fprintln(s.out, "(empty)")
		return nil
	}
	writer := tabwriter.NewWriter(s.out, 0, 0, 2, ' ', 0)
	for _, record := range records {
		fmt.Fprintf(writer, "%s\t%s/%s\t%s\n", record.ID, record.Category, record.PlateName, record.Timestamp.Format("2006-01-02 15:04"))
	}
	writer.Flush()
	return nil
}

func (s *Shell) outputShow(id string, includeMetadata bool, reveal bool) error {
	record, err := s.outputStore.Get(id)
	if err != nil {
		return err
	}
	text := record.Output
	if reveal {
		if record.RawOutput == "" {
			return errors.New("raw output is not stored for this render")
		}
		text = record.RawOutput
	}
	if !includeMetadata {
		fmt.Fprint(s.out, text)
		return nil
	}
	fmt.Fprintf(s.out, "Render #%s\n\n", record.ID)
	fmt.Fprintln(s.out, "Plate:")
	fmt.Fprintf(s.out, "  %s/%s\n\n", record.Category, record.PlateName)
	fmt.Fprintln(s.out, "Timestamp:")
	fmt.Fprintf(s.out, "  %s\n\n", record.Timestamp.Format("2006-01-02 15:04"))
	fmt.Fprintln(s.out, "Output:")
	fmt.Fprintln(s.out)
	if record.ContainsSecrets && !reveal {
		fmt.Fprintln(s.out, "Output contains secrets. Showing redacted output:")
		fmt.Fprintln(s.out)
	}
	fmt.Fprint(s.out, text)
	return nil
}

func (s *Shell) config(fields []string) error {
	if len(fields) < 2 {
		return errors.New("usage: config show|set <key> <value>")
	}
	if s.configStore == nil {
		return errors.New("config store is not configured")
	}
	switch fields[1] {
	case "show":
		if len(fields) != 2 {
			return errors.New("usage: config show")
		}
		cfg, err := s.configStore.Load()
		if err != nil {
			return err
		}
		s.appConfig = cfg
		s.printConfig(cfg)
	case "set":
		if len(fields) != 4 {
			return errors.New("usage: config set <key> <value>")
		}
		cfg, err := s.configStore.Set(fields[2], fields[3])
		if err != nil {
			return err
		}
		s.appConfig = cfg
		fmt.Fprintf(s.out, "%s = %s\n", fields[2], fields[3])
	default:
		return errors.New("usage: config show|set <key> <value>")
	}
	return nil
}

func (s *Shell) pack(fields []string) error {
	if len(fields) < 2 {
		return errors.New("usage: pack list|export|inspect|validate|import")
	}
	switch fields[1] {
	case "list":
		if len(fields) != 2 {
			return errors.New("usage: pack list")
		}
		return s.packList()
	case "export":
		return s.packExport(fields)
	case "inspect":
		if len(fields) != 3 {
			return errors.New("usage: pack inspect <path>")
		}
		return s.packInspect(fields[2])
	case "validate":
		if len(fields) != 3 {
			return errors.New("usage: pack validate <path>")
		}
		return s.packValidate(fields[2])
	case "import":
		return s.packImport(fields)
	default:
		return errors.New("usage: pack list|export|inspect|validate|import")
	}
}

func (s *Shell) packList() error {
	fmt.Fprintln(s.out, "Plate Packs")
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, "Exported:")
	s.printZipFiles(s.paths.ExportedPacksDir)
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, "Imported:")
	s.printZipFiles(s.paths.ImportedPacksDir)
	return nil
}

func (s *Shell) packExport(fields []string) error {
	if len(fields) < 3 {
		return errors.New("usage: pack export <name> [--category <category>|--tag <tag>|--plate <category/name>]")
	}
	opts := packs.ExportOptions{Name: fields[2], Description: "PLATES plate pack"}
	for i := 3; i < len(fields); i += 2 {
		if i+1 >= len(fields) {
			return errors.New("usage: pack export <name> [--category <category>|--tag <tag>|--plate <category/name>]")
		}
		switch fields[i] {
		case "--category":
			opts.Category = fields[i+1]
		case "--tag":
			opts.Tag = fields[i+1]
		case "--plate":
			opts.Plate = fields[i+1]
		default:
			return errors.New("usage: pack export <name> [--category <category>|--tag <tag>|--plate <category/name>]")
		}
	}
	filters := 0
	for _, value := range []string{opts.Category, opts.Tag, opts.Plate} {
		if value != "" {
			filters++
		}
	}
	if filters > 1 {
		return errors.New("use only one export filter")
	}
	index, err := s.browser.Index()
	if err != nil {
		return err
	}
	path, manifest, err := packs.Export(s.paths.RackDir, s.paths.ExportedPacksDir, index, opts)
	if err != nil {
		return err
	}
	fmt.Fprintf(s.out, "Exported pack: %s\n", s.displayPath(path))
	fmt.Fprintf(s.out, "Plates: %d\n", len(manifest.Plates))
	return nil
}

func (s *Shell) packInspect(path string) error {
	inspected, err := packs.Inspect(path)
	if err != nil {
		return err
	}
	s.printPackInspect(inspected)
	return nil
}

func (s *Shell) packValidate(path string) error {
	results, err := packs.Validate(path)
	if err != nil {
		fmt.Fprintln(s.out, "Pack validation: FAIL")
		fmt.Fprintln(s.out)
		fmt.Fprintf(s.out, "[ERROR] %s\n", err)
		return nil
	}
	fail, warn, pass := lintCounts(results)
	if fail > 0 {
		fmt.Fprintln(s.out, "Pack validation: FAIL")
	} else if warn > 0 {
		fmt.Fprintln(s.out, "Pack validation: WARN")
	} else {
		fmt.Fprintln(s.out, "Pack validation: PASS")
	}
	fmt.Fprintf(s.out, "PASS: %d\nWARN: %d\nFAIL: %d\n", pass, warn, fail)
	return nil
}

func (s *Shell) packImport(fields []string) error {
	if len(fields) != 3 && len(fields) != 4 {
		return errors.New("usage: pack import <path> [--force]")
	}
	force := false
	if len(fields) == 4 {
		if fields[3] != "--force" {
			return errors.New("usage: pack import <path> [--force]")
		}
		force = true
	}
	result, err := packs.Import(fields[2], s.paths.RackDir, force)
	if errors.Is(err, packs.ErrConflicts) {
		fmt.Fprintln(s.out, "Conflict:")
		for _, conflict := range result.Conflicts {
			fmt.Fprintf(s.out, "  %s already exists.\n", conflict)
		}
		fmt.Fprintln(s.out)
		fmt.Fprintln(s.out, "Use:")
		fmt.Fprintf(s.out, "  pack import %s --force\n", fields[2])
		return nil
	}
	if err != nil {
		return err
	}
	if err := s.rememberImportedPack(fields[2]); err != nil {
		return err
	}
	fmt.Fprintf(s.out, "Imported %d plates.\n", result.Imported)
	fmt.Fprintf(s.out, "Skipped %d.\n", result.Skipped)
	fmt.Fprintf(s.out, "Overwritten %d.\n", result.Overwritten)
	fmt.Fprintln(s.out, "Review imported plates before use.")
	return nil
}

func (s *Shell) rememberImportedPack(path string) error {
	if s.paths.ImportedPacksDir == "" {
		return nil
	}
	if err := os.MkdirAll(s.paths.ImportedPacksDir, 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dest := filepath.Join(s.paths.ImportedPacksDir, filepath.Base(path))
	return os.WriteFile(dest, data, 0o644)
}

func (s *Shell) printPackInspect(inspected packs.InspectResult) {
	fail, warn, pass := lintCounts(inspected.Results)
	fmt.Fprintf(s.out, "Pack: %s\n", inspected.Manifest.Name)
	fmt.Fprintf(s.out, "Description: %s\n", inspected.Manifest.Description)
	fmt.Fprintf(s.out, "Version: %s\n", inspected.Manifest.Version)
	fmt.Fprintf(s.out, "Created: %s\n", inspected.Manifest.Created.Format("2006-01-02 15:04 MST"))
	fmt.Fprintf(s.out, "Plates: %d\n", len(inspected.Plates))
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, "Contents:")
	for _, plate := range inspected.Plates {
		fmt.Fprintf(s.out, "  %s\n", plate.Key())
	}
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, "Validation:")
	fmt.Fprintf(s.out, "  PASS: %d\n", pass)
	fmt.Fprintf(s.out, "  WARN: %d\n", warn)
	fmt.Fprintf(s.out, "  FAIL: %d\n", fail)
}

func (s *Shell) printZipFiles(dir string) {
	files, err := os.ReadDir(dir)
	if err != nil || len(files) == 0 {
		fmt.Fprintln(s.out, "  (none)")
		return
	}
	found := false
	for _, file := range files {
		if file.IsDir() || strings.ToLower(filepath.Ext(file.Name())) != ".zip" {
			continue
		}
		found = true
		fmt.Fprintf(s.out, "  %s\n", file.Name())
	}
	if !found {
		fmt.Fprintln(s.out, "  (none)")
	}
}

func lintCounts(results []plates.LintResult) (fail, warn, pass int) {
	for _, result := range results {
		switch result.Status() {
		case "FAIL":
			fail++
		case "WARN":
			warn++
		default:
			pass++
		}
	}
	return fail, warn, pass
}

func (s *Shell) printConfig(cfg config.AppConfig) {
	fmt.Fprintf(s.out, "banner: %t\n", cfg.Banner)
	fmt.Fprintf(s.out, "theme: %s\n", cfg.Theme)
	fmt.Fprintf(s.out, "prompt_style: %s\n", cfg.PromptStyle)
	fmt.Fprintf(s.out, "tips: %t\n", cfg.Tips)
	fmt.Fprintf(s.out, "store_secret_outputs: %t\n", cfg.StoreSecretOutputs)
}

func (s *Shell) printStartup() {
	cfg := s.currentConfig()
	if cfg.Banner {
		fmt.Fprint(s.out, banner)
		fmt.Fprintln(s.out, "Type 'help' for commands or 'guide' for built-in guides.")
	}
}

func (s *Shell) latestOutput() (output.RenderRecord, error) {
	if s.outputStore == nil {
		return output.RenderRecord{}, errors.New("history is not configured")
	}
	return s.outputStore.Latest()
}

func (s *Shell) printOutputCounts(counts []output.Count) {
	if len(counts) == 0 {
		fmt.Fprintln(s.out, "  (none)")
		return
	}
	writer := tabwriter.NewWriter(s.out, 0, 0, 2, ' ', 0)
	for _, count := range counts {
		fmt.Fprintf(writer, "  %s\t%d\n", count.Name, count.Count)
	}
	writer.Flush()
}

func (s *Shell) printValues(values map[string]string, secretKeys map[string]bool) {
	if len(values) == 0 {
		fmt.Fprintln(s.out, "(empty)")
		return
	}
	writer := tabwriter.NewWriter(s.out, 0, 0, 2, ' ', 0)
	for _, key := range config.SortedKeys(values) {
		value := values[key]
		if secretKeys[key] {
			value = secrets.Mask()
		}
		fmt.Fprintf(writer, "%s\t= %s\n", key, value)
	}
	writer.Flush()
}

func (s *Shell) printCounts(counts map[string]int) {
	if len(counts) == 0 {
		fmt.Fprintln(s.out, "  (none)")
		return
	}
	writer := tabwriter.NewWriter(s.out, 0, 0, 2, ' ', 0)
	for _, key := range plates.SortedMapKeys(counts) {
		fmt.Fprintf(writer, "  %s\t%d\n", key, counts[key])
	}
	writer.Flush()
}

func (s *Shell) resolveCurrentPlate() (plates.Resolution, error) {
	pantryValues, err := s.store.Globals()
	if err != nil {
		return plates.Resolution{}, err
	}
	workspaceValues := map[string]string{}
	if s.session.Current() != "" {
		workspaceValues, err = s.store.WorkspaceValues(s.session.Current())
		if err != nil {
			return plates.Resolution{}, err
		}
	}
	return plates.Resolve(*s.plate, pantryValues, workspaceValues, s.sessionVars), nil
}

func (s *Shell) ingredientNames() ([]string, []string) {
	var required []string
	var optional []string
	for name, ingredient := range s.plate.Ingredients {
		if ingredient.Required {
			required = append(required, name)
		} else {
			optional = append(optional, name)
		}
	}
	sort.Strings(required)
	sort.Strings(optional)
	return required, optional
}

func (s *Shell) printIngredientGroup(names []string, values map[string]string) {
	if len(names) == 0 {
		fmt.Fprintln(s.out, "  (none)")
		return
	}
	writer := tabwriter.NewWriter(s.out, 0, 0, 2, ' ', 0)
	for _, name := range names {
		ingredient := s.plate.Ingredients[name]
		valueText := ""
		if value, ok := values[name]; ok && value != "" {
			if ingredient.Secret {
				value = secrets.Mask()
			}
			valueText = fmt.Sprintf("\t= %s", value)
		} else if ingredient.Required {
			valueText = "\tMISSING"
		}
		secretText := ""
		if ingredient.Secret {
			secretText = "\tsecret"
		}
		defaultText := ""
		if ingredient.Default != "" {
			defaultText = fmt.Sprintf("\tdefault: %s", ingredient.Default)
		}
		fmt.Fprintf(writer, "  %s\t%s%s%s%s\n", name, ingredient.Description, secretText, defaultText, valueText)
	}
	writer.Flush()
}

func (s *Shell) prompt() string {
	if s.currentConfig().PromptStyle == "compact" {
		return "PLATES > "
	}
	if s.forgeMode {
		name := "new"
		if s.draft != nil && s.draft.Name != "" {
			name = s.draft.Name
		}
		return "FORGE[" + name + "] > "
	}
	parts := []string{"PLATES"}
	if current := s.session.Current(); current != "" {
		parts = append(parts, "["+current+"]")
	}
	if s.plate != nil {
		parts = append(parts, "["+s.plate.Key()+"]")
	}
	return strings.Join(parts, "") + " > "
}

func (s *Shell) currentConfig() config.AppConfig {
	if s.appConfig.Theme == "" || s.appConfig.PromptStyle == "" {
		return config.DefaultAppConfig()
	}
	return s.appConfig
}

func (s *Shell) completer() readline.AutoCompleter {
	plateItems := []readline.PrefixCompleterInterface{}
	categoryItems := []readline.PrefixCompleterInterface{}
	if s.browser != nil {
		if index, err := s.browser.Index(); err == nil {
			for _, plate := range index.Plates {
				plateItems = append(plateItems, readline.PcItem(plate.Key()))
			}
			for _, category := range plates.SortedMapKeys(index.Categories()) {
				categoryItems = append(categoryItems, readline.PcItem(category))
			}
		}
	}
	return readline.NewPrefixCompleter(
		readline.PcItem("clear", readline.PcItem("plate")),
		readline.PcItem("config",
			readline.PcItem("show"),
			readline.PcItem("set",
				readline.PcItem("banner", readline.PcItem("true"), readline.PcItem("false")),
				readline.PcItem("tips", readline.PcItem("true"), readline.PcItem("false")),
				readline.PcItem("theme", readline.PcItem("default"), readline.PcItem("minimal")),
				readline.PcItem("prompt_style", readline.PcItem("full"), readline.PcItem("compact")),
				readline.PcItem("store_secret_outputs", readline.PcItem("true"), readline.PcItem("false")),
			),
		),
		readline.PcItem("copy"),
		readline.PcItem("export", readline.PcItem("markdown"), readline.PcItem("json"), readline.PcItem("yaml")),
		readline.PcItem("forge"),
		readline.PcItem("guide",
			readline.PcItem("plates"),
			readline.PcItem("forge"),
			readline.PcItem("variables"),
			readline.PcItem("rack"),
			readline.PcItem("examples"),
			readline.PcItem("safety"),
		),
		readline.PcItem("history"),
		readline.PcItem("help"),
		readline.PcItem("info"),
		readline.PcItem("init"),
		readline.PcItem("list", readline.PcItem("plates")),
		readline.PcItem("ll"),
		readline.PcItem("r"),
		readline.PcItem("render"),
		readline.PcItem("run"),
		readline.PcItem("save", readline.PcItem("output")),
		readline.PcItem("search", readline.PcItem("plates")),
		readline.PcItem("set"),
		readline.PcItem("setg"),
		readline.PcItem("shell", readline.PcItem("clear")),
		readline.PcItem("show",
			readline.PcItem("pantry"),
			readline.PcItem("workspace"),
			readline.PcItem("options"),
			readline.PcItem("rack"),
			readline.PcItem("tags"),
			readline.PcItem("category", categoryItems...),
		),
		readline.PcItem("use", plateItems...),
		readline.PcItem("workspace"),
	)
}

func (s *Shell) printGuide(fields []string) error {
	if len(fields) == 1 {
		fmt.Fprint(s.out, guide.List())
		return nil
	}
	if len(fields) != 2 {
		return errors.New("usage: guide [topic]")
	}
	text, err := guide.Show(fields[1])
	if err != nil {
		return err
	}
	fmt.Fprint(s.out, text)
	return nil
}

func (s *Shell) executeForge(line string, fields []string) error {
	switch fields[0] {
	case "help":
		s.printForgeHelp()
	case "guide":
		return s.printGuide(fields)
	case "cancel":
		s.forgeMode = false
		s.draft = nil
		fmt.Fprintln(s.out, "Forge draft discarded.")
	case "set":
		return s.forgeSet(fields)
	case "add_line":
		return s.forgeAddLine(line)
	case "insert_line":
		return s.forgeInsertLine(line, fields)
	case "delete_line":
		return s.forgeDeleteLine(fields)
	case "clear_lines":
		s.draft.ClearLines()
		fmt.Fprintln(s.out, "Template lines cleared.")
	case "show":
		return s.forgeShow(fields)
	case "add_var", "add_option":
		return s.forgeAddVar(fields)
	case "add_secret_var":
		return s.forgeAddSecretVar(fields)
	case "add_optional_var", "add_optional_option":
		return s.forgeAddOptionalVar(fields)
	case "set_var_required":
		return s.forgeSetVarRequired(fields)
	case "set_var_default":
		return s.forgeSetVarDefault(fields)
	case "rm_var":
		return s.forgeRemoveVar(fields)
	case "add_tag":
		return s.forgeAddTag(fields)
	case "rm_tag":
		return s.forgeRemoveTag(fields)
	case "validate":
		return s.forgeValidate()
	case "save":
		return s.forgeSave(fields)
	default:
		return fmt.Errorf("unknown forge command %q; run 'help' for forge commands", fields[0])
	}
	return nil
}

func (s *Shell) forgeSet(fields []string) error {
	if len(fields) < 3 {
		return errors.New("usage: set name|category|description <value>")
	}
	value := strings.Join(fields[2:], " ")
	switch fields[1] {
	case "name":
		if strings.ContainsAny(value, " \t") {
			return errors.New("draft name cannot contain spaces")
		}
		s.draft.Name = value
	case "category":
		if strings.ContainsAny(value, " \t") {
			return errors.New("draft category cannot contain spaces")
		}
		s.draft.Category = value
	case "description":
		s.draft.Description = value
	default:
		return errors.New("usage: set name|category|description <value>")
	}
	return nil
}

func (s *Shell) forgeAddLine(line string) error {
	text, err := rawArgument(line, "add_line")
	if err != nil {
		return err
	}
	s.draft.AddLine(stripOneQuotePair(text))
	return nil
}

func (s *Shell) forgeInsertLine(line string, fields []string) error {
	if len(fields) < 3 {
		return errors.New("usage: insert_line <number> <text>")
	}
	number, err := strconv.Atoi(fields[1])
	if err != nil {
		return errors.New("line number must be an integer")
	}
	text, err := rawArgumentAfterToken(line, "insert_line", fields[1])
	if err != nil {
		return err
	}
	return s.draft.InsertLine(number, stripOneQuotePair(text))
}

func (s *Shell) forgeDeleteLine(fields []string) error {
	if len(fields) != 2 {
		return errors.New("usage: delete_line <number>")
	}
	number, err := strconv.Atoi(fields[1])
	if err != nil {
		return errors.New("line number must be an integer")
	}
	return s.draft.DeleteLine(number)
}

func (s *Shell) forgeShow(fields []string) error {
	if len(fields) != 2 {
		return errors.New("usage: show draft|lines|vars|tags")
	}
	switch fields[1] {
	case "draft":
		s.printDraft()
	case "lines":
		s.printDraftLines()
	case "vars":
		s.printDraftVars()
	case "tags":
		s.printDraftTags()
	default:
		return errors.New("usage: show draft|lines|vars|tags")
	}
	return nil
}

func (s *Shell) forgeAddVar(fields []string) error {
	if len(fields) < 3 {
		return fmt.Errorf("usage: %s <name> <description>", fields[0])
	}
	return s.draft.AddRequiredVar(fields[1], strings.Join(fields[2:], " "))
}

func (s *Shell) forgeAddSecretVar(fields []string) error {
	if len(fields) < 3 {
		return errors.New("usage: add_secret_var <name> <description>")
	}
	return s.draft.AddSecretVar(fields[1], strings.Join(fields[2:], " "))
}

func (s *Shell) forgeAddOptionalVar(fields []string) error {
	if len(fields) < 4 {
		return fmt.Errorf("usage: %s <name> <default> <description>", fields[0])
	}
	return s.draft.AddOptionalVar(fields[1], fields[2], strings.Join(fields[3:], " "))
}

func (s *Shell) forgeSetVarRequired(fields []string) error {
	if len(fields) != 3 {
		return errors.New("usage: set_var_required <name> <true|false>")
	}
	required, err := strconv.ParseBool(fields[2])
	if err != nil {
		return errors.New("required value must be true or false")
	}
	return s.draft.SetVarRequired(fields[1], required)
}

func (s *Shell) forgeSetVarDefault(fields []string) error {
	if len(fields) < 3 {
		return errors.New("usage: set_var_default <name> <value>")
	}
	return s.draft.SetVarDefault(fields[1], strings.Join(fields[2:], " "))
}

func (s *Shell) forgeRemoveVar(fields []string) error {
	if len(fields) != 2 {
		return errors.New("usage: rm_var <name>")
	}
	s.draft.RemoveVar(fields[1])
	return nil
}

func (s *Shell) forgeAddTag(fields []string) error {
	if len(fields) != 2 {
		return errors.New("usage: add_tag <tag>")
	}
	s.draft.AddTag(fields[1])
	return nil
}

func (s *Shell) forgeRemoveTag(fields []string) error {
	if len(fields) != 2 {
		return errors.New("usage: rm_tag <tag>")
	}
	s.draft.RemoveTag(fields[1])
	return nil
}

func (s *Shell) forgeValidate() error {
	validation, err := s.draft.ValidateDraft()
	if err != nil {
		return err
	}
	fmt.Fprintln(s.out, "Draft is valid.")
	s.printDraftWarnings(validation)
	return nil
}

func (s *Shell) forgeSave(fields []string) error {
	if len(fields) > 2 || (len(fields) == 2 && fields[1] != "--force") {
		return errors.New("usage: save [--force]")
	}
	force := len(fields) == 2 && fields[1] == "--force"
	validation, err := s.draft.ValidateDraft()
	if err != nil {
		return err
	}
	path, err := s.draft.Save(s.paths.RackDir, force)
	if errors.Is(err, plates.ErrPlateExists) {
		fmt.Fprintf(s.out, "Plate already exists: %s\n", s.displayPath(path))
		fmt.Fprintln(s.out, "Use save --force to overwrite.")
		return nil
	}
	if err != nil {
		return err
	}
	key := s.draft.Category + "/" + s.draft.Name
	fmt.Fprintf(s.out, "Saved plate: %s\n", s.displayPath(path))
	s.printDraftWarnings(validation)
	fmt.Fprintf(s.out, "Use it with: use %s\n", key)
	s.forgeMode = false
	s.draft = nil
	return nil
}

func (s *Shell) printDraft() {
	fmt.Fprintf(s.out, "Name: %s\n", valueOrUnset(s.draft.Name))
	fmt.Fprintf(s.out, "Category: %s\n", valueOrUnset(s.draft.Category))
	fmt.Fprintf(s.out, "Description: %s\n", valueOrUnset(s.draft.Description))
	fmt.Fprintf(s.out, "Tags: %s\n", strings.Join(s.draft.Tags, ", "))
	fmt.Fprintf(s.out, "Save Path: %s\n", s.draftSavePath())
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, "Options:")
	s.printDraftVars()
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, "Template Lines:")
	s.printDraftLines()
}

func (s *Shell) printDraftLines() {
	if len(s.draft.Lines) == 0 {
		fmt.Fprintln(s.out, "  (none)")
		return
	}
	writer := tabwriter.NewWriter(s.out, 0, 0, 2, ' ', 0)
	for i, line := range s.draft.Lines {
		fmt.Fprintf(writer, "%d\t%s\n", i+1, line)
	}
	writer.Flush()
}

func (s *Shell) printDraftVars() {
	required, optional := draftIngredientNames(s.draft)
	fmt.Fprintln(s.out, "Required:")
	s.printDraftVarGroup(required)
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, "Optional:")
	s.printDraftVarGroup(optional)
}

func (s *Shell) printDraftVarGroup(names []string) {
	if len(names) == 0 {
		fmt.Fprintln(s.out, "  (none)")
		return
	}
	writer := tabwriter.NewWriter(s.out, 0, 0, 2, ' ', 0)
	for _, name := range names {
		ingredient := s.draft.Ingredients[name]
		secretText := ""
		if ingredient.Secret {
			secretText = "\tsecret"
		}
		defaultText := ""
		if ingredient.Default != "" {
			defaultText = fmt.Sprintf("\tdefault: %s", ingredient.Default)
		}
		fmt.Fprintf(writer, "  %s\t%s%s%s\n", name, ingredient.Description, secretText, defaultText)
	}
	writer.Flush()
}

func (s *Shell) printDraftTags() {
	if len(s.draft.Tags) == 0 {
		fmt.Fprintln(s.out, "(none)")
		return
	}
	for _, tag := range s.draft.Tags {
		fmt.Fprintln(s.out, tag)
	}
}

func (s *Shell) printDraftWarnings(validation plates.DraftValidation) {
	for _, warning := range validation.Warnings {
		fmt.Fprintf(s.out, "Warning: %s\n", warning)
	}
}

func (s *Shell) draftSavePath() string {
	if s.draft.Name == "" || s.draft.Category == "" {
		return "(set name and category first)"
	}
	return filepath.ToSlash(filepath.Join("data", "rack", filepath.FromSlash(s.draft.Category), s.draft.Name+".yml"))
}

func (s *Shell) displayPath(path string) string {
	rel, err := filepath.Rel(s.paths.RootDir, filepath.FromSlash(path))
	if err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
}

func draftIngredientNames(draft *plates.Draft) ([]string, []string) {
	var required []string
	var optional []string
	for name, ingredient := range draft.Ingredients {
		if ingredient.Required {
			required = append(required, name)
		} else {
			optional = append(optional, name)
		}
	}
	sort.Strings(required)
	sort.Strings(optional)
	return required, optional
}

func valueOrUnset(value string) string {
	if value == "" {
		return "(unset)"
	}
	return value
}

func splitCommand(line string) ([]string, error) {
	var fields []string
	var current strings.Builder
	inQuote := false
	quote := rune(0)
	escaped := false
	for _, ch := range line {
		if escaped {
			current.WriteRune(ch)
			escaped = false
			continue
		}
		if ch == '\\' && inQuote {
			escaped = true
			continue
		}
		if inQuote {
			if ch == quote {
				inQuote = false
			} else {
				current.WriteRune(ch)
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			inQuote = true
			quote = ch
			continue
		}
		if ch == ' ' || ch == '\t' {
			if current.Len() > 0 {
				fields = append(fields, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(ch)
	}
	if inQuote {
		return nil, errors.New("unterminated quote")
	}
	if current.Len() > 0 {
		fields = append(fields, current.String())
	}
	return fields, nil
}

func rawArgument(line, command string) (string, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == command {
		return "", fmt.Errorf("usage: %s <text>", command)
	}
	if !strings.HasPrefix(trimmed, command) {
		return "", fmt.Errorf("usage: %s <text>", command)
	}
	return strings.TrimSpace(strings.TrimPrefix(trimmed, command)), nil
}

func rawArgumentAfterToken(line, command, token string) (string, error) {
	rest, err := rawArgument(line, command)
	if err != nil {
		return "", err
	}
	rest = strings.TrimSpace(rest)
	if !strings.HasPrefix(rest, token) {
		return "", fmt.Errorf("usage: %s %s <text>", command, token)
	}
	return strings.TrimSpace(strings.TrimPrefix(rest, token)), nil
}

func stripOneQuotePair(text string) string {
	if len(text) < 2 {
		return text
	}
	if (text[0] == '"' && text[len(text)-1] == '"') || (text[0] == '\'' && text[len(text)-1] == '\'') {
		return text[1 : len(text)-1]
	}
	return text
}

func (s *Shell) printForgeHelp() {
	fmt.Fprintln(s.out, "Forge commands:")
	fmt.Fprintln(s.out, "  set name <value>                         Set draft name")
	fmt.Fprintln(s.out, "  set category <value>                     Set draft category")
	fmt.Fprintln(s.out, "  set description <value>                  Set draft description")
	fmt.Fprintln(s.out, "  add_line <text>                          Append a template line")
	fmt.Fprintln(s.out, "  insert_line <number> <text>              Insert a template line")
	fmt.Fprintln(s.out, "  delete_line <number>                     Delete a template line")
	fmt.Fprintln(s.out, "  clear_lines                              Remove all template lines")
	fmt.Fprintln(s.out, "  show lines                               Show numbered template lines")
	fmt.Fprintln(s.out, "  add_option <name> <description>          Add required option")
	fmt.Fprintln(s.out, "  add_optional_option <name> <default> <desc> Add optional option")
	fmt.Fprintln(s.out, "  add_var <name> <description>             Alias for add_option")
	fmt.Fprintln(s.out, "  add_secret_var <name> <description>      Add required secret option")
	fmt.Fprintln(s.out, "  add_optional_var <name> <default> <desc> Alias for add_optional_option")
	fmt.Fprintln(s.out, "  set_var_required <name> <true|false>     Toggle required flag")
	fmt.Fprintln(s.out, "  set_var_default <name> <value>           Set option default")
	fmt.Fprintln(s.out, "  rm_var <name>                            Remove option")
	fmt.Fprintln(s.out, "  show vars                                Show draft options")
	fmt.Fprintln(s.out, "  add_tag <tag>                            Add tag")
	fmt.Fprintln(s.out, "  rm_tag <tag>                             Remove tag")
	fmt.Fprintln(s.out, "  show tags                                Show draft tags")
	fmt.Fprintln(s.out, "  show draft                               Show full draft preview")
	fmt.Fprintln(s.out, "  guide forge                              Show Forge Mode guide")
	fmt.Fprintln(s.out, "  guide plates                             Show plate YAML guide")
	fmt.Fprintln(s.out, "  validate                                 Validate draft")
	fmt.Fprintln(s.out, "  save [--force]                           Save draft to rack")
	fmt.Fprintln(s.out, "  cancel                                   Discard draft and exit forge")
}

func previewLines(text string, maxLines int) []string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return lines
}

func (s *Shell) printHelp() {
	fmt.Fprintln(s.out, "Available commands:")
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, "Variables and State")
	fmt.Fprintln(s.out, "  show workspace        Show variables in the active workspace")
	fmt.Fprintln(s.out, "  show pantry           Show global pantry variables")
	fmt.Fprintln(s.out, "  show options          Show options for the current plate")
	fmt.Fprintln(s.out, "  set <key> <value>     Set a variable in the active workspace")
	fmt.Fprintln(s.out, "  setg <key> <value>    Set a global pantry variable")
	fmt.Fprintln(s.out, "  workspace <name>      Switch to or create a workspace")
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, "Plate Usage")
	fmt.Fprintln(s.out, "  list plates           List available plates by category")
	fmt.Fprintln(s.out, "  search plates <query> Search plates by name, description, tags, and options")
	fmt.Fprintln(s.out, "  use <plate>           Load a plate")
	fmt.Fprintln(s.out, "  info                  Show loaded plate metadata")
	fmt.Fprintln(s.out, "  ll                    Alias for info")
	fmt.Fprintln(s.out, "  render                Render current plate")
	fmt.Fprintln(s.out, "  r                     Alias for render")
	fmt.Fprintln(s.out, "  run                   Alias for render; does not execute commands")
	fmt.Fprintln(s.out, "  clear plate           Unload current plate")
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, "Forge Mode")
	fmt.Fprintln(s.out, "  forge                 Create a new plate")
	fmt.Fprintln(s.out, "  use forge             Enter Forge Mode")
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, "Rack / Packs")
	fmt.Fprintln(s.out, "  show rack             Show rack summary")
	fmt.Fprintln(s.out, "  show tags             Show tag counts")
	fmt.Fprintln(s.out, "  show category <name>  Show plates in a category")
	fmt.Fprintln(s.out, "  pack list             List exported and imported packs")
	fmt.Fprintln(s.out, "  pack export <name>    Export the rack as a pack")
	fmt.Fprintln(s.out, "  pack inspect <path>   Inspect a pack without importing")
	fmt.Fprintln(s.out, "  pack validate <path>  Validate a pack")
	fmt.Fprintln(s.out, "  pack import <path>    Import a pack")
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, "Output")
	fmt.Fprintln(s.out, "  copy                  Copy latest rendered output to clipboard")
	fmt.Fprintln(s.out, "  save output [file]    Save latest rendered output")
	fmt.Fprintln(s.out, "  history               Show recent rendered outputs")
	fmt.Fprintln(s.out, "  export markdown/json/yaml Export latest render")
	fmt.Fprintln(s.out)
	fmt.Fprintln(s.out, "Guides and Utility")
	fmt.Fprintln(s.out, "  init                  Create data/pantry, data/workspaces, and data/rack")
	fmt.Fprintln(s.out, "  guide                 List built-in guide topics")
	fmt.Fprintln(s.out, "  guide <topic>         Show a built-in guide topic")
	fmt.Fprintln(s.out, "  shell clear           Clear the terminal screen")
	fmt.Fprintln(s.out, "  config show           Show PLATES config")
	fmt.Fprintln(s.out, "  config set <key> <value> Set a config value")
	fmt.Fprintln(s.out, "  help                  Show this help")
	fmt.Fprintln(s.out, "  exit, quit            Leave PLATES")
}
