package shell

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"plates/internal/config"
	"plates/internal/output"
	"plates/internal/plates"
	"plates/internal/secrets"
	"plates/internal/workspace"
)

func TestExecutePhaseOneSession(t *testing.T) {
	store := newMemoryStore()
	out := &bytes.Buffer{}
	sh := &Shell{
		store:       store,
		discoverer:  memoryDiscoverer{},
		loader:      memoryLoader{},
		browser:     memoryBrowser{},
		renderer:    plates.NewTemplateRenderer(),
		session:     workspace.NewSession(),
		sessionVars: map[string]string{},
		out:         out,
	}

	commands := []string{
		"init",
		"workspace devhub",
		"set target 10.129.202.242",
		"setg my_ip 10.10.14.3",
		"show workspace",
		"show pantry",
		"list plates",
	}
	for _, command := range commands {
		if err := sh.Execute(command); err != nil {
			t.Fatalf("Execute(%q) error = %v", command, err)
		}
	}
	if err := sh.Execute("exit"); err != io.EOF {
		t.Fatalf("Execute(exit) error = %v, want EOF", err)
	}

	if store.workspaces["devhub"]["target"] != "10.129.202.242" {
		t.Fatalf("target = %q", store.workspaces["devhub"]["target"])
	}
	if store.globals["my_ip"] != "10.10.14.3" {
		t.Fatalf("my_ip = %q", store.globals["my_ip"])
	}
}

func TestExecuteStampsLoadedPlate(t *testing.T) {
	store := newMemoryStore()
	out := &bytes.Buffer{}
	sh := &Shell{
		store:       store,
		discoverer:  memoryDiscoverer{},
		loader:      memoryLoader{},
		browser:     memoryBrowser{},
		renderer:    plates.NewTemplateRenderer(),
		session:     workspace.NewSession(),
		sessionVars: map[string]string{},
		out:         out,
	}

	commands := []string{
		"workspace devhub",
		"set target 10.129.202.242",
		"set workdir C:\\Users\\knjoh\\code\\boxes\\devhub",
		"setg http_port 8000",
		"use scanning/nmap_full_tcp",
		"show plate",
		"show ingredients",
		"stamp",
	}
	for _, command := range commands {
		if err := sh.Execute(command); err != nil {
			t.Fatalf("Execute(%q) error = %v", command, err)
		}
	}

	got := out.String()
	if !strings.Contains(got, "Loaded plate: scanning/nmap_full_tcp") {
		t.Fatalf("output missing loaded plate message:\n%s", got)
	}
	if !strings.Contains(got, "sudo nmap -p- --min-rate 5000") {
		t.Fatalf("output missing rendered command:\n%s", got)
	}
}

func TestExecuteRefusesStampWhenRequiredIngredientsAreMissing(t *testing.T) {
	store := newMemoryStore()
	out := &bytes.Buffer{}
	sh := &Shell{
		store:       store,
		discoverer:  memoryDiscoverer{},
		loader:      memoryLoader{},
		browser:     memoryBrowser{},
		renderer:    plates.NewTemplateRenderer(),
		session:     workspace.NewSession(),
		sessionVars: map[string]string{},
		out:         out,
	}

	if err := sh.Execute("use scanning/nmap_full_tcp"); err != nil {
		t.Fatalf("use error = %v", err)
	}
	if err := sh.Execute("stamp"); err != nil {
		t.Fatalf("stamp error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Missing required ingredients:") || !strings.Contains(got, "target") {
		t.Fatalf("output missing required ingredient warning:\n%s", got)
	}
}

func TestExecuteShowRack(t *testing.T) {
	store := newMemoryStore()
	out := &bytes.Buffer{}
	sh := &Shell{
		store:       store,
		discoverer:  memoryDiscoverer{},
		loader:      memoryLoader{},
		browser:     memoryBrowser{},
		renderer:    plates.NewTemplateRenderer(),
		session:     workspace.NewSession(),
		sessionVars: map[string]string{},
		out:         out,
	}

	if err := sh.Execute("show rack"); err != nil {
		t.Fatalf("show rack error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Rack Summary") || !strings.Contains(got, "Total plates: 1") {
		t.Fatalf("show rack output =\n%s", got)
	}
}

func TestExecuteGuideCommands(t *testing.T) {
	sh, out := newTestShell()
	if err := sh.Execute("guide"); err != nil {
		t.Fatalf("guide error = %v", err)
	}
	if !strings.Contains(out.String(), "Available guides:") || !strings.Contains(out.String(), "plates") {
		t.Fatalf("guide output =\n%s", out.String())
	}
	out.Reset()
	if err := sh.Execute("guide plates"); err != nil {
		t.Fatalf("guide plates error = %v", err)
	}
	if !strings.Contains(out.String(), "Plate YAML Guide") || !strings.Contains(out.String(), "http_server") {
		t.Fatalf("guide plates output =\n%s", out.String())
	}
	out.Reset()
	if err := sh.Execute("guide forge"); err != nil {
		t.Fatalf("guide forge error = %v", err)
	}
	if !strings.Contains(out.String(), "Forge Mode Guide") || !strings.Contains(out.String(), "save --force") {
		t.Fatalf("guide forge output =\n%s", out.String())
	}
}

func TestExecuteUnknownGuideTopicReturnsUsefulError(t *testing.T) {
	sh, _ := newTestShell()
	err := sh.Execute("guide nope")
	if err == nil {
		t.Fatal("guide nope error = nil")
	}
	if !strings.Contains(err.Error(), "unknown guide topic") {
		t.Fatalf("guide nope error = %v", err)
	}
}

func TestHelpIncludesGuideCommands(t *testing.T) {
	sh, out := newTestShell()
	if err := sh.Execute("help"); err != nil {
		t.Fatalf("help error = %v", err)
	}
	if !strings.Contains(out.String(), "guide") || !strings.Contains(out.String(), "guide <topic>") {
		t.Fatalf("help output =\n%s", out.String())
	}
}

func TestForgeHelpIncludesGuideReferences(t *testing.T) {
	sh, out := newTestShell()
	if err := sh.Execute("forge"); err != nil {
		t.Fatalf("forge error = %v", err)
	}
	out.Reset()
	if err := sh.Execute("help"); err != nil {
		t.Fatalf("forge help error = %v", err)
	}
	if !strings.Contains(out.String(), "guide forge") || !strings.Contains(out.String(), "guide plates") {
		t.Fatalf("forge help output =\n%s", out.String())
	}
}

func TestExecuteLintPlate(t *testing.T) {
	sh, out := newTestShell()
	if err := sh.Execute("use scanning/nmap_full_tcp"); err != nil {
		t.Fatalf("use error = %v", err)
	}
	out.Reset()
	if err := sh.Execute("lint plate"); err != nil {
		t.Fatalf("lint plate error = %v", err)
	}
	if !strings.Contains(out.String(), "PASS") || !strings.Contains(out.String(), "No issues found.") {
		t.Fatalf("lint plate output =\n%s", out.String())
	}
}

func TestExecuteLintRackAndHealth(t *testing.T) {
	sh, out := newTestShell()
	if err := sh.Execute("lint rack"); err != nil {
		t.Fatalf("lint rack error = %v", err)
	}
	if !strings.Contains(out.String(), "Linting 1 plates") || !strings.Contains(out.String(), "PASS: 1") {
		t.Fatalf("lint rack output =\n%s", out.String())
	}
	out.Reset()
	if err := sh.Execute("health"); err != nil {
		t.Fatalf("health error = %v", err)
	}
	if !strings.Contains(out.String(), "Rack Health") || !strings.Contains(out.String(), "Total Plates: 1") {
		t.Fatalf("health output =\n%s", out.String())
	}
}

func TestExecuteExplainLint(t *testing.T) {
	sh, out := newTestShell()
	if err := sh.Execute("explain lint"); err != nil {
		t.Fatalf("explain lint error = %v", err)
	}
	if !strings.Contains(out.String(), "unused ingredient") || !strings.Contains(out.String(), "undeclared variable") {
		t.Fatalf("explain lint output =\n%s", out.String())
	}
}

func TestExecuteForgeShellFlowSavesPlate(t *testing.T) {
	root := t.TempDir()
	paths := config.Paths{
		RootDir:       root,
		DataDir:       filepath.Join(root, "data"),
		PantryDir:     filepath.Join(root, "data", "pantry"),
		WorkspacesDir: filepath.Join(root, "data", "workspaces"),
		RackDir:       filepath.Join(root, "data", "rack"),
		GlobalsFile:   filepath.Join(root, "data", "pantry", "globals.yaml"),
	}
	store := config.NewYAMLStore(paths)
	rack := plates.NewRackRepository(paths.RootDir, paths.RackDir)
	out := &bytes.Buffer{}
	sh := &Shell{
		paths:       paths,
		store:       store,
		discoverer:  rack,
		loader:      rack,
		browser:     rack,
		renderer:    plates.NewTemplateRenderer(),
		session:     workspace.NewSession(),
		sessionVars: map[string]string{},
		out:         out,
	}

	commands := []string{
		"forge",
		"set name nxc_ldap_commands",
		"set category ldap",
		"set description \"Common NetExec LDAP command helper\"",
		"add_line \"# LDAP command helper for {{target}}\"",
		"add_line \"nxc ldap {{target}} -u {{username}} -p {{password}}\"",
		"add_var target \"Target IP address or hostname\"",
		"add_var username Username",
		"add_var password \"Password or quoted empty string\"",
		"add_tag nxc",
		"add_tag ldap",
		"validate",
		"save",
		"use ldap/nxc_ldap_commands",
	}
	for _, command := range commands {
		if err := sh.Execute(command); err != nil {
			t.Fatalf("Execute(%q) error = %v", command, err)
		}
	}
	got := out.String()
	if !strings.Contains(got, "Saved plate: data/rack/ldap/nxc_ldap_commands.yml") {
		t.Fatalf("output missing save message:\n%s", got)
	}
	if !strings.Contains(got, "Loaded plate: ldap/nxc_ldap_commands") {
		t.Fatalf("output missing load message:\n%s", got)
	}
}

func TestExecuteOutputManagementFlow(t *testing.T) {
	sh, out, clip := newOutputTestShell(t)
	commands := []string{
		"workspace devhub",
		"set target 10.129.202.242",
		"set workdir C:\\Users\\knjoh\\code\\boxes\\devhub",
		"use scanning/nmap_full_tcp",
		"stamp",
	}
	for _, command := range commands {
		if err := sh.Execute(command); err != nil {
			t.Fatalf("Execute(%q) error = %v", command, err)
		}
	}
	if !strings.Contains(out.String(), "Stored as Render #1") {
		t.Fatalf("stamp output =\n%s", out.String())
	}

	out.Reset()
	if err := sh.Execute("output history"); err != nil {
		t.Fatalf("output history error = %v", err)
	}
	if !strings.Contains(out.String(), "Recent Output History") || !strings.Contains(out.String(), "scanning/nmap_full_tcp") {
		t.Fatalf("history output =\n%s", out.String())
	}

	out.Reset()
	if err := sh.Execute("copy"); err != nil {
		t.Fatalf("copy error = %v", err)
	}
	if !strings.Contains(clip.text, "sudo nmap") {
		t.Fatalf("clipboard text = %q", clip.text)
	}

	out.Reset()
	if err := sh.Execute("save output my_scan.txt"); err != nil {
		t.Fatalf("save output error = %v", err)
	}
	if !strings.Contains(out.String(), "my_scan.txt") {
		t.Fatalf("save output =\n%s", out.String())
	}

	out.Reset()
	if err := sh.Execute("export markdown"); err != nil {
		t.Fatalf("export markdown error = %v", err)
	}
	if !strings.Contains(out.String(), ".md") {
		t.Fatalf("export markdown =\n%s", out.String())
	}

	out.Reset()
	if err := sh.Execute("output stats"); err != nil {
		t.Fatalf("output stats error = %v", err)
	}
	if !strings.Contains(out.String(), "Total Renders: 1") {
		t.Fatalf("stats output =\n%s", out.String())
	}

	out.Reset()
	if err := sh.Execute("output clear"); err != nil {
		t.Fatalf("output clear error = %v", err)
	}
	if err := sh.Execute("YES"); err != nil {
		t.Fatalf("YES error = %v", err)
	}
	if !strings.Contains(out.String(), "Output history cleared.") {
		t.Fatalf("clear output =\n%s", out.String())
	}
}

func TestSecretCommandsMaskAndReveal(t *testing.T) {
	sh, out, _ := newOutputTestShell(t)
	if err := sh.Execute("secret set password SuperSecret123"); err != nil {
		t.Fatalf("secret set error = %v", err)
	}
	out.Reset()
	if err := sh.Execute("secret get password"); err != nil {
		t.Fatalf("secret get error = %v", err)
	}
	if strings.Contains(out.String(), "SuperSecret123") || !strings.Contains(out.String(), "********") {
		t.Fatalf("secret get output =\n%s", out.String())
	}
	out.Reset()
	if err := sh.Execute("secret reveal password"); err != nil {
		t.Fatalf("secret reveal error = %v", err)
	}
	if !strings.Contains(out.String(), "SuperSecret123") {
		t.Fatalf("secret reveal output =\n%s", out.String())
	}
	out.Reset()
	if err := sh.Execute("secret list"); err != nil {
		t.Fatalf("secret list error = %v", err)
	}
	if strings.Contains(out.String(), "SuperSecret123") || !strings.Contains(out.String(), "password") {
		t.Fatalf("secret list output =\n%s", out.String())
	}
	if err := sh.Execute("secret delete password"); err != nil {
		t.Fatalf("secret delete error = %v", err)
	}
}

func TestSecretIngredientDisplaysMasked(t *testing.T) {
	sh, out := newSecretPlateShell(t, false)
	if err := sh.Execute("workspace devhub"); err != nil {
		t.Fatalf("workspace error = %v", err)
	}
	if err := sh.Execute("set username alice"); err != nil {
		t.Fatalf("set username error = %v", err)
	}
	if err := sh.Execute("set password SuperSecret123"); err != nil {
		t.Fatalf("set password error = %v", err)
	}
	if err := sh.Execute("use examples/secret_demo"); err != nil {
		t.Fatalf("use error = %v", err)
	}
	out.Reset()
	if err := sh.Execute("show ingredients"); err != nil {
		t.Fatalf("show ingredients error = %v", err)
	}
	if strings.Contains(out.String(), "SuperSecret123") || !strings.Contains(out.String(), "password") || !strings.Contains(out.String(), "********") {
		t.Fatalf("show ingredients output =\n%s", out.String())
	}
}

func TestRenderHistoryStoresRedactedOutputByDefault(t *testing.T) {
	t.Setenv("HOME", "/tmp/home")
	sh, out := newSecretPlateShell(t, false)
	if err := sh.Execute("workspace devhub"); err != nil {
		t.Fatalf("workspace error = %v", err)
	}
	if err := sh.Execute("set username alice"); err != nil {
		t.Fatalf("set username error = %v", err)
	}
	if err := sh.Execute("secret set password SuperSecret123"); err != nil {
		t.Fatalf("secret set error = %v", err)
	}
	if err := sh.Execute("use examples/secret_demo"); err != nil {
		t.Fatalf("use error = %v", err)
	}
	if err := sh.Execute("stamp"); err != nil {
		t.Fatalf("stamp error = %v", err)
	}
	out.Reset()
	if err := sh.Execute("output show 1"); err != nil {
		t.Fatalf("output show error = %v", err)
	}
	if strings.Contains(out.String(), "SuperSecret123") || !strings.Contains(out.String(), "********") {
		t.Fatalf("output show =\n%s", out.String())
	}
	if err := sh.Execute("output show 1 --reveal"); err == nil {
		t.Fatal("output show --reveal error = nil")
	}
	record, err := sh.outputStore.Get("1")
	if err != nil {
		t.Fatalf("Get(1) error = %v", err)
	}
	if record.Variables["password"] == "SuperSecret123" {
		t.Fatalf("secret variable stored raw: %#v", record.Variables)
	}
}

func TestConfigTrueAllowsRawSecretOutputStorage(t *testing.T) {
	t.Setenv("HOME", "/tmp/home")
	sh, out := newSecretPlateShell(t, true)
	if err := sh.Execute("workspace devhub"); err != nil {
		t.Fatalf("workspace error = %v", err)
	}
	_ = sh.Execute("set username alice")
	_ = sh.Execute("secret set password SuperSecret123")
	_ = sh.Execute("use examples/secret_demo")
	if err := sh.Execute("stamp"); err != nil {
		t.Fatalf("stamp error = %v", err)
	}
	out.Reset()
	if err := sh.Execute("output show 1 --reveal"); err != nil {
		t.Fatalf("output show --reveal error = %v", err)
	}
	if !strings.Contains(out.String(), "SuperSecret123") {
		t.Fatalf("output show --reveal =\n%s", out.String())
	}
}

func TestForgeAddSecretVar(t *testing.T) {
	sh, out := newTestShell()
	if err := sh.Execute("forge"); err != nil {
		t.Fatalf("forge error = %v", err)
	}
	if err := sh.Execute("add_secret_var password Password"); err != nil {
		t.Fatalf("add_secret_var error = %v", err)
	}
	out.Reset()
	if err := sh.Execute("show vars"); err != nil {
		t.Fatalf("show vars error = %v", err)
	}
	if !strings.Contains(out.String(), "password") || !strings.Contains(out.String(), "secret") {
		t.Fatalf("show vars output =\n%s", out.String())
	}
}

func TestConfigShowAndSet(t *testing.T) {
	sh, out := newConfigTestShell(t)
	if err := sh.Execute("init"); err != nil {
		t.Fatalf("init error = %v", err)
	}
	out.Reset()
	if err := sh.Execute("config show"); err != nil {
		t.Fatalf("config show error = %v", err)
	}
	if !strings.Contains(out.String(), "banner: true") {
		t.Fatalf("config show output =\n%s", out.String())
	}
	out.Reset()
	if err := sh.Execute("config set banner false"); err != nil {
		t.Fatalf("config set banner error = %v", err)
	}
	if sh.currentConfig().Banner {
		t.Fatalf("Banner = true")
	}
	if err := sh.Execute("config set theme neon"); err == nil {
		t.Fatal("config set theme neon error = nil")
	}
}

func TestBannerEnabledDisabledBehavior(t *testing.T) {
	sh, out := newConfigTestShell(t)
	sh.appConfig = config.DefaultAppConfig()
	sh.printStartup()
	if !strings.Contains(out.String(), "Plate-Based Command Rendering System") {
		t.Fatalf("banner output =\n%s", out.String())
	}
	out.Reset()
	sh.appConfig.Banner = false
	sh.printStartup()
	if out.Len() != 0 {
		t.Fatalf("disabled banner output =\n%s", out.String())
	}
}

func TestPromptStyleFullAndCompact(t *testing.T) {
	sh, _ := newConfigTestShell(t)
	sh.session.Use("devhub")
	plate := memoryPlate()
	sh.plate = &plate
	sh.appConfig = config.DefaultAppConfig()
	if got := sh.prompt(); got != "PLATES[devhub][scanning/nmap_full_tcp] > " {
		t.Fatalf("full prompt = %q", got)
	}
	sh.appConfig.PromptStyle = "compact"
	if got := sh.prompt(); got != "PLATES > " {
		t.Fatalf("compact prompt = %q", got)
	}
}

func TestTipAndFortuneReturnOutput(t *testing.T) {
	sh, out := newTestShell()
	if err := sh.Execute("tip"); err != nil {
		t.Fatalf("tip error = %v", err)
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Fatal("tip output empty")
	}
	out.Reset()
	if err := sh.Execute("fortune"); err != nil {
		t.Fatalf("fortune error = %v", err)
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Fatal("fortune output empty")
	}
}

func TestRandomPlateAndUse(t *testing.T) {
	sh, out := newTestShell()
	if err := sh.Execute("random plate"); err != nil {
		t.Fatalf("random plate error = %v", err)
	}
	if !strings.Contains(out.String(), "Random Plate") || !strings.Contains(out.String(), "scanning/nmap_full_tcp") {
		t.Fatalf("random plate output =\n%s", out.String())
	}
	out.Reset()
	if err := sh.Execute("random plate --use"); err != nil {
		t.Fatalf("random plate --use error = %v", err)
	}
	if sh.plate == nil || sh.plate.Key() != "scanning/nmap_full_tcp" {
		t.Fatalf("loaded plate = %#v", sh.plate)
	}
}

func TestVersionAndAboutOutput(t *testing.T) {
	sh, out := newTestShell()
	if err := sh.Execute("version"); err != nil {
		t.Fatalf("version error = %v", err)
	}
	if !strings.Contains(out.String(), "PLATES version 0.8.0") {
		t.Fatalf("version output =\n%s", out.String())
	}
	out.Reset()
	if err := sh.Execute("about"); err != nil {
		t.Fatalf("about error = %v", err)
	}
	if !strings.Contains(out.String(), "render-only") || !strings.Contains(out.String(), "does not execute") {
		t.Fatalf("about output =\n%s", out.String())
	}
}

func TestHelpIncludesPhaseEightCommands(t *testing.T) {
	sh, out := newTestShell()
	if err := sh.Execute("help"); err != nil {
		t.Fatalf("help error = %v", err)
	}
	for _, want := range []string{"config show", "config set <key> <value>", "tip", "fortune", "random plate", "version", "about"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("help missing %q:\n%s", want, out.String())
		}
	}
}

type memoryStore struct {
	globals    map[string]string
	workspaces map[string]map[string]string
}

func newTestShell() (*Shell, *bytes.Buffer) {
	store := newMemoryStore()
	out := &bytes.Buffer{}
	return &Shell{
		store:       store,
		discoverer:  memoryDiscoverer{},
		loader:      memoryLoader{},
		browser:     memoryBrowser{},
		renderer:    plates.NewTemplateRenderer(),
		session:     workspace.NewSession(),
		sessionVars: map[string]string{},
		out:         out,
	}, out
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		globals:    map[string]string{},
		workspaces: map[string]map[string]string{},
	}
}

func (s *memoryStore) EnsureBase() error {
	return nil
}

func (s *memoryStore) SetGlobal(key, value string) error {
	s.globals[key] = value
	return nil
}

func (s *memoryStore) Globals() (map[string]string, error) {
	return s.globals, nil
}

func (s *memoryStore) EnsureWorkspace(name string) error {
	if _, ok := s.workspaces[name]; !ok {
		s.workspaces[name] = map[string]string{}
	}
	return nil
}

func (s *memoryStore) SetWorkspaceValue(workspaceName, key, value string) error {
	if err := s.EnsureWorkspace(workspaceName); err != nil {
		return err
	}
	s.workspaces[workspaceName][key] = value
	return nil
}

func (s *memoryStore) WorkspaceValues(workspaceName string) (map[string]string, error) {
	if err := s.EnsureWorkspace(workspaceName); err != nil {
		return nil, err
	}
	return s.workspaces[workspaceName], nil
}

type memoryDiscoverer struct{}

func (memoryDiscoverer) List() ([]string, error) {
	return []string{"scanning/nmap_full_tcp"}, nil
}

type memoryLoader struct{}

func (memoryLoader) Load(selector string) (plates.Plate, error) {
	return memoryPlate(), nil
}

type memoryBrowser struct{}

func (memoryBrowser) Index() (*plates.RackIndex, error) {
	return plates.NewRackIndex("data/rack", []plates.Plate{
		memoryPlate(),
	}), nil
}

func memoryPlate() plates.Plate {
	return plates.Plate{
		Name:        "nmap_full_tcp",
		Category:    "scanning",
		Description: "Full TCP port scan with output files",
		Tags:        []string{"nmap", "scanning", "tcp"},
		Path:        "data/rack/scanning/nmap_full_tcp.yml",
		Ingredients: map[string]plates.Ingredient{
			"target": {
				Description: "Target IP address or hostname",
				Required:    true,
			},
			"workdir": {
				Description: "Working directory for output files",
				Required:    true,
			},
			"rate": {
				Description: "Minimum packet rate",
				Default:     "5000",
			},
		},
		Template: "sudo nmap -p- --min-rate {{rate}} -oA {{workdir}}/nmap/full_tcp {{target}}",
	}
}

type fakeClipboard struct {
	text string
}

func (f *fakeClipboard) Copy(text string) error {
	f.text = text
	return nil
}

func newOutputTestShell(t *testing.T) (*Shell, *bytes.Buffer, *fakeClipboard) {
	t.Helper()
	paths := config.Paths{
		RootDir:       t.TempDir(),
		DataDir:       "",
		PantryDir:     "",
		WorkspacesDir: "",
		RackDir:       "",
		GlobalsFile:   "",
	}
	paths.DataDir = filepath.Join(paths.RootDir, "data")
	paths.PantryDir = filepath.Join(paths.DataDir, "pantry")
	paths.WorkspacesDir = filepath.Join(paths.DataDir, "workspaces")
	paths.RackDir = filepath.Join(paths.DataDir, "rack")
	paths.GlobalsFile = filepath.Join(paths.PantryDir, "globals.yaml")
	paths.SecretsDir = filepath.Join(paths.DataDir, "secrets")
	paths.SecretsFile = filepath.Join(paths.SecretsDir, "secrets.yaml")
	store := newMemoryStore()
	out := &bytes.Buffer{}
	clip := &fakeClipboard{}
	return &Shell{
		paths:       paths,
		store:       store,
		discoverer:  memoryDiscoverer{},
		loader:      memoryLoader{},
		browser:     memoryBrowser{},
		renderer:    plates.NewTemplateRenderer(),
		outputStore: output.NewStore(paths.DataDir),
		clipboard:   clip,
		secretStore: secrets.NewStore(paths.SecretsFile),
		session:     workspace.NewSession(),
		sessionVars: map[string]string{},
		out:         out,
	}, out, clip
}

func newSecretPlateShell(t *testing.T, storeRaw bool) (*Shell, *bytes.Buffer) {
	t.Helper()
	sh, out, _ := newOutputTestShell(t)
	sh.appConfig = config.DefaultAppConfig()
	sh.appConfig.StoreSecretOutputs = storeRaw
	sh.loader = secretLoader{}
	sh.browser = secretBrowser{}
	return sh, out
}

type secretLoader struct{}

func (secretLoader) Load(selector string) (plates.Plate, error) {
	return secretDemoPlate(), nil
}

type secretBrowser struct{}

func (secretBrowser) Index() (*plates.RackIndex, error) {
	return plates.NewRackIndex("data/rack", []plates.Plate{secretDemoPlate()}), nil
}

func secretDemoPlate() plates.Plate {
	return plates.Plate{
		Name:        "secret_demo",
		Category:    "examples",
		Description: "Demonstrate secret and environment variable rendering",
		Tags:        []string{"secrets", "demo"},
		Ingredients: map[string]plates.Ingredient{
			"username": {Description: "Example username", Required: true},
			"password": {Description: "Example password stored as a secret ingredient", Required: true, Secret: true},
		},
		Template: "echo \"User: {{username}}\"\necho \"Password: {{secret \"password\"}}\"\necho \"Home: {{env \"HOME\"}}\"\n",
	}
}

func newConfigTestShell(t *testing.T) (*Shell, *bytes.Buffer) {
	t.Helper()
	paths := config.Paths{RootDir: t.TempDir()}
	paths.DataDir = filepath.Join(paths.RootDir, "data")
	paths.PantryDir = filepath.Join(paths.DataDir, "pantry")
	paths.WorkspacesDir = filepath.Join(paths.DataDir, "workspaces")
	paths.RackDir = filepath.Join(paths.DataDir, "rack")
	paths.GlobalsFile = filepath.Join(paths.PantryDir, "globals.yaml")
	paths.ConfigFile = filepath.Join(paths.DataDir, "config.yaml")
	paths.SecretsDir = filepath.Join(paths.DataDir, "secrets")
	paths.SecretsFile = filepath.Join(paths.SecretsDir, "secrets.yaml")
	out := &bytes.Buffer{}
	return &Shell{
		paths:       paths,
		store:       config.NewYAMLStore(paths),
		configStore: config.NewAppConfigStore(paths.ConfigFile),
		appConfig:   config.DefaultAppConfig(),
		discoverer:  memoryDiscoverer{},
		loader:      memoryLoader{},
		browser:     memoryBrowser{},
		renderer:    plates.NewTemplateRenderer(),
		outputStore: output.NewStore(paths.DataDir),
		clipboard:   &fakeClipboard{},
		session:     workspace.NewSession(),
		sessionVars: map[string]string{},
		out:         out,
	}, out
}
