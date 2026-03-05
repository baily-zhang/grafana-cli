package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const defaultContextName = "default"

type profileManifest struct {
	CurrentContext string `json:"current_context"`
}

type ContextSummary struct {
	Name          string `json:"name"`
	Current       bool   `json:"current"`
	Authenticated bool   `json:"authenticated"`
	BaseURL       string `json:"base_url"`
	CloudURL      string `json:"cloud_url"`
}

type ContextStore interface {
	Store
	CurrentContext() (string, error)
	ListContexts() ([]ContextSummary, error)
	UseContext(name string) error
	LoadContext(name string) (Config, error)
	SaveContext(name string, cfg Config) error
}

type ProfileStore struct {
	path         string
	secretStores secretStoreFactory
}

func NewProfileStore(path string) *ProfileStore {
	return NewProfileStoreWithSecretFactory(path, newDefaultSecretStoreFactory())
}

func NewProfileStoreWithSecretFactory(path string, factory secretStoreFactory) *ProfileStore {
	if factory == nil {
		factory = newDefaultSecretStoreFactory()
	}
	return &ProfileStore{path: path, secretStores: factory}
}

func (s *ProfileStore) Path() string {
	return s.path
}

func (s *ProfileStore) Load() (Config, error) {
	name, err := s.currentContextName()
	if err != nil {
		return Config{}, err
	}
	return s.LoadContext(name)
}

func (s *ProfileStore) Save(cfg Config) error {
	name, err := s.currentContextName()
	if err != nil {
		return err
	}
	return s.SaveContext(name, cfg)
}

func (s *ProfileStore) Clear() error {
	name, err := s.currentContextName()
	if err != nil {
		return err
	}
	if err := s.contextStore(name).Clear(); err != nil {
		return err
	}
	contexts, err := s.discoverContexts()
	if err != nil {
		return err
	}
	if len(contexts) == 0 {
		err := os.Remove(s.path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	next := contexts[0]
	for _, candidate := range contexts {
		if candidate != name {
			next = candidate
			break
		}
	}
	return s.writeManifest(profileManifest{CurrentContext: next})
}

func (s *ProfileStore) CurrentContext() (string, error) {
	return s.currentContextName()
}

func (s *ProfileStore) ListContexts() ([]ContextSummary, error) {
	current, err := s.currentContextName()
	if err != nil {
		return nil, err
	}
	names, err := s.discoverContexts()
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		names = []string{current}
	}
	summaries := make([]ContextSummary, 0, len(names))
	for _, name := range names {
		cfg, err := s.LoadContext(name)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, ContextSummary{
			Name:          name,
			Current:       name == current,
			Authenticated: cfg.IsAuthenticated(),
			BaseURL:       cfg.BaseURL,
			CloudURL:      cfg.CloudURL,
		})
	}
	return summaries, nil
}

func (s *ProfileStore) UseContext(name string) error {
	name, err := normalizeContextName(name)
	if err != nil {
		return err
	}
	if !s.contextExists(name) && name != defaultContextName {
		return errors.New("context not found")
	}
	return s.writeManifest(profileManifest{CurrentContext: name})
}

func (s *ProfileStore) LoadContext(name string) (Config, error) {
	name, err := normalizeContextName(name)
	if err != nil {
		return Config{}, err
	}
	return s.contextStore(name).Load()
}

func (s *ProfileStore) SaveContext(name string, cfg Config) error {
	name, err := normalizeContextName(name)
	if err != nil {
		return err
	}
	if err := s.contextStore(name).Save(cfg); err != nil {
		return err
	}
	return s.writeManifest(profileManifest{CurrentContext: name})
}

func (s *ProfileStore) currentContextName() (string, error) {
	manifest, legacy, err := s.readManifest()
	if err != nil {
		return "", err
	}
	if legacy {
		if err := s.migrateLegacyConfig(); err != nil {
			return "", err
		}
		return defaultContextName, nil
	}
	return manifest.CurrentContext, nil
}

func (s *ProfileStore) readManifest() (profileManifest, bool, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return profileManifest{CurrentContext: defaultContextName}, false, nil
		}
		return profileManifest{}, false, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return profileManifest{CurrentContext: defaultContextName}, false, nil
	}
	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return profileManifest{}, false, err
	}
	if _, ok := raw["current_context"]; !ok {
		return profileManifest{}, true, nil
	}
	manifest := profileManifest{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return profileManifest{}, false, err
	}
	if strings.TrimSpace(manifest.CurrentContext) == "" {
		manifest.CurrentContext = defaultContextName
	}
	return manifest, false, nil
}

func (s *ProfileStore) writeManifest(manifest profileManifest) error {
	if strings.TrimSpace(manifest.CurrentContext) == "" {
		manifest.CurrentContext = defaultContextName
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(manifest, "", "  ")
	return os.WriteFile(s.path, data, 0o600)
}

func (s *ProfileStore) migrateLegacyConfig() error {
	legacy := NewFileStoreWithSecretStore(s.path, s.secretStores(s.path))
	cfg, err := legacy.Load()
	if err != nil {
		return err
	}
	if err := s.contextStore(defaultContextName).Save(cfg); err != nil {
		return err
	}
	if err := legacy.Clear(); err != nil {
		return err
	}
	return s.writeManifest(profileManifest{CurrentContext: defaultContextName})
}

func (s *ProfileStore) contextStore(name string) *FileStore {
	path := filepath.Join(filepath.Dir(s.path), "contexts", name, "config.json")
	return NewFileStoreWithSecretStore(path, s.secretStores(path))
}

func (s *ProfileStore) contextExists(name string) bool {
	_, err := os.Stat(s.contextStore(name).Path())
	return err == nil
}

func (s *ProfileStore) discoverContexts() ([]string, error) {
	root := filepath.Join(filepath.Dir(s.path), "contexts")
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name, err := normalizeContextName(entry.Name())
		if err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, name, "config.json")); err == nil {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func normalizeContextName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("context name is required")
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.':
		default:
			return "", errors.New("invalid context name")
		}
	}
	return name, nil
}
