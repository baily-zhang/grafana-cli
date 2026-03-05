package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matiasvillaverde/grafana-cli/internal/config"
	"github.com/matiasvillaverde/grafana-cli/internal/grafana"
)

type fakeStore struct {
	cfg      config.Config
	loadErr  error
	saveErr  error
	clearErr error
}

func (f *fakeStore) Load() (config.Config, error) {
	if f.loadErr != nil {
		return config.Config{}, f.loadErr
	}
	return f.cfg, nil
}

func (f *fakeStore) Save(cfg config.Config) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.cfg = cfg
	return nil
}

func (f *fakeStore) Clear() error {
	if f.clearErr != nil {
		return f.clearErr
	}
	f.cfg = config.Config{}
	return nil
}

func (f *fakeStore) Path() string {
	return "/tmp/config.json"
}

type fakeContextStore struct {
	cfgs        map[string]config.Config
	current     string
	loadErr     error
	saveErr     error
	clearErr    error
	listErr     error
	currentErr  error
	useErr      error
	loadCtxErr  error
	saveCtxErr  error
	currentPath string
}

func (f *fakeContextStore) Load() (config.Config, error) {
	if f.loadErr != nil {
		return config.Config{}, f.loadErr
	}
	name := f.current
	if name == "" {
		name = "default"
	}
	cfg := f.cfgs[name]
	cfg.ApplyDefaults()
	return cfg, nil
}

func (f *fakeContextStore) Save(cfg config.Config) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	name := f.current
	if name == "" {
		name = "default"
	}
	if f.cfgs == nil {
		f.cfgs = map[string]config.Config{}
	}
	f.cfgs[name] = cfg
	return nil
}

func (f *fakeContextStore) Clear() error {
	if f.clearErr != nil {
		return f.clearErr
	}
	name := f.current
	if name == "" {
		name = "default"
	}
	delete(f.cfgs, name)
	return nil
}

func (f *fakeContextStore) Path() string {
	if f.currentPath != "" {
		return f.currentPath
	}
	return "/tmp/config.json"
}

func (f *fakeContextStore) CurrentContext() (string, error) {
	if f.currentErr != nil {
		return "", f.currentErr
	}
	if f.current == "" {
		return "default", nil
	}
	return f.current, nil
}

func (f *fakeContextStore) ListContexts() ([]config.ContextSummary, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if len(f.cfgs) == 0 {
		return []config.ContextSummary{{Name: "default", Current: true}}, nil
	}
	items := make([]config.ContextSummary, 0, len(f.cfgs))
	current, _ := f.CurrentContext()
	for name, cfg := range f.cfgs {
		items = append(items, config.ContextSummary{
			Name:          name,
			Current:       name == current,
			Authenticated: cfg.IsAuthenticated(),
			BaseURL:       cfg.BaseURL,
			CloudURL:      cfg.CloudURL,
		})
	}
	return items, nil
}

func (f *fakeContextStore) UseContext(name string) error {
	if f.useErr != nil {
		return f.useErr
	}
	if _, ok := f.cfgs[name]; !ok && name != "default" {
		return errors.New("context not found")
	}
	f.current = name
	return nil
}

func (f *fakeContextStore) LoadContext(name string) (config.Config, error) {
	if f.loadCtxErr != nil {
		return config.Config{}, f.loadCtxErr
	}
	cfg := f.cfgs[name]
	cfg.ApplyDefaults()
	return cfg, nil
}

func (f *fakeContextStore) SaveContext(name string, cfg config.Config) error {
	if f.saveCtxErr != nil {
		return f.saveCtxErr
	}
	if f.cfgs == nil {
		f.cfgs = map[string]config.Config{}
	}
	f.cfgs[name] = cfg
	f.current = name
	return nil
}

type fakeClient struct {
	rawResult           any
	rawErr              error
	cloudResult         any
	cloudErr            error
	searchDashResult    any
	searchDashErr       error
	getDashResult       any
	getDashErr          error
	createDashResult    any
	createDashErr       error
	deleteDashResult    any
	deleteDashErr       error
	dashVersionsResult  any
	dashVersionsErr     error
	renderDashboardResp grafana.RenderedDashboard
	renderDashboardErr  error
	renderDashboardReq  grafana.DashboardRenderRequest
	listDSResult        any
	listDSErr           error
	listFoldersResult   any
	listFoldersErr      error
	getFolderResult     any
	getFolderErr        error
	annotationsResult   any
	annotationsErr      error
	annotationsReq      grafana.AnnotationListRequest
	alertRulesResult    any
	alertRulesErr       error
	alertContactResult  any
	alertContactErr     error
	alertPoliciesResult any
	alertPoliciesErr    error
	assistantChatResult any
	assistantChatErr    error
	assistantStatusResp any
	assistantStatusErr  error
	assistantSkillsResp any
	assistantSkillsErr  error
	assistantPrompt     string
	assistantChatID     string
	assistantStatusID   string
	metricsResult       any
	metricsErr          error
	logsResult          any
	logsErr             error
	tracesResult        any
	tracesErr           error
	aggregateResult     grafana.AggregateSnapshot
	aggregateErr        error
	aggregateReq        grafana.AggregateRequest
	createDashboardArg  map[string]any
	createFolderID      int64
	createOverwrite     bool
}

func (f *fakeClient) Raw(_ context.Context, _, _ string, _ any) (any, error) {
	return f.rawResult, f.rawErr
}

func (f *fakeClient) CloudStacks(_ context.Context) (any, error) {
	return f.cloudResult, f.cloudErr
}

func (f *fakeClient) SearchDashboards(_ context.Context, _, _ string, _ int) (any, error) {
	return f.searchDashResult, f.searchDashErr
}

func (f *fakeClient) GetDashboard(_ context.Context, _ string) (any, error) {
	return f.getDashResult, f.getDashErr
}

func (f *fakeClient) CreateDashboard(_ context.Context, dashboard map[string]any, folderID int64, overwrite bool) (any, error) {
	f.createDashboardArg = dashboard
	f.createFolderID = folderID
	f.createOverwrite = overwrite
	return f.createDashResult, f.createDashErr
}

func (f *fakeClient) DeleteDashboard(_ context.Context, _ string) (any, error) {
	return f.deleteDashResult, f.deleteDashErr
}

func (f *fakeClient) DashboardVersions(_ context.Context, _ string, _ int) (any, error) {
	return f.dashVersionsResult, f.dashVersionsErr
}

func (f *fakeClient) RenderDashboard(_ context.Context, req grafana.DashboardRenderRequest) (grafana.RenderedDashboard, error) {
	f.renderDashboardReq = req
	return f.renderDashboardResp, f.renderDashboardErr
}

func (f *fakeClient) ListDatasources(_ context.Context) (any, error) {
	return f.listDSResult, f.listDSErr
}

func (f *fakeClient) ListFolders(_ context.Context) (any, error) {
	return f.listFoldersResult, f.listFoldersErr
}

func (f *fakeClient) GetFolder(_ context.Context, _ string) (any, error) {
	return f.getFolderResult, f.getFolderErr
}

func (f *fakeClient) ListAnnotations(_ context.Context, req grafana.AnnotationListRequest) (any, error) {
	f.annotationsReq = req
	return f.annotationsResult, f.annotationsErr
}

func (f *fakeClient) AlertingRules(_ context.Context) (any, error) {
	return f.alertRulesResult, f.alertRulesErr
}

func (f *fakeClient) AlertingContactPoints(_ context.Context) (any, error) {
	return f.alertContactResult, f.alertContactErr
}

func (f *fakeClient) AlertingPolicies(_ context.Context) (any, error) {
	return f.alertPoliciesResult, f.alertPoliciesErr
}

func (f *fakeClient) AssistantChat(_ context.Context, prompt, chatID string) (any, error) {
	f.assistantPrompt = prompt
	f.assistantChatID = chatID
	return f.assistantChatResult, f.assistantChatErr
}

func (f *fakeClient) AssistantChatStatus(_ context.Context, chatID string) (any, error) {
	f.assistantStatusID = chatID
	return f.assistantStatusResp, f.assistantStatusErr
}

func (f *fakeClient) AssistantSkills(_ context.Context) (any, error) {
	return f.assistantSkillsResp, f.assistantSkillsErr
}

func (f *fakeClient) MetricsRange(_ context.Context, _, _, _, _ string) (any, error) {
	return f.metricsResult, f.metricsErr
}

func (f *fakeClient) LogsRange(_ context.Context, _, _, _ string, _ int) (any, error) {
	return f.logsResult, f.logsErr
}

func (f *fakeClient) TracesSearch(_ context.Context, _, _, _ string, _ int) (any, error) {
	return f.tracesResult, f.tracesErr
}

func (f *fakeClient) AggregateSnapshot(_ context.Context, req grafana.AggregateRequest) (grafana.AggregateSnapshot, error) {
	f.aggregateReq = req
	return f.aggregateResult, f.aggregateErr
}

func newTestApp(store config.Store, client APIClient) (*App, *strings.Builder, *strings.Builder) {
	out := &strings.Builder{}
	errOut := &strings.Builder{}
	app := NewApp(store)
	app.Out = out
	app.Err = errOut
	app.NewClient = func(config.Config) APIClient { return client }
	app.Now = func() time.Time { return time.Date(2026, 3, 5, 15, 4, 0, 0, time.UTC) }
	return app, out, errOut
}

func decodeJSON(t *testing.T, value string) map[string]any {
	t.Helper()
	out := map[string]any{}
	if err := json.Unmarshal([]byte(value), &out); err != nil {
		t.Fatalf("invalid JSON output: %v, value=%s", err, value)
	}
	return out
}

func decodeJSONArray(t *testing.T, value string) []map[string]any {
	t.Helper()
	out := []map[string]any{}
	if err := json.Unmarshal([]byte(value), &out); err != nil {
		t.Fatalf("invalid JSON array output: %v, value=%s", err, value)
	}
	return out
}

func TestParseGlobalOptions(t *testing.T) {
	opts, rest, err := parseGlobalOptions([]string{"--fields", "a,b", "--output", "pretty", "auth", "status"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Output != "pretty" || len(opts.Fields) != 2 {
		t.Fatalf("unexpected opts: %+v", opts)
	}
	if len(rest) != 2 || rest[0] != "auth" {
		t.Fatalf("unexpected rest: %+v", rest)
	}

	opts, _, err = parseGlobalOptions([]string{"--output=json", "--fields=x.y"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Output != "json" || len(opts.Fields) != 1 {
		t.Fatalf("unexpected opts: %+v", opts)
	}

	if _, _, err := parseGlobalOptions([]string{"--output"}); err == nil {
		t.Fatalf("expected missing output error")
	}
	if _, _, err := parseGlobalOptions([]string{"--fields"}); err == nil {
		t.Fatalf("expected missing fields error")
	}
	if _, _, err := parseGlobalOptions([]string{"--output", "bad"}); err == nil {
		t.Fatalf("expected invalid output error")
	}
}

func TestParseGlobalOptionsExtended(t *testing.T) {
	opts, rest, err := parseGlobalOptions([]string{"--json", "a,b", "--jq", ".x", "context", "view"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Output != "json" || opts.JQ != ".x" || len(opts.Fields) != 2 {
		t.Fatalf("unexpected opts: %+v", opts)
	}
	if len(rest) != 2 || rest[0] != "context" || rest[1] != "view" {
		t.Fatalf("unexpected rest: %+v", rest)
	}

	opts, _, err = parseGlobalOptions([]string{"--template={{.context}}", "context", "view"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Template != "{{.context}}" {
		t.Fatalf("unexpected template: %+v", opts)
	}

	if _, _, err := parseGlobalOptions([]string{"--json"}); err == nil {
		t.Fatalf("expected missing json error")
	}
	if _, _, err := parseGlobalOptions([]string{"--jq"}); err == nil {
		t.Fatalf("expected missing jq error")
	}
	if _, _, err := parseGlobalOptions([]string{"--template"}); err == nil {
		t.Fatalf("expected missing template error")
	}
	if _, _, err := parseGlobalOptions([]string{"--jq", ".x", "--template", "{{.}}"}); err == nil {
		t.Fatalf("expected jq/template conflict")
	}
}

func TestNewAppDefaults(t *testing.T) {
	app := NewApp(&fakeStore{})
	if app.Out == nil || app.Err == nil || app.NewClient == nil || app.Now == nil {
		t.Fatalf("expected defaults to be initialized")
	}
	if app.NewClient(config.Config{}) == nil {
		t.Fatalf("expected default client factory to return a client")
	}
}

func TestNewAppWithContextStore(t *testing.T) {
	store := &fakeContextStore{cfgs: map[string]config.Config{"default": {}}}
	app := NewApp(store)
	if app.Contexts == nil {
		t.Fatalf("expected context store to be wired")
	}
}

func TestRunHelpAndUnknown(t *testing.T) {
	store := &fakeStore{}
	client := &fakeClient{}
	app, out, errOut := newTestApp(store, client)

	if code := app.Run(context.Background(), []string{}); code != 0 {
		t.Fatalf("expected success for help")
	}
	resp := decodeJSON(t, out.String())
	commands, ok := resp["commands"].([]any)
	if !ok {
		t.Fatalf("expected commands output")
	}
	foundAssistant := false
	foundContext := false
	for _, command := range commands {
		if value, _ := command.(string); value == "assistant" {
			foundAssistant = true
		}
		if value, _ := command.(string); value == "context" {
			foundContext = true
		}
	}
	if !foundAssistant || !foundContext {
		t.Fatalf("expected assistant and context commands in help output")
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"-help"}); code != 0 {
		t.Fatalf("expected root -help to succeed")
	}
	if _, ok := decodeJSON(t, out.String())["commands"]; !ok {
		t.Fatalf("expected command list for root -help")
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"bad"}); code != 1 {
		t.Fatalf("expected failure for unknown command")
	}
	if !strings.Contains(errOut.String(), "unknown command") {
		t.Fatalf("expected unknown command error, got %s", errOut.String())
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"--output"}); code != 1 {
		t.Fatalf("expected parse failure")
	}
	if !strings.Contains(errOut.String(), "--output requires a value") {
		t.Fatalf("expected global option error, got %s", errOut.String())
	}
}

func TestAuthFlows(t *testing.T) {
	store := &fakeStore{}
	client := &fakeClient{}
	app, out, errOut := newTestApp(store, client)

	if code := app.Run(context.Background(), []string{"auth", "login", "--token", "abc", "--base-url", "https://stack"}); code != 0 {
		t.Fatalf("auth login should succeed, err=%s", errOut.String())
	}
	resp := decodeJSON(t, out.String())
	if resp["status"] != "authenticated" {
		t.Fatalf("unexpected login response: %+v", resp)
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"auth", "status"}); code != 0 {
		t.Fatalf("status should succeed")
	}
	resp = decodeJSON(t, out.String())
	if resp["status"] != "authenticated" {
		t.Fatalf("expected authenticated status")
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"auth", "logout"}); code != 0 {
		t.Fatalf("logout should succeed")
	}
	resp = decodeJSON(t, out.String())
	if resp["status"] != "logged_out" {
		t.Fatalf("unexpected logout response")
	}

	if code := app.Run(context.Background(), []string{"auth", "login"}); code != 1 {
		t.Fatalf("missing token should fail")
	}
	if !strings.Contains(errOut.String(), "--token is required") {
		t.Fatalf("expected token error")
	}

	if code := app.Run(context.Background(), []string{"auth", "bad"}); code != 1 {
		t.Fatalf("unknown auth command should fail")
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"auth"}); code != 0 {
		t.Fatalf("auth summary should succeed")
	}
	if _, ok := decodeJSON(t, out.String())["commands"]; !ok {
		t.Fatalf("expected auth command list")
	}

	if code := app.Run(context.Background(), []string{"auth", "login", "--bad"}); code != 1 {
		t.Fatalf("unknown auth login flag should fail")
	}
}

func TestCommandErrorsFromStore(t *testing.T) {
	store := &fakeStore{loadErr: errors.New("load fail")}
	client := &fakeClient{}
	app, _, errOut := newTestApp(store, client)

	if code := app.Run(context.Background(), []string{"auth", "status"}); code != 1 {
		t.Fatalf("expected failure")
	}
	if !strings.Contains(errOut.String(), "load fail") {
		t.Fatalf("expected load fail error")
	}

	store = &fakeStore{cfg: config.Config{Token: "x"}, saveErr: errors.New("save fail")}
	app, _, errOut = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"auth", "login", "--token", "x"}); code != 1 {
		t.Fatalf("expected save failure")
	}
	if !strings.Contains(errOut.String(), "save fail") {
		t.Fatalf("expected save fail error")
	}

	store = &fakeStore{cfg: config.Config{Token: "x"}, clearErr: errors.New("clear fail")}
	app, _, errOut = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"auth", "logout"}); code != 1 {
		t.Fatalf("expected clear failure")
	}
	if !strings.Contains(errOut.String(), "clear fail") {
		t.Fatalf("expected clear fail error")
	}
}

func TestAPICloudDashboardDatasourceCommands(t *testing.T) {
	store := &fakeStore{cfg: config.Config{Token: "token"}}
	client := &fakeClient{
		rawResult:        map[string]any{"ok": true},
		cloudResult:      map[string]any{"items": []any{map[string]any{"id": 1}}},
		searchDashResult: []any{map[string]any{"uid": "x"}},
		createDashResult: map[string]any{"status": "success"},
		listDSResult: []any{
			map[string]any{"name": "prom", "type": "prometheus"},
			map[string]any{"name": "loki", "type": "loki"},
		},
	}
	app, out, errOut := newTestApp(store, client)

	if code := app.Run(context.Background(), []string{"api", "GET", "/api/test", "--body", "{\"k\":1}"}); code != 0 {
		t.Fatalf("api should succeed: %s", errOut.String())
	}
	if decodeJSON(t, out.String())["ok"] != true {
		t.Fatalf("unexpected api output")
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"--output", "pretty", "--fields", "ok", "api", "GET", "/api/test"}); code != 0 {
		t.Fatalf("api pretty output should succeed")
	}
	prettyResp := decodeJSON(t, out.String())
	if prettyResp["ok"] != true {
		t.Fatalf("unexpected pretty fields output: %+v", prettyResp)
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"api", "GET", "/api/test", "--body", "{"}); code != 1 {
		t.Fatalf("invalid body should fail")
	}
	if code := app.Run(context.Background(), []string{"api", "GET"}); code != 1 {
		t.Fatalf("missing api args should fail")
	}
	if code := app.Run(context.Background(), []string{"api", "GET", "/api/test", "--bad"}); code != 1 {
		t.Fatalf("unknown api flag should fail")
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"cloud", "stacks", "list"}); code != 0 {
		t.Fatalf("cloud list should succeed")
	}
	if decodeJSON(t, out.String())["items"] == nil {
		t.Fatalf("unexpected cloud output")
	}
	if code := app.Run(context.Background(), []string{"cloud"}); code != 1 {
		t.Fatalf("cloud usage should fail")
	}
	if code := app.Run(context.Background(), []string{"cloud", "bad"}); code != 1 {
		t.Fatalf("unknown cloud should fail")
	}
	if code := app.Run(context.Background(), []string{"cloud", "stacks", "bad"}); code != 1 {
		t.Fatalf("cloud stacks bad verb should fail")
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"dashboards", "list", "--query", "err"}); code != 0 {
		t.Fatalf("dashboard list should succeed")
	}
	if code := app.Run(context.Background(), []string{"dashboards", "list", "--bad"}); code != 1 {
		t.Fatalf("dashboard list parse should fail")
	}
	if code := app.Run(context.Background(), []string{"dashboards", "create", "--title", "Ops"}); code != 0 {
		t.Fatalf("dashboard create should succeed")
	}
	if client.createDashboardArg["title"] != "Ops" {
		t.Fatalf("expected generated dashboard payload")
	}
	if code := app.Run(context.Background(), []string{"dashboards", "create", "--template-json", "{"}); code != 1 {
		t.Fatalf("invalid dashboard template should fail")
	}
	if code := app.Run(context.Background(), []string{"dashboards", "create", "--template-json", "{\"title\":\"FromTemplate\"}"}); code != 0 {
		t.Fatalf("valid dashboard template should succeed")
	}
	if code := app.Run(context.Background(), []string{"dashboards", "create"}); code != 1 {
		t.Fatalf("missing dashboard create flags should fail")
	}
	if code := app.Run(context.Background(), []string{"dashboards"}); code != 1 {
		t.Fatalf("missing dashboards subcommand should fail")
	}
	if code := app.Run(context.Background(), []string{"dashboards", "bad"}); code != 1 {
		t.Fatalf("unknown dashboard command should fail")
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"datasources", "list", "--type", "loki"}); code != 0 {
		t.Fatalf("datasource list should succeed")
	}
	filtered := make([]map[string]any, 0)
	if err := json.Unmarshal([]byte(out.String()), &filtered); err != nil {
		t.Fatalf("unexpected datasource JSON: %v", err)
	}
	if len(filtered) != 1 || filtered[0]["type"] != "loki" {
		t.Fatalf("unexpected datasource filter output: %+v", filtered)
	}
	if code := app.Run(context.Background(), []string{"datasources", "bad"}); code != 1 {
		t.Fatalf("invalid datasources usage should fail")
	}
	if code := app.Run(context.Background(), []string{"datasources", "list", "--bad"}); code != 1 {
		t.Fatalf("datasources list parse should fail")
	}
}

func TestDashboardFolderAnnotationAndAlertingCommands(t *testing.T) {
	store := &fakeStore{cfg: config.Config{Token: "token"}}
	client := &fakeClient{
		getDashResult:       map[string]any{"dashboard": map[string]any{"uid": "ops"}},
		deleteDashResult:    map[string]any{"status": "deleted"},
		dashVersionsResult:  []any{map[string]any{"version": 1}},
		renderDashboardResp: grafana.RenderedDashboard{Data: []byte("png-bytes"), ContentType: "image/png", Endpoint: "https://stack/render/d/ops/render", Bytes: 9},
		listFoldersResult:   []any{map[string]any{"uid": "root"}},
		getFolderResult:     map[string]any{"uid": "ops"},
		annotationsResult:   []any{map[string]any{"id": 1}},
		alertRulesResult:    []any{map[string]any{"uid": "rule-1"}},
		alertContactResult:  []any{map[string]any{"name": "pagerduty"}},
		alertPoliciesResult: map[string]any{"receiver": "default"},
	}
	app, out, _ := newTestApp(store, client)

	if code := app.Run(context.Background(), []string{"dashboards", "get", "--uid", "ops"}); code != 0 {
		t.Fatalf("dashboard get should succeed")
	}
	if decodeJSON(t, out.String())["dashboard"] == nil {
		t.Fatalf("unexpected dashboard get output")
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"dashboards", "delete", "--uid", "ops"}); code != 0 {
		t.Fatalf("dashboard delete should succeed")
	}
	if decodeJSON(t, out.String())["status"] != "deleted" {
		t.Fatalf("unexpected dashboard delete output")
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"dashboards", "versions", "--uid", "ops", "--limit", "5"}); code != 0 {
		t.Fatalf("dashboard versions should succeed")
	}
	versions := make([]map[string]any, 0)
	if err := json.Unmarshal([]byte(out.String()), &versions); err != nil {
		t.Fatalf("unexpected versions JSON: %v", err)
	}
	if len(versions) != 1 || versions[0]["version"] != float64(1) {
		t.Fatalf("unexpected dashboard versions output: %+v", versions)
	}

	renderPath := filepath.Join(t.TempDir(), "renders", "ops.png")
	out.Reset()
	if code := app.Run(context.Background(), []string{"dashboards", "render", "--uid", "ops", "--panel-id", "12", "--out", renderPath}); code != 0 {
		t.Fatalf("dashboard render should succeed")
	}
	rendered := decodeJSON(t, out.String())
	if rendered["path"] != renderPath || rendered["content_type"] != "image/png" {
		t.Fatalf("unexpected render output: %+v", rendered)
	}
	data, err := os.ReadFile(renderPath)
	if err != nil {
		t.Fatalf("expected rendered file to exist: %v", err)
	}
	if string(data) != "png-bytes" {
		t.Fatalf("unexpected rendered bytes: %q", data)
	}
	if client.renderDashboardReq.PanelID != 12 || client.renderDashboardReq.UID != "ops" || client.renderDashboardReq.Theme != "light" {
		t.Fatalf("unexpected render request: %+v", client.renderDashboardReq)
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"folders", "list"}); code != 0 {
		t.Fatalf("folders list should succeed")
	}
	folders := make([]map[string]any, 0)
	if err := json.Unmarshal([]byte(out.String()), &folders); err != nil {
		t.Fatalf("unexpected folders JSON: %v", err)
	}
	if len(folders) != 1 || folders[0]["uid"] != "root" {
		t.Fatalf("unexpected folders output: %+v", folders)
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"folders", "get", "--uid", "ops"}); code != 0 {
		t.Fatalf("folders get should succeed")
	}
	if decodeJSON(t, out.String())["uid"] != "ops" {
		t.Fatalf("unexpected folder get output")
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"annotations", "list", "--dashboard-uid", "ops", "--panel-id", "4", "--limit", "20", "--from", "now-1h", "--to", "now", "--tags", "prod,error", "--type", "annotation"}); code != 0 {
		t.Fatalf("annotations list should succeed")
	}
	annotations := make([]map[string]any, 0)
	if err := json.Unmarshal([]byte(out.String()), &annotations); err != nil {
		t.Fatalf("unexpected annotations JSON: %v", err)
	}
	if len(annotations) != 1 || annotations[0]["id"] != float64(1) {
		t.Fatalf("unexpected annotations output: %+v", annotations)
	}
	if client.annotationsReq.DashboardUID != "ops" || client.annotationsReq.PanelID != 4 || client.annotationsReq.Limit != 20 || len(client.annotationsReq.Tags) != 2 {
		t.Fatalf("unexpected annotations request: %+v", client.annotationsReq)
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"alerting", "rules", "list"}); code != 0 {
		t.Fatalf("alerting rules should succeed")
	}
	rules := make([]map[string]any, 0)
	if err := json.Unmarshal([]byte(out.String()), &rules); err != nil {
		t.Fatalf("unexpected alert rules JSON: %v", err)
	}
	if len(rules) != 1 || rules[0]["uid"] != "rule-1" {
		t.Fatalf("unexpected alert rules output: %+v", rules)
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"alerting", "contact-points", "list"}); code != 0 {
		t.Fatalf("alerting contact points should succeed")
	}
	contacts := make([]map[string]any, 0)
	if err := json.Unmarshal([]byte(out.String()), &contacts); err != nil {
		t.Fatalf("unexpected contact points JSON: %v", err)
	}
	if len(contacts) != 1 || contacts[0]["name"] != "pagerduty" {
		t.Fatalf("unexpected contact points output: %+v", contacts)
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"alerting", "policies", "get"}); code != 0 {
		t.Fatalf("alerting policies should succeed")
	}
	if decodeJSON(t, out.String())["receiver"] != "default" {
		t.Fatalf("unexpected alerting policies output")
	}

	if code := app.Run(context.Background(), []string{"dashboards", "get"}); code != 1 {
		t.Fatalf("dashboard get missing uid should fail")
	}
	if code := app.Run(context.Background(), []string{"dashboards", "delete"}); code != 1 {
		t.Fatalf("dashboard delete missing uid should fail")
	}
	if code := app.Run(context.Background(), []string{"dashboards", "versions"}); code != 1 {
		t.Fatalf("dashboard versions missing uid should fail")
	}
	if code := app.Run(context.Background(), []string{"dashboards", "render", "--uid", "ops"}); code != 1 {
		t.Fatalf("dashboard render missing out should fail")
	}
	if code := app.Run(context.Background(), []string{"dashboards", "render", "--out", renderPath}); code != 1 {
		t.Fatalf("dashboard render missing uid should fail")
	}
	if code := app.Run(context.Background(), []string{"dashboards", "get", "--bad"}); code != 1 {
		t.Fatalf("dashboard get parse should fail")
	}
	if code := app.Run(context.Background(), []string{"dashboards", "delete", "--bad"}); code != 1 {
		t.Fatalf("dashboard delete parse should fail")
	}
	if code := app.Run(context.Background(), []string{"dashboards", "versions", "--bad"}); code != 1 {
		t.Fatalf("dashboard versions parse should fail")
	}
	if code := app.Run(context.Background(), []string{"dashboards", "render", "--bad"}); code != 1 {
		t.Fatalf("dashboard render parse should fail")
	}

	badOut := filepath.Join(t.TempDir(), "parent-file")
	if err := os.WriteFile(badOut, []byte("x"), 0o600); err != nil {
		t.Fatalf("write bad out parent failed: %v", err)
	}
	if code := app.Run(context.Background(), []string{"dashboards", "render", "--uid", "ops", "--out", filepath.Join(badOut, "render.png")}); code != 1 {
		t.Fatalf("dashboard render parent file should fail")
	}

	directoryOut := filepath.Join(t.TempDir(), "render-dir")
	if err := os.MkdirAll(directoryOut, 0o755); err != nil {
		t.Fatalf("mkdir render dir failed: %v", err)
	}
	if code := app.Run(context.Background(), []string{"dashboards", "render", "--uid", "ops", "--out", directoryOut}); code != 1 {
		t.Fatalf("dashboard render directory path should fail")
	}

	if code := app.Run(context.Background(), []string{"folders"}); code != 1 {
		t.Fatalf("folders usage should fail")
	}
	if code := app.Run(context.Background(), []string{"folders", "list", "extra"}); code != 1 {
		t.Fatalf("folders list usage should fail")
	}
	if code := app.Run(context.Background(), []string{"folders", "get"}); code != 1 {
		t.Fatalf("folders get missing uid should fail")
	}
	if code := app.Run(context.Background(), []string{"folders", "get", "--bad"}); code != 1 {
		t.Fatalf("folders get parse should fail")
	}
	if code := app.Run(context.Background(), []string{"folders", "bad"}); code != 1 {
		t.Fatalf("folders unknown command should fail")
	}

	if code := app.Run(context.Background(), []string{"annotations"}); code != 1 {
		t.Fatalf("annotations usage should fail")
	}
	if code := app.Run(context.Background(), []string{"annotations", "bad"}); code != 1 {
		t.Fatalf("annotations unknown command should fail")
	}
	if code := app.Run(context.Background(), []string{"annotations", "list", "--bad"}); code != 1 {
		t.Fatalf("annotations parse should fail")
	}

	if code := app.Run(context.Background(), []string{"alerting"}); code != 1 {
		t.Fatalf("alerting usage should fail")
	}
	if code := app.Run(context.Background(), []string{"alerting", "rules", "bad"}); code != 1 {
		t.Fatalf("alerting rules usage should fail")
	}
	if code := app.Run(context.Background(), []string{"alerting", "contact-points", "bad"}); code != 1 {
		t.Fatalf("alerting contact points usage should fail")
	}
	if code := app.Run(context.Background(), []string{"alerting", "policies", "bad"}); code != 1 {
		t.Fatalf("alerting policies usage should fail")
	}
	if code := app.Run(context.Background(), []string{"alerting", "bad", "list"}); code != 1 {
		t.Fatalf("alerting unknown command should fail")
	}
}

func TestGroupHelpWithoutAuth(t *testing.T) {
	store := &fakeContextStore{
		current: "default",
		cfgs:    map[string]config.Config{"default": {}},
	}
	client := &fakeClient{}
	app, out, errOut := newTestApp(store, client)

	for _, args := range [][]string{
		{"auth", "-help"},
		{"context", "-help"},
		{"config", "-help"},
		{"cloud", "-help"},
		{"dashboards", "-help"},
		{"datasources", "-help"},
		{"folders", "-help"},
		{"annotations", "-help"},
		{"alerting", "-help"},
		{"assistant", "-help"},
		{"runtime", "-help"},
		{"aggregate", "-help"},
		{"incident", "-help"},
		{"agent", "-help"},
	} {
		out.Reset()
		errOut.Reset()
		if code := app.Run(context.Background(), args); code != 0 {
			t.Fatalf("expected help to succeed for %v, err=%s", args, errOut.String())
		}
		if _, ok := decodeJSON(t, out.String())["commands"]; !ok {
			t.Fatalf("expected commands output for %v", args)
		}
	}
}

func TestAssistantCommands(t *testing.T) {
	store := &fakeStore{cfg: config.Config{Token: "token"}}
	client := &fakeClient{
		assistantChatResult: map[string]any{"chatId": "c1"},
		assistantStatusResp: map[string]any{"status": "completed"},
		assistantSkillsResp: map[string]any{"items": []any{map[string]any{"name": "InvestigateIncident"}}},
	}
	app, out, errOut := newTestApp(store, client)

	if code := app.Run(context.Background(), []string{"assistant", "chat", "--prompt", "Investigate error rate"}); code != 0 {
		t.Fatalf("assistant chat should succeed: %s", errOut.String())
	}
	if client.assistantPrompt != "Investigate error rate" || client.assistantChatID != "" {
		t.Fatalf("assistant chat args not propagated")
	}
	if decodeJSON(t, out.String())["chatId"] != "c1" {
		t.Fatalf("unexpected assistant chat response")
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"assistant", "chat", "--prompt", "Continue", "--chat-id", "c1"}); code != 0 {
		t.Fatalf("assistant chat continuation should succeed")
	}
	if client.assistantChatID != "c1" {
		t.Fatalf("assistant chat-id not propagated")
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"assistant", "status", "--chat-id", "c1"}); code != 0 {
		t.Fatalf("assistant status should succeed")
	}
	if client.assistantStatusID != "c1" {
		t.Fatalf("assistant status chat-id not propagated")
	}
	if decodeJSON(t, out.String())["status"] != "completed" {
		t.Fatalf("unexpected assistant status response")
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"assistant", "skills"}); code != 0 {
		t.Fatalf("assistant skills should succeed")
	}
	if decodeJSON(t, out.String())["items"] == nil {
		t.Fatalf("unexpected assistant skills response")
	}

	if code := app.Run(context.Background(), []string{"assistant"}); code != 1 {
		t.Fatalf("assistant usage should fail")
	}
	if code := app.Run(context.Background(), []string{"assistant", "chat"}); code != 1 {
		t.Fatalf("assistant chat missing prompt should fail")
	}
	if code := app.Run(context.Background(), []string{"assistant", "chat", "--bad"}); code != 1 {
		t.Fatalf("assistant chat parse should fail")
	}
	if code := app.Run(context.Background(), []string{"assistant", "status"}); code != 1 {
		t.Fatalf("assistant status missing chat id should fail")
	}
	if code := app.Run(context.Background(), []string{"assistant", "status", "--bad"}); code != 1 {
		t.Fatalf("assistant status parse should fail")
	}
	if code := app.Run(context.Background(), []string{"assistant", "skills", "extra"}); code != 1 {
		t.Fatalf("assistant skills usage should fail")
	}
	if code := app.Run(context.Background(), []string{"assistant", "bad"}); code != 1 {
		t.Fatalf("assistant unknown command should fail")
	}
}

func TestRuntimeAggregateIncidentAgent(t *testing.T) {
	store := &fakeStore{cfg: config.Config{Token: "token"}}
	client := &fakeClient{
		metricsResult: map[string]any{"m": 1},
		logsResult:    map[string]any{"l": 1},
		tracesResult:  map[string]any{"t": 1},
		aggregateResult: grafana.AggregateSnapshot{
			Metrics: map[string]any{"data": map[string]any{"result": []any{1, 2}}},
			Logs:    map[string]any{"data": map[string]any{"result": []any{1}}},
			Traces:  map[string]any{"traces": []any{1, 2, 3}},
		},
		cloudResult: map[string]any{"items": []any{1, 2}},
	}
	app, out, errOut := newTestApp(store, client)

	if code := app.Run(context.Background(), []string{"runtime", "metrics", "query", "--expr", "up"}); code != 0 {
		t.Fatalf("runtime metrics failed: %s", errOut.String())
	}
	if code := app.Run(context.Background(), []string{"runtime"}); code != 1 {
		t.Fatalf("runtime usage should fail")
	}
	if code := app.Run(context.Background(), []string{"runtime", "metrics", "bad"}); code != 1 {
		t.Fatalf("runtime metrics bad verb should fail")
	}
	if code := app.Run(context.Background(), []string{"runtime", "logs", "bad"}); code != 1 {
		t.Fatalf("runtime logs bad verb should fail")
	}
	if code := app.Run(context.Background(), []string{"runtime", "traces", "bad"}); code != 1 {
		t.Fatalf("runtime traces bad verb should fail")
	}
	if code := app.Run(context.Background(), []string{"runtime", "metrics", "query", "--bad"}); code != 1 {
		t.Fatalf("runtime metrics parse should fail")
	}
	if code := app.Run(context.Background(), []string{"runtime", "logs", "query", "--bad"}); code != 1 {
		t.Fatalf("runtime logs parse should fail")
	}
	if code := app.Run(context.Background(), []string{"runtime", "traces", "search", "--bad"}); code != 1 {
		t.Fatalf("runtime traces parse should fail")
	}
	if code := app.Run(context.Background(), []string{"runtime", "logs", "query", "--query", "{}"}); code != 0 {
		t.Fatalf("runtime logs failed")
	}
	if code := app.Run(context.Background(), []string{"runtime", "traces", "search", "--query", "{}"}); code != 0 {
		t.Fatalf("runtime traces failed")
	}
	if code := app.Run(context.Background(), []string{"runtime", "metrics", "query"}); code != 1 {
		t.Fatalf("missing expr should fail")
	}
	if code := app.Run(context.Background(), []string{"runtime", "bad", "query"}); code != 1 {
		t.Fatalf("bad runtime command should fail")
	}

	if code := app.Run(context.Background(), []string{"aggregate", "snapshot", "--metric-expr", "up", "--log-query", "{}", "--trace-query", "{}"}); code != 0 {
		t.Fatalf("aggregate should succeed")
	}
	if client.aggregateReq.MetricExpr != "up" {
		t.Fatalf("aggregate request not captured")
	}
	if code := app.Run(context.Background(), []string{"aggregate", "snapshot"}); code != 1 {
		t.Fatalf("aggregate missing flags should fail")
	}
	if code := app.Run(context.Background(), []string{"aggregate"}); code != 1 {
		t.Fatalf("aggregate usage should fail")
	}
	if code := app.Run(context.Background(), []string{"aggregate", "snapshot", "--bad"}); code != 1 {
		t.Fatalf("aggregate parse should fail")
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"incident", "analyze", "--goal", "error spike"}); code != 0 {
		t.Fatalf("incident analyze should succeed: %s", errOut.String())
	}
	inc := decodeJSON(t, out.String())
	summary := inc["summary"].(map[string]any)
	if summary["metrics_series"].(float64) != 2 {
		t.Fatalf("unexpected incident summary: %+v", summary)
	}
	if code := app.Run(context.Background(), []string{"incident", "analyze"}); code != 1 {
		t.Fatalf("missing goal should fail")
	}
	if code := app.Run(context.Background(), []string{"incident", "bad"}); code != 1 {
		t.Fatalf("incident usage should fail")
	}
	if code := app.Run(context.Background(), []string{"incident", "analyze", "--goal", "slow", "--metric-expr", "m", "--log-query", "l", "--trace-query", "t", "--start", "s", "--end", "e", "--step", "1m", "--limit", "10", "--include-raw"}); code != 0 {
		t.Fatalf("incident include-raw should succeed")
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"agent", "plan", "--goal", "latency"}); code != 0 {
		t.Fatalf("agent plan should succeed")
	}
	plan := decodeJSON(t, out.String())
	if plan["playbook"] != "latency" {
		t.Fatalf("expected latency playbook")
	}
	if code := app.Run(context.Background(), []string{"agent", "run", "--goal", "errors"}); code != 0 {
		t.Fatalf("agent run should succeed")
	}
	if code := app.Run(context.Background(), []string{"agent", "run", "--goal", "errors", "--include-raw"}); code != 0 {
		t.Fatalf("agent include-raw should succeed")
	}
	if code := app.Run(context.Background(), []string{"agent", "bad", "--goal", "x"}); code != 1 {
		t.Fatalf("unknown agent command should fail")
	}
	if code := app.Run(context.Background(), []string{"agent", "plan"}); code != 1 {
		t.Fatalf("missing goal should fail")
	}
}

func TestRequireAuthAndClientErrors(t *testing.T) {
	store := &fakeStore{loadErr: errors.New("load failed")}
	client := &fakeClient{}
	app, _, errOut := newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"cloud", "stacks", "list"}); code != 1 {
		t.Fatalf("expected load failure")
	}
	if !strings.Contains(errOut.String(), "load failed") {
		t.Fatalf("unexpected load error: %s", errOut.String())
	}

	store = &fakeStore{cfg: config.Config{}}
	client = &fakeClient{}
	app, _, errOut = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"cloud", "stacks", "list"}); code != 1 {
		t.Fatalf("expected auth failure")
	}
	if !strings.Contains(errOut.String(), "not authenticated") {
		t.Fatalf("unexpected auth error: %s", errOut.String())
	}

	store = &fakeStore{cfg: config.Config{Token: "x"}}
	client = &fakeClient{cloudErr: errors.New("cloud fail")}
	app, _, errOut = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"cloud", "stacks", "list"}); code != 1 {
		t.Fatalf("expected cloud client failure")
	}
	if !strings.Contains(errOut.String(), "cloud fail") {
		t.Fatalf("unexpected cloud error: %s", errOut.String())
	}

	store = &fakeStore{cfg: config.Config{Token: "x"}}
	client = &fakeClient{
		getDashErr:         errors.New("get dash fail"),
		deleteDashErr:      errors.New("delete dash fail"),
		dashVersionsErr:    errors.New("versions fail"),
		renderDashboardErr: errors.New("render fail"),
		listFoldersErr:     errors.New("folders fail"),
		getFolderErr:       errors.New("folder fail"),
		annotationsErr:     errors.New("annotations fail"),
		alertRulesErr:      errors.New("alert rules fail"),
		alertContactErr:    errors.New("alert contact fail"),
		alertPoliciesErr:   errors.New("alert policies fail"),
	}
	app, _, _ = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"dashboards", "get", "--uid", "ops"}); code != 1 {
		t.Fatalf("expected dashboard get client failure")
	}
	if code := app.Run(context.Background(), []string{"dashboards", "delete", "--uid", "ops"}); code != 1 {
		t.Fatalf("expected dashboard delete client failure")
	}
	if code := app.Run(context.Background(), []string{"dashboards", "versions", "--uid", "ops"}); code != 1 {
		t.Fatalf("expected dashboard versions client failure")
	}
	if code := app.Run(context.Background(), []string{"dashboards", "render", "--uid", "ops", "--out", filepath.Join(t.TempDir(), "x.png")}); code != 1 {
		t.Fatalf("expected dashboard render client failure")
	}
	if code := app.Run(context.Background(), []string{"folders", "list"}); code != 1 {
		t.Fatalf("expected folders list client failure")
	}
	if code := app.Run(context.Background(), []string{"folders", "get", "--uid", "ops"}); code != 1 {
		t.Fatalf("expected folder get client failure")
	}
	if code := app.Run(context.Background(), []string{"annotations", "list"}); code != 1 {
		t.Fatalf("expected annotations client failure")
	}
	if code := app.Run(context.Background(), []string{"alerting", "rules", "list"}); code != 1 {
		t.Fatalf("expected alerting rules client failure")
	}
	if code := app.Run(context.Background(), []string{"alerting", "contact-points", "list"}); code != 1 {
		t.Fatalf("expected alerting contact points client failure")
	}
	if code := app.Run(context.Background(), []string{"alerting", "policies", "get"}); code != 1 {
		t.Fatalf("expected alerting policies client failure")
	}

	store = &fakeStore{cfg: config.Config{}}
	client = &fakeClient{}
	app, _, _ = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"folders", "list"}); code != 1 {
		t.Fatalf("expected folders auth failure")
	}
	if code := app.Run(context.Background(), []string{"annotations", "list"}); code != 1 {
		t.Fatalf("expected annotations auth failure")
	}
	if code := app.Run(context.Background(), []string{"alerting", "rules", "list"}); code != 1 {
		t.Fatalf("expected alerting auth failure")
	}
}

func TestAdditionalCommandBranches(t *testing.T) {
	store := &fakeStore{cfg: config.Config{Token: "token"}}
	client := &fakeClient{
		rawResult:        map[string]any{"ok": true},
		createDashResult: map[string]any{"status": "ok"},
		listDSResult:     []any{map[string]any{"name": "a", "type": "x"}},
		aggregateResult: grafana.AggregateSnapshot{
			Metrics: map[string]any{},
			Logs:    map[string]any{},
			Traces:  map[string]any{},
		},
		cloudResult: map[string]any{"items": []any{}},
	}
	app, _, errOut := newTestApp(store, client)

	// auth login with all optional flags.
	if code := app.Run(context.Background(), []string{
		"auth", "login",
		"--token", "abc",
		"--base-url", "https://base",
		"--cloud-url", "https://cloud",
		"--prom-url", "https://prom",
		"--logs-url", "https://logs",
		"--traces-url", "https://traces",
		"--org-id", "7",
	}); code != 0 {
		t.Fatalf("expected full auth login to succeed: %s", errOut.String())
	}

	// API should propagate client errors.
	client.rawErr = errors.New("raw fail")
	if code := app.Run(context.Background(), []string{"api", "GET", "/x"}); code != 1 {
		t.Fatalf("expected raw failure")
	}
	client.rawErr = nil

	// Dashboards create with generated optional fields.
	if code := app.Run(context.Background(), []string{"dashboards", "create", "--title", "Ops", "--uid", "uid1", "--tags", "a,b", "--folder-id", "12", "--overwrite=false"}); code != 0 {
		t.Fatalf("dashboard create optional fields should succeed")
	}
	if client.createDashboardArg["uid"] != "uid1" || client.createFolderID != 12 || client.createOverwrite != false {
		t.Fatalf("dashboard create options not propagated")
	}

	// Datasources list without filters should pass through all entries.
	if code := app.Run(context.Background(), []string{"datasources", "list"}); code != 0 {
		t.Fatalf("datasources list without filters should succeed")
	}

	// runtime required query checks.
	if code := app.Run(context.Background(), []string{"runtime", "logs", "query"}); code != 1 {
		t.Fatalf("logs query should require --query")
	}
	if code := app.Run(context.Background(), []string{"runtime", "traces", "search"}); code != 1 {
		t.Fatalf("traces search should require --query")
	}

	// incident parse error branch.
	if code := app.Run(context.Background(), []string{"incident", "analyze", "--goal", "x", "--bad"}); code != 1 {
		t.Fatalf("incident parse error expected")
	}

	// agent with no subcommand.
	if code := app.Run(context.Background(), []string{"agent"}); code != 1 {
		t.Fatalf("agent usage should fail")
	}
}

func TestAppErrorBranches(t *testing.T) {
	// runAuthLogin load error branch.
	store := &fakeStore{loadErr: errors.New("load error")}
	client := &fakeClient{}
	app, _, _ := newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"auth", "login", "--token", "x"}); code != 1 {
		t.Fatalf("expected auth login load error")
	}

	// runAPI requireAuthConfig error branch.
	store = &fakeStore{cfg: config.Config{}}
	app, _, _ = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"api", "GET", "/x"}); code != 1 {
		t.Fatalf("expected api auth error")
	}

	// runDashboards requireAuthConfig error branch.
	if code := app.Run(context.Background(), []string{"dashboards", "list"}); code != 1 {
		t.Fatalf("expected dashboards auth error")
	}

	// runDashboards list and create client error branches + create parse error.
	store = &fakeStore{cfg: config.Config{Token: "x"}}
	client = &fakeClient{searchDashErr: errors.New("search fail")}
	app, _, _ = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"dashboards", "list"}); code != 1 {
		t.Fatalf("expected dashboards list client error")
	}
	if code := app.Run(context.Background(), []string{"dashboards", "create", "--title", "x", "--bad"}); code != 1 {
		t.Fatalf("expected dashboards create parse error")
	}
	client.createDashErr = errors.New("create fail")
	if code := app.Run(context.Background(), []string{"dashboards", "create", "--title", "x"}); code != 1 {
		t.Fatalf("expected dashboards create client error")
	}

	// runDatasources auth + client error branches.
	store = &fakeStore{cfg: config.Config{}}
	client = &fakeClient{}
	app, _, _ = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"datasources", "list"}); code != 1 {
		t.Fatalf("expected datasources auth error")
	}
	store = &fakeStore{cfg: config.Config{Token: "x"}}
	client = &fakeClient{listDSErr: errors.New("ds fail")}
	app, _, _ = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"datasources", "list"}); code != 1 {
		t.Fatalf("expected datasources client error")
	}

	// runAssistant auth + client error branches.
	store = &fakeStore{cfg: config.Config{}}
	client = &fakeClient{}
	app, _, _ = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"assistant", "skills"}); code != 1 {
		t.Fatalf("expected assistant auth error")
	}
	store = &fakeStore{cfg: config.Config{Token: "x"}}
	client = &fakeClient{assistantChatErr: errors.New("assistant chat fail")}
	app, _, _ = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"assistant", "chat", "--prompt", "x"}); code != 1 {
		t.Fatalf("expected assistant chat client error")
	}
	client.assistantChatErr = nil
	client.assistantStatusErr = errors.New("assistant status fail")
	if code := app.Run(context.Background(), []string{"assistant", "status", "--chat-id", "c1"}); code != 1 {
		t.Fatalf("expected assistant status client error")
	}
	client.assistantStatusErr = nil
	client.assistantSkillsErr = errors.New("assistant skills fail")
	if code := app.Run(context.Background(), []string{"assistant", "skills"}); code != 1 {
		t.Fatalf("expected assistant skills client error")
	}

	// runRuntime auth + client error branches.
	store = &fakeStore{cfg: config.Config{}}
	client = &fakeClient{}
	app, _, _ = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"runtime", "metrics", "query", "--expr", "up"}); code != 1 {
		t.Fatalf("expected runtime auth error")
	}
	store = &fakeStore{cfg: config.Config{Token: "x"}}
	client = &fakeClient{metricsErr: errors.New("metrics fail")}
	app, _, _ = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"runtime", "metrics", "query", "--expr", "up"}); code != 1 {
		t.Fatalf("expected runtime metrics client error")
	}
	client.metricsErr = nil
	client.logsErr = errors.New("logs fail")
	if code := app.Run(context.Background(), []string{"runtime", "logs", "query", "--query", "{}"}); code != 1 {
		t.Fatalf("expected runtime logs client error")
	}
	client.logsErr = nil
	client.tracesErr = errors.New("traces fail")
	if code := app.Run(context.Background(), []string{"runtime", "traces", "search", "--query", "{}"}); code != 1 {
		t.Fatalf("expected runtime traces client error")
	}

	// runAggregate auth + aggregate error branches.
	store = &fakeStore{cfg: config.Config{}}
	client = &fakeClient{}
	app, _, _ = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"aggregate", "snapshot", "--metric-expr", "m", "--log-query", "l", "--trace-query", "t"}); code != 1 {
		t.Fatalf("expected aggregate auth error")
	}
	store = &fakeStore{cfg: config.Config{Token: "x"}}
	client = &fakeClient{aggregateErr: errors.New("aggregate fail")}
	app, _, _ = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"aggregate", "snapshot", "--metric-expr", "m", "--log-query", "l", "--trace-query", "t"}); code != 1 {
		t.Fatalf("expected aggregate client error")
	}

	// runIncident auth + aggregate error branches.
	store = &fakeStore{cfg: config.Config{}}
	client = &fakeClient{}
	app, _, _ = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"incident", "analyze", "--goal", "x"}); code != 1 {
		t.Fatalf("expected incident auth error")
	}
	store = &fakeStore{cfg: config.Config{Token: "x"}}
	client = &fakeClient{aggregateErr: errors.New("incident aggregate fail")}
	app, _, _ = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"incident", "analyze", "--goal", "x"}); code != 1 {
		t.Fatalf("expected incident aggregate error")
	}

	// runAgent parse + auth + cloud + aggregate error branches.
	store = &fakeStore{cfg: config.Config{Token: "x"}}
	client = &fakeClient{}
	app, _, _ = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"agent", "plan", "--bad"}); code != 1 {
		t.Fatalf("expected agent parse error")
	}

	store = &fakeStore{cfg: config.Config{}}
	app, _, _ = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"agent", "run", "--goal", "x"}); code != 1 {
		t.Fatalf("expected agent run auth error")
	}

	store = &fakeStore{cfg: config.Config{Token: "x"}}
	client = &fakeClient{cloudErr: errors.New("cloud fail")}
	app, _, _ = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"agent", "run", "--goal", "x"}); code != 1 {
		t.Fatalf("expected agent cloud error")
	}

	client = &fakeClient{cloudResult: map[string]any{"items": []any{}}, aggregateErr: errors.New("aggregate fail")}
	app, _, _ = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"agent", "run", "--goal", "x"}); code != 1 {
		t.Fatalf("expected agent aggregate error")
	}
}

func TestHelpers(t *testing.T) {
	if got := splitCSV("a, b,,c"); len(got) != 3 {
		t.Fatalf("unexpected splitCSV result: %+v", got)
	}

	payload := []any{
		map[string]any{"name": "Prom", "type": "prometheus"},
		map[string]any{"name": "Loki", "type": "loki"},
		"bad",
	}
	filtered := filterDatasources(payload, "loki", "lo")
	items := filtered.([]any)
	if len(items) != 1 {
		t.Fatalf("unexpected filtered datasources: %+v", filtered)
	}
	if none := filterDatasources(payload, "", "nomatch").([]any); len(none) != 0 {
		t.Fatalf("expected name-filter mismatch to drop entries")
	}
	if all := filterDatasources(payload, "", "").([]any); len(all) != 3 {
		t.Fatalf("expected unfiltered payload")
	}
	if filterDatasources(map[string]any{"x": 1}, "", "") == nil {
		t.Fatalf("non-array payload should be returned unchanged")
	}

	summary := summarizeSnapshot(grafana.AggregateSnapshot{
		Metrics: map[string]any{"data": map[string]any{"result": []any{1}}},
		Logs:    map[string]any{"data": map[string]any{"result": []any{1, 2}}},
		Traces:  map[string]any{"data": map[string]any{"traces": []any{1, 2, 3}}},
	})
	if summary["trace_matches"].(int) != 3 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	if inferCollectionCount([]any{1, 2}) != 2 {
		t.Fatalf("unexpected collection count")
	}
	if inferCollectionCount(map[string]any{"items": []any{1}}) != 1 {
		t.Fatalf("unexpected map collection count")
	}
	if inferCollectionCount("x") != 0 {
		t.Fatalf("unexpected fallback count")
	}

	if countPath(map[string]any{"a": map[string]any{"b": []any{1, 2}}}, "a", "b") != 2 {
		t.Fatalf("unexpected countPath result")
	}
	if countPath(map[string]any{"a": 1}, "a", "b") != 0 {
		t.Fatalf("countPath should return 0")
	}
	if countPath(map[string]any{"a": map[string]any{"b": 1}}, "a", "b") != 0 {
		t.Fatalf("countPath should return 0 for non-array leaf")
	}

	projected := projectFields(map[string]any{"a": map[string]any{"b": 1}, "x": 2}, []string{"a.b", "x"}).(map[string]any)
	if projected["a.b"] != 1 || projected["x"] != 2 {
		t.Fatalf("unexpected projection: %+v", projected)
	}
	projectedArr := projectFields([]any{map[string]any{"x": 1}}, []string{"x"}).([]any)
	if len(projectedArr) != 1 {
		t.Fatalf("unexpected projected array")
	}
	if passthrough := projectFields(map[string]any{"z": 1}, nil).(map[string]any); passthrough["z"] != 1 {
		t.Fatalf("expected passthrough projection")
	}
	if scalar := projectFields("value", []string{"x"}); scalar != "value" {
		t.Fatalf("expected scalar projection passthrough")
	}
	if _, ok := lookupPath(map[string]any{"a": map[string]any{"b": 1}}, []string{"a", "b"}); !ok {
		t.Fatalf("lookupPath should find value")
	}
	if _, ok := lookupPath(map[string]any{"a": map[string]any{"b": 1}}, []string{"a", "c"}); ok {
		t.Fatalf("lookupPath should fail for missing key")
	}
	if _, ok := lookupPath(map[string]any{"a": 1}, []string{"a", "b"}); ok {
		t.Fatalf("lookupPath should fail")
	}
	if maxInt(3, 2) != 3 || maxInt(1, 2) != 2 {
		t.Fatalf("unexpected max results")
	}
	if parseInt("10", 5) != 10 || parseInt("bad", 5) != 5 {
		t.Fatalf("unexpected parseInt result")
	}
}

func TestContextAndConfigCommands(t *testing.T) {
	store := &fakeContextStore{
		current: "default",
		cfgs: map[string]config.Config{
			"default": {
				Token:         "default-token",
				BaseURL:       "https://default.grafana.net",
				CloudURL:      "https://grafana.com",
				PrometheusURL: "https://prom-default.grafana.net",
				LogsURL:       "https://logs-default.grafana.net",
				TracesURL:     "https://traces-default.grafana.net",
				OrgID:         1,
			},
			"prod": {
				Token:         "prod-token",
				BaseURL:       "https://prod.grafana.net",
				CloudURL:      "https://grafana.com",
				PrometheusURL: "https://prom-prod.grafana.net",
				LogsURL:       "https://logs-prod.grafana.net",
				TracesURL:     "https://traces-prod.grafana.net",
				OrgID:         2,
			},
		},
	}
	app, out, errOut := newTestApp(store, &fakeClient{})

	if code := app.Run(context.Background(), []string{"context", "-help"}); code != 0 {
		t.Fatalf("context help should succeed: %s", errOut.String())
	}
	helpResp := decodeJSON(t, out.String())
	if _, ok := helpResp["commands"]; !ok {
		t.Fatalf("expected context command list")
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"context", "list"}); code != 0 {
		t.Fatalf("context list should succeed: %s", errOut.String())
	}
	items := decodeJSONArray(t, out.String())
	if len(items) != 2 {
		t.Fatalf("expected two contexts, got %+v", items)
	}
	foundDefault := false
	foundProd := false
	for _, item := range items {
		switch item["name"] {
		case "default":
			foundDefault = item["current"] == true
		case "prod":
			foundProd = item["authenticated"] == true
		}
	}
	if !foundDefault || !foundProd {
		t.Fatalf("unexpected context list payload: %+v", items)
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"context", "view"}); code != 0 {
		t.Fatalf("context view should succeed: %s", errOut.String())
	}
	view := decodeJSON(t, out.String())
	if view["context"] != "default" || view["base_url"] != "https://default.grafana.net" {
		t.Fatalf("unexpected context view payload: %+v", view)
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"context", "use", "prod"}); code != 0 {
		t.Fatalf("context use should succeed: %s", errOut.String())
	}
	view = decodeJSON(t, out.String())
	if view["context"] != "prod" || view["base_url"] != "https://prod.grafana.net" {
		t.Fatalf("unexpected context use payload: %+v", view)
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"config", "list"}); code != 0 {
		t.Fatalf("config list should succeed: %s", errOut.String())
	}
	listResp := decodeJSON(t, out.String())
	if listResp["context"] != "prod" || listResp["org_id"] != float64(2) {
		t.Fatalf("unexpected config list payload: %+v", listResp)
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"config", "get", "base-url"}); code != 0 {
		t.Fatalf("config get should succeed: %s", errOut.String())
	}
	getResp := decodeJSON(t, out.String())
	if getResp["context"] != "prod" || getResp["key"] != "base-url" || getResp["value"] != "https://prod.grafana.net" {
		t.Fatalf("unexpected config get payload: %+v", getResp)
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"config", "set", "base-url", "https://prod-updated.grafana.net"}); code != 0 {
		t.Fatalf("config set should succeed: %s", errOut.String())
	}
	setResp := decodeJSON(t, out.String())
	if setResp["base_url"] != "https://prod-updated.grafana.net" || store.cfgs["prod"].BaseURL != "https://prod-updated.grafana.net" {
		t.Fatalf("unexpected config set result: %+v store=%+v", setResp, store.cfgs["prod"])
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"config", "get", "--context", "default", "org-id"}); code != 0 {
		t.Fatalf("config get with context should succeed: %s", errOut.String())
	}
	getResp = decodeJSON(t, out.String())
	if getResp["context"] != "default" || getResp["value"] != float64(1) {
		t.Fatalf("unexpected config get with context payload: %+v", getResp)
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"config", "get", "base-url", "--context", "default"}); code != 0 {
		t.Fatalf("config get with trailing context should succeed: %s", errOut.String())
	}
	getResp = decodeJSON(t, out.String())
	if getResp["context"] != "default" || getResp["value"] != "https://default.grafana.net" {
		t.Fatalf("unexpected trailing context get payload: %+v", getResp)
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"config", "set", "--context", "default", "org-id", "9"}); code != 0 {
		t.Fatalf("config set with context should succeed: %s", errOut.String())
	}
	setResp = decodeJSON(t, out.String())
	if setResp["context"] != "default" || setResp["org_id"] != float64(9) || store.cfgs["default"].OrgID != 9 {
		t.Fatalf("unexpected context-specific config set payload: %+v store=%+v", setResp, store.cfgs["default"])
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"config", "set", "org-id", "10", "--context", "default"}); code != 0 {
		t.Fatalf("config set with trailing context should succeed: %s", errOut.String())
	}
	setResp = decodeJSON(t, out.String())
	if setResp["context"] != "default" || setResp["org_id"] != float64(10) || store.cfgs["default"].OrgID != 10 {
		t.Fatalf("unexpected trailing context set payload: %+v store=%+v", setResp, store.cfgs["default"])
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"auth", "login", "--context", "ops", "--token", "ops-token", "--base-url", "https://ops.grafana.net"}); code != 0 {
		t.Fatalf("auth login with context should succeed: %s", errOut.String())
	}
	loginResp := decodeJSON(t, out.String())
	if loginResp["context"] != "ops" || store.current != "ops" {
		t.Fatalf("unexpected context auth login payload: %+v current=%s", loginResp, store.current)
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"auth", "status"}); code != 0 {
		t.Fatalf("auth status should succeed: %s", errOut.String())
	}
	status := decodeJSON(t, out.String())
	if status["context"] != "ops" || status["status"] != "authenticated" {
		t.Fatalf("unexpected auth status: %+v", status)
	}
}

func TestContextAndConfigErrors(t *testing.T) {
	client := &fakeClient{}

	app, _, errOut := newTestApp(&fakeStore{}, client)
	if code := app.Run(context.Background(), []string{"context", "list"}); code != 1 {
		t.Fatalf("expected context support failure")
	}
	if !strings.Contains(errOut.String(), "context support is unavailable") {
		t.Fatalf("expected context support error, got %s", errOut.String())
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"auth", "login", "--context", "prod", "--token", "x"}); code != 1 {
		t.Fatalf("expected auth context failure")
	}
	if !strings.Contains(errOut.String(), "context support is unavailable") {
		t.Fatalf("expected auth context support error, got %s", errOut.String())
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"config", "get", "base-url", "--context", "prod"}); code != 1 {
		t.Fatalf("expected config context failure with trailing flag order")
	}
	if !strings.Contains(errOut.String(), "context support is unavailable") {
		t.Fatalf("expected trailing context support error, got %s", errOut.String())
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"config", "get", "--context", "prod", "base-url"}); code != 1 {
		t.Fatalf("expected config context failure")
	}
	if !strings.Contains(errOut.String(), "context support is unavailable") {
		t.Fatalf("expected config context support error, got %s", errOut.String())
	}

	for _, args := range [][]string{
		{"config", "list", "--context"},
		{"config", "get", "--context"},
		{"config", "set", "--context"},
	} {
		errOut.Reset()
		if code := app.Run(context.Background(), args); code != 1 {
			t.Fatalf("expected missing context value failure for %v", args)
		}
		if !strings.Contains(errOut.String(), "--context requires a value") {
			t.Fatalf("expected missing context value error for %v, got %s", args, errOut.String())
		}
	}

	store := &fakeContextStore{cfgs: map[string]config.Config{"default": {Token: "token"}}}
	app, _, errOut = newTestApp(store, client)
	if code := app.Run(context.Background(), []string{"context", "use", "prod"}); code != 1 {
		t.Fatalf("expected missing context failure")
	}
	if !strings.Contains(errOut.String(), "context not found") {
		t.Fatalf("expected missing context error, got %s", errOut.String())
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"context", "use"}); code != 1 {
		t.Fatalf("expected usage failure")
	}
	if !strings.Contains(errOut.String(), "usage: context use <NAME>") {
		t.Fatalf("expected context use usage error, got %s", errOut.String())
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"context", "view", "prod"}); code != 1 {
		t.Fatalf("expected context view usage failure")
	}
	if !strings.Contains(errOut.String(), "usage: context view") {
		t.Fatalf("expected context view usage error, got %s", errOut.String())
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"context", "bad"}); code != 1 {
		t.Fatalf("expected unknown context command failure")
	}
	if !strings.Contains(errOut.String(), "unknown context command") {
		t.Fatalf("expected unknown context command error, got %s", errOut.String())
	}

	store.listErr = errors.New("list fail")
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"context", "list"}); code != 1 {
		t.Fatalf("expected list error")
	}
	if !strings.Contains(errOut.String(), "list fail") {
		t.Fatalf("expected list error, got %s", errOut.String())
	}

	store.listErr = nil
	store.loadErr = errors.New("load fail")
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"context", "view"}); code != 1 {
		t.Fatalf("expected view load failure")
	}
	if !strings.Contains(errOut.String(), "load fail") {
		t.Fatalf("expected view load error, got %s", errOut.String())
	}

	store.loadErr = nil
	store.useErr = errors.New("use fail")
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"context", "use", "default"}); code != 1 {
		t.Fatalf("expected use error")
	}
	if !strings.Contains(errOut.String(), "use fail") {
		t.Fatalf("expected use error, got %s", errOut.String())
	}

	store.useErr = nil
	store.loadErr = errors.New("reload fail")
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"context", "use", "default"}); code != 1 {
		t.Fatalf("expected reload failure")
	}
	if !strings.Contains(errOut.String(), "reload fail") {
		t.Fatalf("expected reload error after context use, got %s", errOut.String())
	}

	store.loadErr = nil
	store.loadCtxErr = errors.New("load context fail")
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"config", "list", "--context", "default"}); code != 1 {
		t.Fatalf("expected config load context failure")
	}
	if !strings.Contains(errOut.String(), "load context fail") {
		t.Fatalf("expected config load context error, got %s", errOut.String())
	}

	store.loadCtxErr = nil
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"config", "list", "--context", "default", "extra"}); code != 1 {
		t.Fatalf("expected config list usage failure")
	}
	if !strings.Contains(errOut.String(), "usage: config list") {
		t.Fatalf("expected config list usage error, got %s", errOut.String())
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"config", "get"}); code != 1 {
		t.Fatalf("expected config get usage failure")
	}
	if !strings.Contains(errOut.String(), "usage: config get") {
		t.Fatalf("expected config get usage error, got %s", errOut.String())
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"config", "set"}); code != 1 {
		t.Fatalf("expected config set usage failure")
	}
	if !strings.Contains(errOut.String(), "usage: config set") {
		t.Fatalf("expected config set usage error, got %s", errOut.String())
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"config", "bad"}); code != 1 {
		t.Fatalf("expected unknown config command failure")
	}
	if !strings.Contains(errOut.String(), "unknown config command") {
		t.Fatalf("expected unknown config command error, got %s", errOut.String())
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"config", "get", "bad-key"}); code != 1 {
		t.Fatalf("expected unknown config get key failure")
	}
	if !strings.Contains(errOut.String(), "unknown config key") {
		t.Fatalf("expected unknown config get key error, got %s", errOut.String())
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"config", "set", "bad-key", "value"}); code != 1 {
		t.Fatalf("expected unknown config key failure")
	}
	if !strings.Contains(errOut.String(), "unknown config key") {
		t.Fatalf("expected unknown config key error, got %s", errOut.String())
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"config", "set", "org-id", "bad"}); code != 1 {
		t.Fatalf("expected invalid org id failure")
	}
	if !strings.Contains(errOut.String(), "invalid org-id") {
		t.Fatalf("expected invalid org-id error, got %s", errOut.String())
	}

	store.saveCtxErr = errors.New("save context fail")
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"auth", "login", "--context", "ops", "--token", "x"}); code != 1 {
		t.Fatalf("expected auth context save failure")
	}
	if !strings.Contains(errOut.String(), "save context fail") {
		t.Fatalf("expected auth context save error, got %s", errOut.String())
	}

	store.saveCtxErr = nil
	store.saveErr = errors.New("save fail")
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"config", "set", "base-url", "https://x"}); code != 1 {
		t.Fatalf("expected config save failure")
	}
	if !strings.Contains(errOut.String(), "save fail") {
		t.Fatalf("expected config save error, got %s", errOut.String())
	}

	store.saveErr = nil
	store.loadCtxErr = errors.New("load context fail for set")
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"config", "set", "--context", "default", "base-url", "https://x"}); code != 1 {
		t.Fatalf("expected config context load failure")
	}
	if !strings.Contains(errOut.String(), "load context fail for set") {
		t.Fatalf("expected config context load error, got %s", errOut.String())
	}

	store.loadCtxErr = nil
	store.saveCtxErr = errors.New("save context fail")
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"config", "set", "--context", "default", "base-url", "https://x"}); code != 1 {
		t.Fatalf("expected config context save failure")
	}
	if !strings.Contains(errOut.String(), "save context fail") {
		t.Fatalf("expected config context save error, got %s", errOut.String())
	}
}

func TestOutputFormattingAndContextHelpers(t *testing.T) {
	store := &fakeContextStore{
		current: "default",
		cfgs: map[string]config.Config{
			"default": {
				Token:         "token",
				BaseURL:       "https://default.grafana.net",
				CloudURL:      "https://grafana.com",
				PrometheusURL: "https://prom.grafana.net",
				LogsURL:       "https://logs.grafana.net",
				TracesURL:     "https://traces.grafana.net",
				TokenBackend:  "keyring",
				OrgID:         7,
			},
		},
	}
	app, out, errOut := newTestApp(store, &fakeClient{})

	if code := app.Run(context.Background(), []string{"--output", "pretty", "context", "view"}); code != 0 {
		t.Fatalf("pretty output should succeed: %s", errOut.String())
	}
	if !strings.Contains(out.String(), "\n  \"context\": \"default\"") {
		t.Fatalf("expected indented pretty output, got %s", out.String())
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"--json", "context,base_url", "context", "view"}); code != 0 {
		t.Fatalf("json field projection should succeed: %s", errOut.String())
	}
	projected := decodeJSON(t, out.String())
	if len(projected) != 2 || projected["context"] != "default" || projected["base_url"] != "https://default.grafana.net" {
		t.Fatalf("unexpected field projection: %+v", projected)
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"--jq", ".base_url", "context", "view"}); code != 0 {
		t.Fatalf("jq output should succeed: %s", errOut.String())
	}
	if out.String() != "https://default.grafana.net\n" {
		t.Fatalf("unexpected jq scalar output: %q", out.String())
	}

	out.Reset()
	if code := app.Run(context.Background(), []string{"--template", "{{.context}} {{.org_id}}", "context", "view"}); code != 0 {
		t.Fatalf("template output should succeed: %s", errOut.String())
	}
	if out.String() != "default 7\n" {
		t.Fatalf("unexpected template output: %q", out.String())
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"--jq", ".[", "context", "view"}); code != 1 {
		t.Fatalf("expected jq parse failure")
	}
	if errOut.Len() == 0 {
		t.Fatalf("expected jq parse error output")
	}

	if selectedContextName(nil, "") != "default" {
		t.Fatalf("expected default context fallback")
	}
	if selectedContextName(store, "explicit") != "explicit" {
		t.Fatalf("expected explicit context name")
	}
	if selectedContextName(&fakeContextStore{currentErr: errors.New("boom")}, "") != "default" {
		t.Fatalf("expected default context on current-context error")
	}

	cfg, name, err := app.loadConfigForContext("")
	if err != nil || name != "default" || cfg.BaseURL != "https://default.grafana.net" {
		t.Fatalf("unexpected load current context result: cfg=%+v name=%s err=%v", cfg, name, err)
	}
	cfg, name, err = app.loadConfigForContext("default")
	if err != nil || name != "default" || cfg.BaseURL != "https://default.grafana.net" {
		t.Fatalf("unexpected load explicit context result: cfg=%+v name=%s err=%v", cfg, name, err)
	}

	payload := configPayload("default", store.cfgs["default"])
	if payload["token_backend"] != "keyring" {
		t.Fatalf("unexpected config payload: %+v", payload)
	}

	if normalizeConfigKey(" Base-URL ") != "base-url" {
		t.Fatalf("unexpected normalized key")
	}

	valueCfg := config.Config{
		BaseURL:       "base",
		CloudURL:      "cloud",
		PrometheusURL: "prom",
		LogsURL:       "logs",
		TracesURL:     "traces",
		TokenBackend:  "keyring",
		OrgID:         5,
	}
	for key, want := range map[string]any{
		"base-url":       "base",
		"base_url":       "base",
		"cloud-url":      "cloud",
		"prom-url":       "prom",
		"prometheus_url": "prom",
		"logs-url":       "logs",
		"traces_url":     "traces",
		"org-id":         int64(5),
		"token-backend":  "keyring",
	} {
		got, err := configValueForKey(valueCfg, key)
		if err != nil || got != want {
			t.Fatalf("unexpected config value for %s: got=%v err=%v", key, got, err)
		}
	}
	if _, err := configValueForKey(valueCfg, "bad"); err == nil {
		t.Fatalf("expected unknown config key error")
	}

	mutable := config.Config{}
	updates := []struct {
		key   string
		value string
		check func(config.Config) bool
	}{
		{key: "base-url", value: "base", check: func(cfg config.Config) bool { return cfg.BaseURL == "base" }},
		{key: "cloud-url", value: "cloud", check: func(cfg config.Config) bool { return cfg.CloudURL == "cloud" }},
		{key: "prom-url", value: "prom", check: func(cfg config.Config) bool { return cfg.PrometheusURL == "prom" }},
		{key: "logs-url", value: "logs", check: func(cfg config.Config) bool { return cfg.LogsURL == "logs" }},
		{key: "traces-url", value: "traces", check: func(cfg config.Config) bool { return cfg.TracesURL == "traces" }},
		{key: "org-id", value: "11", check: func(cfg config.Config) bool { return cfg.OrgID == 11 }},
	}
	for _, update := range updates {
		if err := setConfigValue(&mutable, update.key, update.value); err != nil || !update.check(mutable) {
			t.Fatalf("unexpected config update for %s: cfg=%+v err=%v", update.key, mutable, err)
		}
	}
	if err := setConfigValue(&mutable, "org-id", "-1"); err == nil {
		t.Fatalf("expected invalid negative org id")
	}

	values, err := applyJQ([]any{map[string]any{"x": 1}, map[string]any{"x": 2}}, ".[].x")
	if err != nil {
		t.Fatalf("unexpected jq multi-result error: %v", err)
	}
	results, ok := values.([]any)
	if !ok || len(results) != 2 {
		t.Fatalf("unexpected jq multi-result payload: %#v", values)
	}

	value, err := applyJQ(map[string]any{"x": 1}, ".x")
	if err != nil || value != 1 {
		t.Fatalf("unexpected jq single result: value=%v err=%v", value, err)
	}
	value, err = applyJQ(map[string]any{"x": 1}, "empty")
	if err != nil || value != nil {
		t.Fatalf("unexpected jq empty result: value=%v err=%v", value, err)
	}
	if _, err := applyJQ(map[string]any{"x": 1}, ".["); err == nil {
		t.Fatalf("expected jq parse error")
	}

	builder := &strings.Builder{}
	if err := renderTemplate(builder, map[string]any{"name": "grafana"}, "{{.name}}"); err != nil {
		t.Fatalf("unexpected template render error: %v", err)
	}
	if builder.String() != "grafana\n" {
		t.Fatalf("unexpected template render output: %q", builder.String())
	}
	builder.Reset()
	if err := renderTemplate(builder, map[string]any{"name": "grafana"}, "{{json .}}"); err != nil {
		t.Fatalf("unexpected template json render error: %v", err)
	}
	if builder.String() != "{\"name\":\"grafana\"}\n" {
		t.Fatalf("unexpected template json output: %q", builder.String())
	}
	if err := renderTemplate(&strings.Builder{}, map[string]any{}, "{{"); err == nil {
		t.Fatalf("expected template parse error")
	}
	if err := renderTemplate(&strings.Builder{}, map[string]any{"bad": make(chan int)}, "{{json .bad}}"); err == nil {
		t.Fatalf("expected template execution error")
	}

	if !isScalar(nil) || !isScalar(true) || !isScalar(json.Number("7")) {
		t.Fatalf("expected scalar values to be detected")
	}
	if isScalar([]any{1}) {
		t.Fatalf("expected array to be non-scalar")
	}
	if scalarString(nil) != "null" || scalarString(7) != "7" {
		t.Fatalf("unexpected scalar string conversion")
	}
}

func TestContextConfigAndOutputEdgeBranches(t *testing.T) {
	store := &fakeContextStore{
		current: "default",
		cfgs: map[string]config.Config{
			"default": {Token: "token", BaseURL: "https://default.grafana.net", CloudURL: "https://grafana.com"},
		},
	}
	app, _, errOut := newTestApp(store, &fakeClient{})

	if code := app.Run(context.Background(), []string{"context", "list", "extra"}); code != 1 {
		t.Fatalf("expected context list usage failure")
	}
	if !strings.Contains(errOut.String(), "usage: context list") {
		t.Fatalf("expected context list usage error, got %s", errOut.String())
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"config"}); code != 0 {
		t.Fatalf("expected config summary to succeed")
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"config", "get", "--context"}); code != 1 {
		t.Fatalf("expected missing context value failure")
	}
	if !strings.Contains(errOut.String(), "--context requires a value") {
		t.Fatalf("expected missing context value error, got %s", errOut.String())
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"config", "get", "--context=default", "base-url"}); code != 0 {
		t.Fatalf("expected inline context arg to succeed: %s", errOut.String())
	}

	if _, _, err := parseGlobalOptions([]string{"--json=context", "--jq=.context", "context", "view"}); err != nil {
		t.Fatalf("expected inline json/jq options to succeed: %v", err)
	}

	loadFailApp := &App{Store: &fakeStore{loadErr: errors.New("load fail")}}
	if _, _, err := loadFailApp.loadConfigForContext(""); err == nil {
		t.Fatalf("expected loadConfigForContext store failure")
	}

	if _, err := applyJQ(map[string]any{"x": "value"}, ".x + 1"); err == nil {
		t.Fatalf("expected jq runtime error")
	}

	builder := &strings.Builder{}
	if err := renderTemplate(builder, map[string]any{"name": "grafana"}, "{{.name}}\n"); err != nil {
		t.Fatalf("unexpected template render with trailing newline error: %v", err)
	}
	if builder.String() != "grafana\n" {
		t.Fatalf("unexpected template output with trailing newline: %q", builder.String())
	}
	builder.Reset()
	if err := renderTemplate(builder, map[string]any{"name": "grafana"}, "{{json .}}"); err != nil {
		t.Fatalf("unexpected template json render error: %v", err)
	}
	if builder.String() != "{\"name\":\"grafana\"}\n" {
		t.Fatalf("unexpected template json output: %q", builder.String())
	}
}
