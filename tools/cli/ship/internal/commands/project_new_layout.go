package commands

import (
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"

	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
)

var canonicalStarterTemplateFiles = []string{
	"README.md",
	"app/foundation/container.go",
	"app/router.go",
	"app/router_test.go",
	"app/views/templates.go",
	"app/views/web/pages/home_feed.templ",
	"app/views/web/pages/home_feed_templ.go",
	"app/views/web/pages/landing.templ",
	"app/views/web/pages/landing_templ.go",
	"app/views/web/pages/profile.templ",
	"app/views/web/pages/profile_templ.go",
	"app/web/routenames/routenames.go",
	"cmd/web/main.go",
	"config/modules.yaml",
}

var canonicalGeneratedProjectFiles = []string{
	".github/dependabot.yml",
	".github/workflows/ci.yml",
	".github/workflows/deploy.yml",
	".github/workflows/security.yml",
	".env",
	".env.example",
	"Makefile",
	"Procfile",
	"Procfile.dev",
	"Procfile.worker",
	"README.md",
	"app/emailsubscriptions/repo.go",
	"app/foundation/container.go",
	"app/jobs/jobs.go",
	"app/notifications/notifier.go",
	"app/profiles/repo.go",
	"app/router.go",
	"app/router_test.go",
	"app/subscriptions/repo.go",
	"app/views/templates.go",
	"app/views/web/layouts/base.templ",
	"app/views/web/pages/home_feed.templ",
	"app/views/web/pages/home_feed_templ.go",
	"app/views/web/pages/landing.templ",
	"app/views/web/pages/landing_templ.go",
	"app/views/web/pages/profile.templ",
	"app/views/web/pages/profile_templ.go",
	"app/web/controllers/controllers.go",
	"app/web/middleware/middleware.go",
	"app/web/routenames/routenames.go",
	"app/web/ui/ui.go",
	"app/web/viewmodels/viewmodels.go",
	"cmd/web/main.go",
	"cmd/worker/main.go",
	"config/modules.yaml",
	"db/bobgen.yaml",
	"db/gen/.gitkeep",
	"db/migrate/migrations/00001_starter_bootstrap.sql",
	"db/migrate/migrations/.gitkeep",
	"db/queries/user.sql",
	"docs/00-index.md",
	"docs/architecture/01-architecture.md",
	"docs/architecture/08-cognitive-model.md",
	"docs/architecture/10-extension-zones.md",
	"go.mod",
	"go.sum",
	"static/styles_bundle.css",
	"styles/styles.css",
	"tmp/.gitkeep",
	policies.AgentPolicyFilePath,
}

var canonicalGeneratedProjectI18nFiles = []string{
	"locales/en.toml",
	"locales/fr.toml",
}

func defaultNewLayoutArtifactPaths() []string {
	return []string{
		filepath.ToSlash(filepath.Join(policies.AgentGeneratedDir, "INSTALL.md")),
		filepath.ToSlash(filepath.Join(policies.AgentGeneratedDir, "allowed-prefixes.json")),
		filepath.ToSlash(filepath.Join(policies.AgentGeneratedDir, "agent-prefixes.txt")),
	}
}

func canonicalGeneratedProjectLayoutSnapshot(opts NewProjectOptions, artifactPaths []string) []string {
	files := append([]string(nil), canonicalGeneratedProjectFiles...)
	if opts.APIMode {
		excluded := map[string]struct{}{
			"app/views/web/layouts/base.templ":      {},
			"app/views/web/pages/home_feed.templ":   {},
			"app/views/web/pages/home_feed_templ.go": {},
			"app/views/web/pages/landing.templ":     {},
			"app/views/web/pages/landing_templ.go":  {},
			"app/views/web/pages/profile.templ":     {},
			"app/views/web/pages/profile_templ.go":  {},
			"static/styles_bundle.css":              {},
			"styles/styles.css":                     {},
		}
		filtered := make([]string, 0, len(files))
		for _, file := range files {
			if _, skip := excluded[file]; skip {
				continue
			}
			filtered = append(filtered, file)
		}
		files = filtered
	}
	files = append(files, artifactPaths...)
	if opts.I18nEnabled {
		files = append(files, canonicalGeneratedProjectI18nFiles...)
	}

	fileSet := make(map[string]struct{}, len(files))
	dirSet := make(map[string]struct{})
	for _, file := range files {
		clean := path.Clean(filepath.ToSlash(file))
		if clean == "." {
			continue
		}
		fileSet[clean] = struct{}{}
		for dir := path.Dir(clean); dir != "." && dir != "/"; dir = path.Dir(dir) {
			dirSet[dir] = struct{}{}
		}
	}

	snapshot := make([]string, 0, len(fileSet)+len(dirSet))
	for dir := range dirSet {
		snapshot = append(snapshot, dir+"/")
	}
	for file := range fileSet {
		snapshot = append(snapshot, file)
	}
	sort.Strings(snapshot)
	return snapshot
}

func snapshotGeneratedProjectLayout(root string) ([]string, error) {
	var snapshot []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			snapshot = append(snapshot, rel+"/")
			return nil
		}
		snapshot = append(snapshot, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(snapshot)
	return snapshot, nil
}

func starterTemplateLayoutSnapshot(templateFS fs.FS, root string) ([]string, error) {
	var snapshot []string
	err := fs.WalkDir(templateFS, root, func(current string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == root {
			return nil
		}
		rel := strings.TrimPrefix(filepath.ToSlash(current), filepath.ToSlash(root)+"/")
		if d.IsDir() {
			snapshot = append(snapshot, rel+"/")
			return nil
		}
		snapshot = append(snapshot, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(snapshot)
	return snapshot, nil
}

func renderBaseLayoutTempl(provider string) string {
	headAssets := ""
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case newUIProviderDaisy:
		headAssets = `		<script src="https://cdn.jsdelivr.net/npm/flowbite@2.5.2/dist/flowbite.min.js"></script>
`
	case newUIProviderBare:
		headAssets = ""
	default:
		headAssets = `		<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/franken-ui@2.0.0/dist/css/core.min.css" />
		<script src="https://cdn.jsdelivr.net/npm/uikit@3.21.16/dist/js/uikit.min.js"></script>
`
	}

	return `package layouts

// Renders: base app layout shell with provider-selected head assets.
templ Base(children templ.Component) {
	<html lang="en" data-component="base">
	<head>
		<meta charset="UTF-8" />
		<meta name="viewport" content="width=device-width, initial-scale=1.0" />
` + headAssets + `	</head>
	<body>
		@children
	</body>
	</html>
}
`
}
