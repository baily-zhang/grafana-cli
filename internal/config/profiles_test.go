package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func fileOnlySecretFactory(configPath string) SecretStore {
	return &chainSecretStore{
		backends: []secretBackend{
			&fileSecretBackend{path: filepath.Join(filepath.Dir(configPath), "token")},
		},
	}
}

type hookSecretStore struct {
	loadErr  error
	saveErr  error
	clearErr error
	onClear  func()
}

func (s *hookSecretStore) Load() (string, string, error) {
	return "", "", s.loadErr
}

func (s *hookSecretStore) Save(string) (string, error) {
	return "hook", s.saveErr
}

func (s *hookSecretStore) Clear() error {
	if s.onClear != nil {
		s.onClear()
	}
	return s.clearErr
}

func TestProfileStoreConstructors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	store := NewProfileStore(path)
	if store.Path() != path {
		t.Fatalf("unexpected profile store path: %s", store.Path())
	}
	if store.secretStores == nil {
		t.Fatalf("expected default secret store factory")
	}

	factory := newDefaultSecretStoreFactory()
	if factory == nil {
		t.Fatalf("expected default secret store factory")
	}
	if factory(path) == nil {
		t.Fatalf("expected factory to build a secret store")
	}

	store = NewProfileStoreWithSecretFactory(path, nil)
	if store.secretStores == nil {
		t.Fatalf("expected nil factory to fall back to default")
	}
}

func TestProfileStoreLifecycle(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.json")
	store := NewProfileStoreWithSecretFactory(path, fileOnlySecretFactory)

	if current, err := store.CurrentContext(); err != nil || current != defaultContextName {
		t.Fatalf("unexpected initial current context: current=%q err=%v", current, err)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("unexpected initial load error: %v", err)
	}
	if cfg.BaseURL != defaultBaseURL || cfg.CloudURL != defaultCloudURL || cfg.Token != "" {
		t.Fatalf("unexpected initial config: %+v", cfg)
	}

	contexts, err := store.ListContexts()
	if err != nil {
		t.Fatalf("unexpected initial list error: %v", err)
	}
	if len(contexts) != 1 || contexts[0].Name != defaultContextName || !contexts[0].Current || contexts[0].Authenticated {
		t.Fatalf("unexpected initial contexts: %+v", contexts)
	}

	defaultCfg := Config{
		BaseURL:       "https://default.grafana.net",
		CloudURL:      "https://grafana.com",
		PrometheusURL: "https://prom-default.grafana.net",
		LogsURL:       "https://logs-default.grafana.net",
		TracesURL:     "https://traces-default.grafana.net",
		Token:         "default-token",
		OrgID:         1,
	}
	if err := store.Save(defaultCfg); err != nil {
		t.Fatalf("unexpected default save error: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("unexpected default reload error: %v", err)
	}
	if loaded.Token != "default-token" || loaded.TokenBackend != "file" || loaded.BaseURL != defaultCfg.BaseURL {
		t.Fatalf("unexpected default reload payload: %+v", loaded)
	}

	prodCfg := Config{
		BaseURL:       "https://prod.grafana.net",
		CloudURL:      "https://grafana.com",
		PrometheusURL: "https://prom-prod.grafana.net",
		LogsURL:       "https://logs-prod.grafana.net",
		TracesURL:     "https://traces-prod.grafana.net",
		Token:         "prod-token",
		OrgID:         2,
	}
	if err := store.SaveContext("prod", prodCfg); err != nil {
		t.Fatalf("unexpected prod save error: %v", err)
	}

	contexts, err = store.ListContexts()
	if err != nil {
		t.Fatalf("unexpected context list error: %v", err)
	}
	if len(contexts) != 2 || contexts[0].Name != defaultContextName || contexts[1].Name != "prod" || !contexts[1].Current {
		t.Fatalf("unexpected context summaries: %+v", contexts)
	}

	if err := store.UseContext(defaultContextName); err != nil {
		t.Fatalf("unexpected default context switch error: %v", err)
	}
	loaded, err = store.Load()
	if err != nil {
		t.Fatalf("unexpected default context load error: %v", err)
	}
	if loaded.Token != "default-token" {
		t.Fatalf("unexpected default context token: %+v", loaded)
	}

	prodLoaded, err := store.LoadContext("prod")
	if err != nil {
		t.Fatalf("unexpected prod load error: %v", err)
	}
	if prodLoaded.Token != "prod-token" || prodLoaded.TokenBackend != "file" {
		t.Fatalf("unexpected prod context payload: %+v", prodLoaded)
	}

	if err := store.Clear(); err != nil {
		t.Fatalf("unexpected clear error for default context: %v", err)
	}
	if current, err := store.CurrentContext(); err != nil || current != "prod" {
		t.Fatalf("unexpected current context after default clear: current=%q err=%v", current, err)
	}

	loaded, err = store.Load()
	if err != nil {
		t.Fatalf("unexpected load after default clear: %v", err)
	}
	if loaded.Token != "prod-token" || loaded.BaseURL != prodCfg.BaseURL {
		t.Fatalf("unexpected active context after default clear: %+v", loaded)
	}

	if err := store.Clear(); err != nil {
		t.Fatalf("unexpected clear error for final context: %v", err)
	}
	if current, err := store.CurrentContext(); err != nil || current != defaultContextName {
		t.Fatalf("unexpected current context after final clear: current=%q err=%v", current, err)
	}

	loaded, err = store.Load()
	if err != nil {
		t.Fatalf("unexpected load after final clear: %v", err)
	}
	if loaded.Token != "" || loaded.BaseURL != defaultBaseURL || loaded.CloudURL != defaultCloudURL {
		t.Fatalf("unexpected config after final clear: %+v", loaded)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected manifest to be removed after final clear")
	}
}

func TestProfileStoreMigrationAndHelpers(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.json")
	store := NewProfileStoreWithSecretFactory(path, fileOnlySecretFactory)

	if err := os.WriteFile(path, []byte(`{"base_url":"https://legacy.grafana.net","cloud_url":"https://grafana.com","token":"legacy-token"}`), 0o600); err != nil {
		t.Fatalf("write legacy config failed: %v", err)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("unexpected legacy migration error: %v", err)
	}
	if cfg.Token != "legacy-token" || cfg.TokenBackend != "file" || cfg.BaseURL != "https://legacy.grafana.net" {
		t.Fatalf("unexpected migrated config: %+v", cfg)
	}

	manifestBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest failed: %v", err)
	}
	if strings.Contains(string(manifestBytes), "legacy-token") || !strings.Contains(string(manifestBytes), "current_context") {
		t.Fatalf("unexpected manifest contents: %s", manifestBytes)
	}

	contextBytes, err := os.ReadFile(filepath.Join(tmp, "contexts", defaultContextName, "config.json"))
	if err != nil {
		t.Fatalf("read migrated context config failed: %v", err)
	}
	if strings.Contains(string(contextBytes), "legacy-token") {
		t.Fatalf("migrated context config should not store token: %s", contextBytes)
	}

	tokenBytes, err := os.ReadFile(filepath.Join(tmp, "contexts", defaultContextName, "token"))
	if err != nil {
		t.Fatalf("read migrated token failed: %v", err)
	}
	if strings.TrimSpace(string(tokenBytes)) != "legacy-token" {
		t.Fatalf("unexpected migrated token payload: %q", tokenBytes)
	}
	if _, err := os.Stat(filepath.Join(tmp, "token")); !os.IsNotExist(err) {
		t.Fatalf("expected legacy root token file to be removed")
	}

	emptyPath := filepath.Join(tmp, "empty.json")
	emptyStore := NewProfileStoreWithSecretFactory(emptyPath, fileOnlySecretFactory)
	if err := os.WriteFile(emptyPath, []byte{}, 0o600); err != nil {
		t.Fatalf("write empty manifest failed: %v", err)
	}
	if current, err := emptyStore.CurrentContext(); err != nil || current != defaultContextName {
		t.Fatalf("unexpected empty-manifest current context: current=%q err=%v", current, err)
	}
	if err := emptyStore.writeManifest(profileManifest{}); err != nil {
		t.Fatalf("unexpected blank manifest write error: %v", err)
	}
	if current, err := emptyStore.CurrentContext(); err != nil || current != defaultContextName {
		t.Fatalf("unexpected blank manifest defaulting: current=%q err=%v", current, err)
	}

	if _, err := normalizeContextName(""); err == nil {
		t.Fatalf("expected empty context name error")
	}
	if _, err := normalizeContextName("prod/us"); err == nil {
		t.Fatalf("expected invalid context name error")
	}
	if err := emptyStore.UseContext("missing"); err == nil {
		t.Fatalf("expected missing context error")
	}
	if _, err := emptyStore.LoadContext("prod/us"); err == nil {
		t.Fatalf("expected invalid load context name error")
	}
	if err := emptyStore.SaveContext("prod/us", Config{}); err == nil {
		t.Fatalf("expected invalid save context name error")
	}

	badManifestPath := filepath.Join(tmp, "bad.json")
	if err := os.WriteFile(badManifestPath, []byte("{"), 0o600); err != nil {
		t.Fatalf("write bad manifest failed: %v", err)
	}
	badManifestStore := NewProfileStoreWithSecretFactory(badManifestPath, fileOnlySecretFactory)
	if _, err := badManifestStore.CurrentContext(); err == nil {
		t.Fatalf("expected bad manifest parse error")
	}

	errorRootPath := filepath.Join(tmp, "error-parent", "config.json")
	if err := os.WriteFile(filepath.Dir(errorRootPath), []byte("x"), 0o600); err != nil {
		t.Fatalf("write parent file failed: %v", err)
	}
	errorStore := NewProfileStoreWithSecretFactory(errorRootPath, fileOnlySecretFactory)
	if err := errorStore.writeManifest(profileManifest{CurrentContext: defaultContextName}); err == nil {
		t.Fatalf("expected manifest write error when parent is a file")
	}

	discoverPath := filepath.Join(tmp, "discover.json")
	discoverStore := NewProfileStoreWithSecretFactory(discoverPath, fileOnlySecretFactory)
	if err := discoverStore.writeManifest(profileManifest{CurrentContext: defaultContextName}); err != nil {
		t.Fatalf("unexpected discover manifest write error: %v", err)
	}
	discoverRoot := filepath.Join(filepath.Dir(discoverPath), "contexts")
	if err := os.RemoveAll(discoverRoot); err != nil {
		t.Fatalf("remove discover root failed: %v", err)
	}
	if err := os.WriteFile(discoverRoot, []byte("x"), 0o600); err != nil {
		t.Fatalf("write discover root file failed: %v", err)
	}
	if _, err := discoverStore.discoverContexts(); err == nil {
		t.Fatalf("expected discoverContexts error when root is a file")
	}
}

func TestProfileStoreAdditionalErrorBranches(t *testing.T) {
	brokenPath := filepath.Join(t.TempDir(), "broken.json")
	if err := os.WriteFile(brokenPath, []byte("{"), 0o600); err != nil {
		t.Fatalf("write broken manifest failed: %v", err)
	}
	brokenStore := NewProfileStoreWithSecretFactory(brokenPath, fileOnlySecretFactory)
	if _, err := brokenStore.Load(); err == nil {
		t.Fatalf("expected load failure for broken manifest")
	}
	if err := brokenStore.Save(Config{}); err == nil {
		t.Fatalf("expected save failure for broken manifest")
	}
	if err := brokenStore.Clear(); err == nil {
		t.Fatalf("expected clear failure for broken manifest")
	}
	if _, err := brokenStore.ListContexts(); err == nil {
		t.Fatalf("expected list contexts failure for broken manifest")
	}

	loadContextErrPath := filepath.Join(t.TempDir(), "load-context.json")
	loadContextErrStore := NewProfileStoreWithSecretFactory(loadContextErrPath, fileOnlySecretFactory)
	if err := loadContextErrStore.writeManifest(profileManifest{CurrentContext: defaultContextName}); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}
	badConfigPath := filepath.Join(filepath.Dir(loadContextErrPath), "contexts", defaultContextName, "config.json")
	if err := os.MkdirAll(filepath.Join(badConfigPath, "child"), 0o700); err != nil {
		t.Fatalf("mkdir bad config path failed: %v", err)
	}
	if _, err := loadContextErrStore.ListContexts(); err == nil {
		t.Fatalf("expected list contexts error for directory config path")
	}

	if err := os.WriteFile(loadContextErrPath, []byte(`{"current_context":1}`), 0o600); err != nil {
		t.Fatalf("write invalid typed manifest failed: %v", err)
	}
	if _, _, err := loadContextErrStore.readManifest(); err == nil {
		t.Fatalf("expected typed manifest unmarshal error")
	}

	filterPath := filepath.Join(t.TempDir(), "filter.json")
	filterStore := NewProfileStoreWithSecretFactory(filterPath, fileOnlySecretFactory)
	filterRoot := filepath.Join(filepath.Dir(filterPath), "contexts")
	if err := os.MkdirAll(filterRoot, 0o700); err != nil {
		t.Fatalf("mkdir filter root failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(filterRoot, "notes.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write file entry failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(filterRoot, "bad name"), 0o700); err != nil {
		t.Fatalf("mkdir invalid context failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(filterRoot, "bad name", "config.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write invalid context config failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(filterRoot, "missing-config"), 0o700); err != nil {
		t.Fatalf("mkdir missing config context failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(filterRoot, "prod"), 0o700); err != nil {
		t.Fatalf("mkdir valid context failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(filterRoot, "prod", "config.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write valid context config failed: %v", err)
	}
	names, err := filterStore.discoverContexts()
	if err != nil {
		t.Fatalf("unexpected filtered discover error: %v", err)
	}
	if !reflect.DeepEqual(names, []string{"prod"}) {
		t.Fatalf("unexpected filtered contexts: %+v", names)
	}
}

func TestProfileStoreMigrateLegacyConfigErrors(t *testing.T) {
	loadErrStore := NewProfileStoreWithSecretFactory(filepath.Join(t.TempDir(), "config.json"), fileOnlySecretFactory)
	if err := os.MkdirAll(loadErrStore.Path(), 0o700); err != nil {
		t.Fatalf("mkdir load error path failed: %v", err)
	}
	if err := loadErrStore.migrateLegacyConfig(); err == nil {
		t.Fatalf("expected migrateLegacyConfig load failure")
	}

	saveErrPath := filepath.Join(t.TempDir(), "config.json")
	saveErrStore := NewProfileStoreWithSecretFactory(saveErrPath, fileOnlySecretFactory)
	if err := os.WriteFile(saveErrPath, []byte(`{"token":"legacy-token"}`), 0o600); err != nil {
		t.Fatalf("write legacy config failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(saveErrPath), "contexts"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocking contexts file failed: %v", err)
	}
	if err := saveErrStore.migrateLegacyConfig(); err == nil {
		t.Fatalf("expected migrateLegacyConfig save failure")
	}

	clearErrStore := NewProfileStoreWithSecretFactory(filepath.Join(t.TempDir(), "config.json"), func(string) SecretStore {
		return &stubSecretStore{clearErr: errors.New("clear fail")}
	})
	if err := os.WriteFile(clearErrStore.Path(), []byte(`{"token":"legacy-token"}`), 0o600); err != nil {
		t.Fatalf("write legacy config failed: %v", err)
	}
	if err := clearErrStore.migrateLegacyConfig(); err == nil {
		t.Fatalf("expected migrateLegacyConfig clear failure")
	}
}

func TestProfileStoreErrorBranches(t *testing.T) {
	badManifestPath := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(badManifestPath, []byte("{"), 0o600); err != nil {
		t.Fatalf("write broken manifest failed: %v", err)
	}
	badManifestStore := NewProfileStoreWithSecretFactory(badManifestPath, fileOnlySecretFactory)
	if _, err := badManifestStore.Load(); err == nil {
		t.Fatalf("expected load failure for broken manifest")
	}
	if err := badManifestStore.Save(Config{}); err == nil {
		t.Fatalf("expected save failure for broken manifest")
	}
	if err := badManifestStore.Clear(); err == nil {
		t.Fatalf("expected clear failure for broken manifest")
	}
	if _, err := badManifestStore.ListContexts(); err == nil {
		t.Fatalf("expected list contexts failure for broken manifest")
	}

	dirManifestPath := filepath.Join(t.TempDir(), "manifest-dir")
	if err := os.MkdirAll(dirManifestPath, 0o700); err != nil {
		t.Fatalf("mkdir manifest dir failed: %v", err)
	}
	dirManifestStore := NewProfileStoreWithSecretFactory(dirManifestPath, fileOnlySecretFactory)
	if _, _, err := dirManifestStore.readManifest(); err == nil {
		t.Fatalf("expected readManifest directory error")
	}

	typedManifestPath := filepath.Join(t.TempDir(), "typed.json")
	if err := os.WriteFile(typedManifestPath, []byte(`{"current_context":1}`), 0o600); err != nil {
		t.Fatalf("write typed manifest failed: %v", err)
	}
	typedManifestStore := NewProfileStoreWithSecretFactory(typedManifestPath, fileOnlySecretFactory)
	if _, _, err := typedManifestStore.readManifest(); err == nil {
		t.Fatalf("expected typed manifest error")
	}

	blankManifestPath := filepath.Join(t.TempDir(), "blank.json")
	if err := os.WriteFile(blankManifestPath, []byte(`{"current_context":"   "}`), 0o600); err != nil {
		t.Fatalf("write blank manifest failed: %v", err)
	}
	blankManifestStore := NewProfileStoreWithSecretFactory(blankManifestPath, fileOnlySecretFactory)
	if current, err := blankManifestStore.CurrentContext(); err != nil || current != defaultContextName {
		t.Fatalf("unexpected blank-manifest current context: current=%q err=%v", current, err)
	}
	if err := blankManifestStore.UseContext("bad/name"); err == nil {
		t.Fatalf("expected invalid use context name error")
	}

	saveErrorPath := filepath.Join(t.TempDir(), "save-error", "config.json")
	if err := os.WriteFile(filepath.Dir(saveErrorPath), []byte("x"), 0o600); err != nil {
		t.Fatalf("write save-error parent file failed: %v", err)
	}
	saveErrorStore := NewProfileStoreWithSecretFactory(saveErrorPath, fileOnlySecretFactory)
	if err := saveErrorStore.SaveContext("prod", Config{}); err == nil {
		t.Fatalf("expected save context failure when parent is a file")
	}

	listLoadErrPath := filepath.Join(t.TempDir(), "list-load-error.json")
	listLoadErrStore := NewProfileStoreWithSecretFactory(listLoadErrPath, fileOnlySecretFactory)
	if err := listLoadErrStore.writeManifest(profileManifest{CurrentContext: defaultContextName}); err != nil {
		t.Fatalf("write list load manifest failed: %v", err)
	}
	listBadConfigPath := filepath.Join(filepath.Dir(listLoadErrPath), "contexts", defaultContextName, "config.json")
	if err := os.MkdirAll(filepath.Join(listBadConfigPath, "child"), 0o700); err != nil {
		t.Fatalf("mkdir list bad config path failed: %v", err)
	}
	if _, err := listLoadErrStore.ListContexts(); err == nil {
		t.Fatalf("expected list contexts load error")
	}

	listDiscoverErrPath := filepath.Join(t.TempDir(), "list-discover-error.json")
	listDiscoverErrStore := NewProfileStoreWithSecretFactory(listDiscoverErrPath, fileOnlySecretFactory)
	if err := listDiscoverErrStore.writeManifest(profileManifest{CurrentContext: defaultContextName}); err != nil {
		t.Fatalf("write list discover manifest failed: %v", err)
	}
	listDiscoverRoot := filepath.Join(filepath.Dir(listDiscoverErrPath), "contexts")
	if err := os.RemoveAll(listDiscoverRoot); err != nil {
		t.Fatalf("remove list discover root failed: %v", err)
	}
	if err := os.WriteFile(listDiscoverRoot, []byte("x"), 0o600); err != nil {
		t.Fatalf("write list discover root failed: %v", err)
	}
	if _, err := listDiscoverErrStore.ListContexts(); err == nil {
		t.Fatalf("expected list contexts discover error")
	}

	clearConfigErrPath := filepath.Join(t.TempDir(), "clear-config-error.json")
	clearConfigErrStore := NewProfileStoreWithSecretFactory(clearConfigErrPath, fileOnlySecretFactory)
	if err := clearConfigErrStore.writeManifest(profileManifest{CurrentContext: defaultContextName}); err != nil {
		t.Fatalf("write clear config manifest failed: %v", err)
	}
	clearBadConfigPath := filepath.Join(filepath.Dir(clearConfigErrPath), "contexts", defaultContextName, "config.json")
	if err := os.MkdirAll(clearBadConfigPath, 0o700); err != nil {
		t.Fatalf("mkdir clear bad config path failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(clearBadConfigPath, "child"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write clear bad config child failed: %v", err)
	}
	if err := clearConfigErrStore.Clear(); err == nil {
		t.Fatalf("expected clear config path error")
	}

	clearDiscoverErrRoot := t.TempDir()
	clearDiscoverErrPath := filepath.Join(clearDiscoverErrRoot, "clear-discover-error.json")
	clearDiscoverErrStore := NewProfileStoreWithSecretFactory(clearDiscoverErrPath, func(configPath string) SecretStore {
		return &hookSecretStore{
			onClear: func() {
				if configPath != filepath.Join(clearDiscoverErrRoot, "contexts", defaultContextName, "config.json") {
					return
				}
				root := filepath.Join(clearDiscoverErrRoot, "contexts")
				if err := os.RemoveAll(root); err != nil {
					t.Fatalf("remove clear discover root failed: %v", err)
				}
				if err := os.WriteFile(root, []byte("x"), 0o600); err != nil {
					t.Fatalf("write clear discover root failed: %v", err)
				}
			},
		}
	})
	if err := clearDiscoverErrStore.Save(Config{Token: "token"}); err != nil {
		t.Fatalf("save clear discover config failed: %v", err)
	}
	if err := clearDiscoverErrStore.Clear(); err == nil {
		t.Fatalf("expected clear discover error")
	}

	clearRemoveErrRoot := t.TempDir()
	clearRemoveErrPath := filepath.Join(clearRemoveErrRoot, "config.json")
	clearRemoveErrStore := NewProfileStoreWithSecretFactory(clearRemoveErrPath, func(configPath string) SecretStore {
		return &hookSecretStore{
			onClear: func() {
				if configPath != filepath.Join(clearRemoveErrRoot, "contexts", defaultContextName, "config.json") {
					return
				}
				if err := os.Remove(clearRemoveErrPath); err != nil && !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("remove clear remove manifest failed: %v", err)
				}
				if err := os.MkdirAll(filepath.Join(clearRemoveErrPath, "child"), 0o700); err != nil {
					t.Fatalf("mkdir clear remove manifest dir failed: %v", err)
				}
			},
		}
	})
	if err := clearRemoveErrStore.Save(Config{Token: "token"}); err != nil {
		t.Fatalf("save clear remove config failed: %v", err)
	}
	if err := clearRemoveErrStore.Clear(); err == nil {
		t.Fatalf("expected clear remove error")
	}

	legacyCurrentErrPath := filepath.Join(t.TempDir(), "legacy-current.json")
	if err := os.WriteFile(legacyCurrentErrPath, []byte(`{"token":"legacy-token"}`), 0o600); err != nil {
		t.Fatalf("write legacy current config failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(legacyCurrentErrPath), "contexts"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write legacy blocking contexts file failed: %v", err)
	}
	legacyCurrentErrStore := NewProfileStoreWithSecretFactory(legacyCurrentErrPath, fileOnlySecretFactory)
	if _, err := legacyCurrentErrStore.currentContextName(); err == nil {
		t.Fatalf("expected currentContextName migration error")
	}
}

func TestProfileStoreCoverageBranches(t *testing.T) {
	if name, err := normalizeContextName("Prod1"); err != nil || name != "Prod1" {
		t.Fatalf("expected uppercase and digit context name to succeed: %q err=%v", name, err)
	}

	invalidUseStore := NewProfileStoreWithSecretFactory(filepath.Join(t.TempDir(), "config.json"), fileOnlySecretFactory)
	if err := invalidUseStore.UseContext("bad/name"); err == nil {
		t.Fatalf("expected invalid context name error for UseContext")
	}

	readDirStore := NewProfileStoreWithSecretFactory(filepath.Join(t.TempDir(), "config.json"), fileOnlySecretFactory)
	if err := os.MkdirAll(readDirStore.Path(), 0o700); err != nil {
		t.Fatalf("mkdir read dir store failed: %v", err)
	}
	if _, _, err := readDirStore.readManifest(); err == nil {
		t.Fatalf("expected readManifest file read error")
	}

	blankManifestPath := filepath.Join(t.TempDir(), "blank.json")
	if err := os.WriteFile(blankManifestPath, []byte(`{"current_context":"   "}`), 0o600); err != nil {
		t.Fatalf("write blank manifest failed: %v", err)
	}
	blankManifestStore := NewProfileStoreWithSecretFactory(blankManifestPath, fileOnlySecretFactory)
	manifest, legacy, err := blankManifestStore.readManifest()
	if err != nil || legacy || manifest.CurrentContext != defaultContextName {
		t.Fatalf("unexpected blank manifest read: manifest=%+v legacy=%v err=%v", manifest, legacy, err)
	}

	listDiscoverErrPath := filepath.Join(t.TempDir(), "discover-error.json")
	listDiscoverErrStore := NewProfileStoreWithSecretFactory(listDiscoverErrPath, fileOnlySecretFactory)
	if err := listDiscoverErrStore.writeManifest(profileManifest{CurrentContext: defaultContextName}); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}
	discoverRoot := filepath.Join(filepath.Dir(listDiscoverErrPath), "contexts")
	if err := os.WriteFile(discoverRoot, []byte("x"), 0o600); err != nil {
		t.Fatalf("write discover root file failed: %v", err)
	}
	if _, err := listDiscoverErrStore.ListContexts(); err == nil {
		t.Fatalf("expected list contexts discover error")
	}

	saveErrStore := NewProfileStoreWithSecretFactory(filepath.Join(t.TempDir(), "save-error", "config.json"), fileOnlySecretFactory)
	if err := os.MkdirAll(filepath.Dir(saveErrStore.Path()), 0o700); err != nil {
		t.Fatalf("mkdir save error store failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(saveErrStore.Path()), "contexts"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write contexts blocker failed: %v", err)
	}
	if err := saveErrStore.SaveContext("prod", Config{Token: "token"}); err == nil {
		t.Fatalf("expected save context error")
	}

	legacyErrPath := filepath.Join(t.TempDir(), "legacy-error.json")
	if err := os.WriteFile(legacyErrPath, []byte(`{"token":"legacy-token"}`), 0o600); err != nil {
		t.Fatalf("write legacy config failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(legacyErrPath), "contexts"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write legacy contexts blocker failed: %v", err)
	}
	legacyErrStore := NewProfileStoreWithSecretFactory(legacyErrPath, fileOnlySecretFactory)
	if _, err := legacyErrStore.currentContextName(); err == nil {
		t.Fatalf("expected currentContextName migration error")
	}

	clearContextErrPath := filepath.Join(t.TempDir(), "clear-context", "config.json")
	clearContextErrStore := NewProfileStoreWithSecretFactory(clearContextErrPath, fileOnlySecretFactory)
	if err := clearContextErrStore.writeManifest(profileManifest{CurrentContext: defaultContextName}); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}
	clearContextConfigPath := filepath.Join(filepath.Dir(clearContextErrPath), "contexts", defaultContextName, "config.json")
	if err := os.MkdirAll(filepath.Join(clearContextConfigPath, "child"), 0o700); err != nil {
		t.Fatalf("mkdir clear context config path failed: %v", err)
	}
	if err := clearContextErrStore.Clear(); err == nil {
		t.Fatalf("expected clear context-store error")
	}

	clearDiscoverPath := filepath.Join(t.TempDir(), "clear-discover", "config.json")
	clearDiscoverStore := NewProfileStoreWithSecretFactory(clearDiscoverPath, func(string) SecretStore {
		return &stubSecretStore{
			clearHook: func() {
				root := filepath.Join(filepath.Dir(clearDiscoverPath), "contexts")
				if err := os.RemoveAll(root); err != nil {
					t.Fatalf("remove contexts root failed: %v", err)
				}
				if err := os.WriteFile(root, []byte("x"), 0o600); err != nil {
					t.Fatalf("write contexts root file failed: %v", err)
				}
			},
		}
	})
	if err := clearDiscoverStore.Save(Config{Token: "token"}); err != nil {
		t.Fatalf("save clear-discover config failed: %v", err)
	}
	if err := clearDiscoverStore.Clear(); err == nil {
		t.Fatalf("expected clear discoverContexts error")
	}

	clearRemovePath := filepath.Join(t.TempDir(), "clear-remove", "config.json")
	clearRemoveStore := NewProfileStoreWithSecretFactory(clearRemovePath, func(string) SecretStore {
		return &stubSecretStore{
			clearHook: func() {
				if err := os.Remove(clearRemovePath); err != nil && !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("remove manifest failed: %v", err)
				}
				if err := os.MkdirAll(filepath.Join(clearRemovePath, "child"), 0o700); err != nil {
					t.Fatalf("mkdir manifest directory failed: %v", err)
				}
			},
		}
	})
	if err := clearRemoveStore.Save(Config{Token: "token"}); err != nil {
		t.Fatalf("save clear-remove config failed: %v", err)
	}
	if err := clearRemoveStore.Clear(); err == nil {
		t.Fatalf("expected clear manifest removal error")
	}
}
