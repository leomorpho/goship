package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"
)

func syncLocalModuleDependency(root string, info moduleInfo, dryRun bool, out io.Writer) (bool, error) {
	if strings.TrimSpace(info.ModulePath) == "" || strings.TrimSpace(info.LocalPath) == "" {
		return false, nil
	}

	var changed bool

	goModPath := filepath.Join(root, "go.mod")
	goModChanged, goModContent, err := updateGoModDependency(goModPath, info.ModulePath, info.LocalPath, true)
	if err != nil {
		return false, err
	}
	if goModChanged {
		if err := writeOrDiff(goModPath, goModContent, dryRun, out); err != nil {
			return false, err
		}
		changed = true
	}

	goWorkPath := filepath.Join(root, "go.work")
	goWorkChanged, goWorkContent, err := updateGoWorkUse(goWorkPath, info.LocalPath, info.ModulePath, true)
	if err != nil {
		return false, err
	}
	if goWorkChanged {
		if err := writeOrDiff(goWorkPath, goWorkContent, dryRun, out); err != nil {
			return false, err
		}
		changed = true
	}

	return changed, nil
}

func removeLocalModuleDependency(root string, info moduleInfo, dryRun bool, out io.Writer) (bool, error) {
	if strings.TrimSpace(info.ModulePath) == "" || strings.TrimSpace(info.LocalPath) == "" {
		return false, nil
	}

	var changed bool

	goModPath := filepath.Join(root, "go.mod")
	goModChanged, goModContent, err := updateGoModDependency(goModPath, info.ModulePath, info.LocalPath, false)
	if err != nil {
		return false, err
	}
	if goModChanged {
		if err := writeOrDiff(goModPath, goModContent, dryRun, out); err != nil {
			return false, err
		}
		changed = true
	}

	goWorkPath := filepath.Join(root, "go.work")
	goWorkChanged, goWorkContent, err := updateGoWorkUse(goWorkPath, info.LocalPath, info.ModulePath, false)
	if err != nil {
		return false, err
	}
	if goWorkChanged {
		if err := writeOrDiff(goWorkPath, goWorkContent, dryRun, out); err != nil {
			return false, err
		}
		changed = true
	}

	return changed, nil
}

func updateGoModDependency(path, modulePath, localPath string, add bool) (bool, string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return false, "", fmt.Errorf("read %s: %w", path, err)
	}
	if !add && !strings.Contains(string(body), modulePath) {
		return false, string(body), nil
	}

	file, err := modfile.Parse(path, body, nil)
	if err != nil {
		return false, "", fmt.Errorf("parse %s: %w", path, err)
	}

	before, err := file.Format()
	if err != nil {
		return false, "", fmt.Errorf("format %s: %w", path, err)
	}

	if add {
		localModulePath := "./" + filepath.ToSlash(localPath)
		if !hasGoModRequire(file, modulePath) {
			file.AddNewRequire(modulePath, "v0.0.0", false)
		}
		if !hasGoModReplace(file, modulePath, localModulePath) {
			if err := file.DropReplace(modulePath, ""); err != nil {
				return false, "", fmt.Errorf("drop stale %s replace: %w", path, err)
			}
			if err := file.AddReplace(modulePath, "", localModulePath, ""); err != nil {
				return false, "", fmt.Errorf("update %s replace: %w", path, err)
			}
		}
	} else {
		if err := file.DropRequire(modulePath); err != nil {
			return false, "", fmt.Errorf("drop require %s: %w", modulePath, err)
		}
		if err := file.DropReplace(modulePath, ""); err != nil {
			return false, "", fmt.Errorf("drop replace %s: %w", modulePath, err)
		}
	}

	file.Cleanup()
	file.SortBlocks()
	after, err := file.Format()
	if err != nil {
		return false, "", fmt.Errorf("format %s: %w", path, err)
	}
	if string(before) == string(after) {
		return false, string(before), nil
	}
	return true, string(after), nil
}

func hasGoModRequire(file *modfile.File, modulePath string) bool {
	for _, req := range file.Require {
		if req.Mod.Path == modulePath {
			return true
		}
	}
	return false
}

func hasGoModReplace(file *modfile.File, modulePath, localModulePath string) bool {
	for _, repl := range file.Replace {
		if repl.Old.Path != modulePath || repl.Old.Version != "" {
			continue
		}
		if repl.New.Path == localModulePath && repl.New.Version == "" {
			return true
		}
	}
	return false
}

func updateGoWorkUse(path, localPath, modulePath string, add bool) (bool, string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return false, "", fmt.Errorf("read %s: %w", path, err)
	}
	usePath := "./" + filepath.ToSlash(localPath)
	if !add && !strings.Contains(string(body), usePath) {
		return false, string(body), nil
	}

	file, err := modfile.ParseWork(path, body, nil)
	if err != nil {
		return false, "", fmt.Errorf("parse %s: %w", path, err)
	}

	before := string(modfile.Format(file.Syntax))
	if add {
		if !hasGoWorkUse(file, usePath, modulePath) {
			if err := file.AddUse(usePath, modulePath); err != nil {
				return false, "", fmt.Errorf("add use %s: %w", usePath, err)
			}
		}
	} else {
		if err := file.DropUse(usePath); err != nil {
			return false, "", fmt.Errorf("drop use %s: %w", usePath, err)
		}
	}

	file.Cleanup()
	file.SortBlocks()
	after := string(modfile.Format(file.Syntax))
	if before == after {
		return false, before, nil
	}
	return true, after, nil
}

func hasGoWorkUse(file *modfile.WorkFile, diskPath, modulePath string) bool {
	for _, use := range file.Use {
		if use.Path == diskPath && use.ModulePath == modulePath {
			return true
		}
	}
	return false
}

func findModuleRemovalBlockers(root string, info moduleInfo) ([]string, error) {
	if strings.TrimSpace(info.ModulePath) == "" {
		return nil, nil
	}

	managed := map[string]struct{}{
		filepath.Clean(filepath.Join(root, "go.mod")):                            {},
		filepath.Clean(filepath.Join(root, "go.work")):                           {},
		filepath.Clean(filepath.Join(root, "config", "modules.yaml")):            {},
		filepath.Clean(filepath.Join(root, "app", "foundation", "container.go")): {},
	}

	blockers := []string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".docket", ".worktrees", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		if _, ok := managed[filepath.Clean(path)]; ok {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(body), info.ModulePath) {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			blockers = append(blockers, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(blockers)
	return blockers, nil
}
