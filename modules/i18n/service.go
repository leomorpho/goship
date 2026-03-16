package i18n

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

const (
	defaultLanguage = "en"
)

type contextLanguageKey struct{}

type Options struct {
	LocaleDir       string
	DefaultLanguage string
}

type Service struct {
	bundle          *goi18n.Bundle
	defaultLanguage string
	supported       map[string]struct{}

	mu         sync.RWMutex
	localizers map[string]*goi18n.Localizer
}

func NewService(opts Options) (*Service, error) {
	localeDir := strings.TrimSpace(opts.LocaleDir)
	if localeDir == "" {
		return nil, errors.New("locale directory is required")
	}

	defaultLang := normalizeLanguageTag(opts.DefaultLanguage)
	if defaultLang == "" {
		defaultLang = defaultLanguage
	}

	entries, err := os.ReadDir(localeDir)
	if err != nil {
		return nil, fmt.Errorf("read locale directory: %w", err)
	}

	bundle := goi18n.NewBundle(language.Make(defaultLang))
	supported := map[string]struct{}{}

	type localeFile struct {
		name       string
		path       string
		localeCode string
		priority   int
	}
	files := make([]localeFile, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		priority := 0
		switch ext {
		case ".toml":
			priority = 2
		case ".yaml", ".yml":
			priority = 1
		default:
			continue
		}
		localeCode := normalizeLanguageTag(strings.TrimSuffix(entry.Name(), ext))
		if localeCode == "" {
			continue
		}
		files = append(files, localeFile{
			name:       entry.Name(),
			path:       filepath.Join(localeDir, entry.Name()),
			localeCode: localeCode,
			priority:   priority,
		})
	}
	sort.Slice(files, func(i, j int) bool {
		left := files[i]
		right := files[j]
		if left.localeCode != right.localeCode {
			return left.localeCode < right.localeCode
		}
		if left.priority != right.priority {
			return left.priority < right.priority
		}
		return left.name < right.name
	})

	localeMaps := map[string]map[string]string{}
	for _, file := range files {
		raw, readErr := os.ReadFile(file.path)
		if readErr != nil {
			return nil, fmt.Errorf("read locale file %s: %w", file.name, readErr)
		}

		ext := strings.ToLower(filepath.Ext(file.name))
		var flat map[string]string
		var parseErr error
		switch ext {
		case ".toml":
			flat, parseErr = flattenLocaleTOML(raw)
		default:
			flat, parseErr = flattenLocaleYAML(raw)
		}
		if parseErr != nil {
			return nil, fmt.Errorf("parse locale file %s: %w", file.name, parseErr)
		}

		if _, ok := localeMaps[file.localeCode]; !ok {
			localeMaps[file.localeCode] = map[string]string{}
		}
		for key, value := range flat {
			localeMaps[file.localeCode][key] = value
		}
	}

	localeCodes := make([]string, 0, len(localeMaps))
	for localeCode := range localeMaps {
		localeCodes = append(localeCodes, localeCode)
	}
	sort.Strings(localeCodes)
	for _, localeCode := range localeCodes {
		flat := localeMaps[localeCode]
		messages := make([]*goi18n.Message, 0, len(flat))
		keys := make([]string, 0, len(flat))
		for key := range flat {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			messages = append(messages, &goi18n.Message{
				ID:    key,
				Other: flat[key],
			})
		}
		if len(messages) > 0 {
			if err := bundle.AddMessages(language.Make(localeCode), messages...); err != nil {
				return nil, fmt.Errorf("register locale %s: %w", localeCode, err)
			}
		}
		supported[localeCode] = struct{}{}
	}

	if len(supported) == 0 {
		return nil, fmt.Errorf("no locale files found in %s", localeDir)
	}
	if _, ok := supported[defaultLang]; !ok {
		return nil, fmt.Errorf("default locale %q is not available", defaultLang)
	}

	return &Service{
		bundle:          bundle,
		defaultLanguage: defaultLang,
		supported:       supported,
		localizers:      map[string]*goi18n.Localizer{},
	}, nil
}

func (s *Service) DefaultLanguage() string {
	if s == nil {
		return defaultLanguage
	}
	return s.defaultLanguage
}

func (s *Service) SupportedLanguages() []string {
	if s == nil {
		return nil
	}
	values := make([]string, 0, len(s.supported))
	for code := range s.supported {
		values = append(values, code)
	}
	sort.Strings(values)
	return values
}

func (s *Service) NormalizeLanguage(raw string) string {
	if s == nil {
		return defaultLanguage
	}
	lang := normalizeLanguageTag(raw)
	if lang == "" {
		return s.defaultLanguage
	}
	if _, ok := s.supported[lang]; ok {
		return lang
	}
	if idx := strings.Index(lang, "-"); idx > 0 {
		base := lang[:idx]
		if _, ok := s.supported[base]; ok {
			return base
		}
	}
	return s.defaultLanguage
}

func (s *Service) T(ctx context.Context, key string, templateData ...map[string]any) string {
	if s == nil {
		return key
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}

	lang := s.NormalizeLanguage(LanguageFromContext(ctx))
	localizer := s.localizer(lang)
	cfg := &goi18n.LocalizeConfig{MessageID: key}
	if len(templateData) > 0 {
		cfg.TemplateData = templateData[0]
	}

	msg, err := localizer.Localize(cfg)
	if err != nil || strings.TrimSpace(msg) == "" {
		return key
	}
	return msg
}

func (s *Service) localizer(lang string) *goi18n.Localizer {
	s.mu.RLock()
	if existing, ok := s.localizers[lang]; ok {
		s.mu.RUnlock()
		return existing
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.localizers[lang]; ok {
		return existing
	}
	created := goi18n.NewLocalizer(s.bundle, lang, s.defaultLanguage)
	s.localizers[lang] = created
	return created
}

func WithLanguage(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, contextLanguageKey{}, normalizeLanguageTag(lang))
}

func LanguageFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(contextLanguageKey{}).(string)
	return normalizeLanguageTag(value)
}

func flattenLocaleYAML(data []byte) (map[string]string, error) {
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	out := map[string]string{}
	flattenLocaleMap("", raw, out)
	return out, nil
}

func flattenLocaleTOML(data []byte) (map[string]string, error) {
	var raw map[string]any
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return nil, err
	}
	out := map[string]string{}
	flattenLocaleMap("", raw, out)
	return out, nil
}

func flattenLocaleMap(prefix string, value any, out map[string]string) {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			nextPrefix := key
			if prefix != "" {
				nextPrefix = prefix + "." + key
			}
			flattenLocaleMap(nextPrefix, typed[key], out)
		}
	case string:
		if prefix != "" {
			out[prefix] = strings.TrimSpace(typed)
		}
	case nil:
		if prefix != "" {
			out[prefix] = ""
		}
	default:
		if prefix != "" {
			out[prefix] = strings.TrimSpace(fmt.Sprint(typed))
		}
	}
}

func normalizeLanguageTag(value string) string {
	clean := strings.ToLower(strings.TrimSpace(value))
	clean = strings.ReplaceAll(clean, "_", "-")
	if idx := strings.Index(clean, ","); idx >= 0 {
		clean = strings.TrimSpace(clean[:idx])
	}
	if idx := strings.Index(clean, ";"); idx >= 0 {
		clean = strings.TrimSpace(clean[:idx])
	}
	return clean
}
