package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/leomorpho/goship/config"
	coreadapters "github.com/leomorpho/goship/framework/core/adapters"
	"github.com/leomorpho/goship/framework/runtimeplan"
)

type AdapterDeps struct {
	Out        io.Writer
	Err        io.Writer
	LoadConfig func() (config.Config, error)
}

type adapterPreset struct {
	key      string
	envKeys  []string
	apply    func(*config.Config, string) error
	validate func(*config.Config) error
}

var adapterPresets = map[string]adapterPreset{
	"db": {
		key:     "db",
		envKeys: []string{"PAGODA_ADAPTERS_DB", "PAGODA_DATABASE_DRIVER", "PAGODA_DB_DRIVER", "PAGODA_DATABASE_DBMODE"},
		apply: func(cfg *config.Config, value string) error {
			switch strings.ToLower(strings.TrimSpace(value)) {
			case "postgres", "postgresql", "pgx":
				cfg.Adapters.DB = "postgres"
				cfg.Database.Driver = config.DBDriverPostgres
				cfg.Database.DbMode = config.DBModeStandalone
			case "sqlite", "sqlite3":
				cfg.Adapters.DB = "sqlite"
				cfg.Database.Driver = config.DBDriverSQLite
				cfg.Database.DbMode = config.DBModeEmbedded
			default:
				return fmt.Errorf("unsupported db adapter %q (allowed: postgres, sqlite)", value)
			}
			return nil
		},
	},
	"cache": {
		key:     "cache",
		envKeys: []string{"PAGODA_ADAPTERS_CACHE", "PAGODA_CACHE_DRIVER"},
		apply: func(cfg *config.Config, value string) error {
			cfg.Adapters.Cache = strings.ToLower(strings.TrimSpace(value))
			return nil
		},
	},
	"jobs": {
		key:     "jobs",
		envKeys: []string{"PAGODA_ADAPTERS_JOBS", "PAGODA_JOBS_DRIVER"},
		apply: func(cfg *config.Config, value string) error {
			cfg.Adapters.Jobs = strings.ToLower(strings.TrimSpace(value))
			return nil
		},
	},
	"pubsub": {
		key:     "pubsub",
		envKeys: []string{"PAGODA_ADAPTERS_PUBSUB"},
		apply: func(cfg *config.Config, value string) error {
			cfg.Adapters.PubSub = strings.ToLower(strings.TrimSpace(value))
			return nil
		},
	},
	"storage": {
		key:     "storage",
		envKeys: []string{"PAGODA_STORAGE_DRIVER"},
		apply: func(cfg *config.Config, value string) error {
			switch strings.ToLower(strings.TrimSpace(value)) {
			case string(config.StorageDriverLocal):
				cfg.Storage.Driver = config.StorageDriverLocal
			case string(config.StorageDriverMinIO):
				cfg.Storage.Driver = config.StorageDriverMinIO
			default:
				return fmt.Errorf("unsupported storage adapter %q (allowed: local, minio)", value)
			}
			return nil
		},
	},
	"mailer": {
		key:     "mailer",
		envKeys: []string{"PAGODA_MAIL_DRIVER", "MAIL_DRIVER"},
		apply: func(cfg *config.Config, value string) error {
			cfg.Mail.Driver = strings.ToLower(strings.TrimSpace(value))
			return nil
		},
	},
}

var adapterOrder = []string{"db", "cache", "jobs", "pubsub", "storage", "mailer"}

func RunAdapter(args []string, d AdapterDeps) int {
	if len(args) == 0 {
		PrintAdapterHelp(d.Out)
		return 0
	}

	switch args[0] {
	case "set":
		return runAdapterSet(args[1:], d)
	case "help", "-h", "--help":
		PrintAdapterHelp(d.Out)
		return 0
	default:
		fmt.Fprintf(d.Err, "unknown adapter command: %s\n\n", args[0])
		PrintAdapterHelp(d.Err)
		return 1
	}
}

func PrintAdapterHelp(w io.Writer) {
	fmt.Fprintln(w, "ship adapter commands:")
	fmt.Fprintln(w, "  ship adapter:set <db|cache|jobs|pubsub|storage|mailer>=<impl>...  Rewrite canonical adapter env vars with validation")
}

func runAdapterSet(args []string, d AdapterDeps) int {
	if len(args) == 0 {
		fmt.Fprintln(d.Err, "usage: ship adapter:set <db|cache|jobs|pubsub|storage|mailer>=<impl>...")
		return 1
	}
	if hasHelpArg(args) {
		PrintAdapterHelp(d.Out)
		return 0
	}
	if d.LoadConfig == nil {
		fmt.Fprintln(d.Err, "adapter:set requires config loader dependency")
		return 1
	}

	desired, err := parseAdapterAssignments(args)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid adapter:set arguments: %v\n", err)
		return 1
	}

	cfg, err := d.LoadConfig()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to load config: %v\n", err)
		return 1
	}

	next := cfg
	for _, key := range adapterOrder {
		value, ok := desired[key]
		if !ok {
			continue
		}
		spec := adapterPresets[key]
		if err := spec.apply(&next, value); err != nil {
			fmt.Fprintf(d.Err, "%v\n", err)
			return 1
		}
	}

	if err := validateAdapterSelection(&next); err != nil {
		fmt.Fprintf(d.Err, "invalid adapter selection: %v\n", err)
		return 1
	}
	if err := validateRuntimeCombination(&next); err != nil {
		fmt.Fprintf(d.Err, "invalid runtime combination: %v\n", err)
		return 1
	}

	envPath, err := findEnvFile(".")
	if err != nil {
		fmt.Fprintf(d.Err, "adapter:set requires a .env file: %v\n", err)
		return 1
	}
	original, err := os.ReadFile(envPath)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to read %s: %v\n", envPath, err)
		return 1
	}

	updated, changed, err := rewriteEnvAssignments(string(original), adapterEnvValues(desired), adapterEnvOrder(desired))
	if err != nil {
		fmt.Fprintf(d.Err, "failed to update %s: %v\n", envPath, err)
		return 1
	}
	if !changed {
		fmt.Fprintf(d.Out, "adapter selection already applied in %s\n", envPath)
		return 0
	}

	if err := os.WriteFile(envPath, []byte(updated), 0o644); err != nil {
		fmt.Fprintf(d.Err, "failed to write %s: %v\n", envPath, err)
		return 1
	}

	fmt.Fprintf(d.Out, "adapter selection applied in %s\n", envPath)
	for _, key := range adapterOrder {
		value, ok := desired[key]
		if !ok {
			continue
		}
		fmt.Fprintf(d.Out, "- %s=%s\n", key, value)
	}
	return 0
}

func hasHelpArg(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "help", "-h", "--help":
			return true
		}
	}
	return false
}

func parseAdapterAssignments(args []string) (map[string]string, error) {
	desired := map[string]string{}
	for _, arg := range args {
		key, value, ok := strings.Cut(arg, "=")
		if !ok {
			return nil, fmt.Errorf("expected key=value, got %q", arg)
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		spec, ok := adapterPresets[key]
		if !ok {
			return nil, fmt.Errorf("unknown adapter kind %q (expected one of %s)", key, strings.Join(adapterOrder, ", "))
		}
		if value == "" {
			return nil, fmt.Errorf("missing value for %s", key)
		}
		desired[spec.key] = value
	}
	return desired, nil
}

func adapterEnvValues(desired map[string]string) map[string]string {
	values := map[string]string{}
	for _, key := range adapterOrder {
		value, ok := desired[key]
		if !ok {
			continue
		}
		for _, envKey := range adapterPresets[key].envKeys {
			values[envKey] = normalizedAdapterEnvValue(key, envKey, value)
		}
	}
	return values
}

func adapterEnvOrder(desired map[string]string) []string {
	order := make([]string, 0)
	seen := map[string]struct{}{}
	for _, key := range adapterOrder {
		if _, ok := desired[key]; !ok {
			continue
		}
		for _, envKey := range adapterPresets[key].envKeys {
			if _, ok := seen[envKey]; ok {
				continue
			}
			seen[envKey] = struct{}{}
			order = append(order, envKey)
		}
	}
	return order
}

func normalizedAdapterEnvValue(kind, envKey, raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch kind {
	case "db":
		switch envKey {
		case "PAGODA_DATABASE_DBMODE":
			switch value {
			case "postgres", "postgresql", "pgx":
				return string(config.DBModeStandalone)
			case "sqlite", "sqlite3":
				return string(config.DBModeEmbedded)
			}
		default:
			switch value {
			case "postgres", "postgresql", "pgx":
				return "postgres"
			case "sqlite", "sqlite3":
				return "sqlite"
			}
		}
	case "cache", "jobs", "pubsub", "storage", "mailer":
		return value
	}
	return value
}

func validateAdapterSelection(cfg *config.Config) error {
	_, err := coreadapters.ResolveFromConfig(cfg)
	return err
}

func validateRuntimeCombination(cfg *config.Config) error {
	_, err := runtimeplan.Resolve(cfg)
	return err
}
