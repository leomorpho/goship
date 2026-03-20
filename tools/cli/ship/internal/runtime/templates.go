package runtime

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func RelocateTemplGenerated(rootPath string) error {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absRoot); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	goModDir, modulePath, err := FindGoModule(absRoot)
	if err != nil {
		return err
	}

	var generatedFiles []string
	err = filepath.WalkDir(absRoot, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == "gen" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), "_templ.go") {
			generatedFiles = append(generatedFiles, p)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(generatedFiles) == 0 {
		return nil
	}

	importMap := make(map[string]string)
	movedFiles := make([]string, 0, len(generatedFiles))
	for _, src := range generatedFiles {
		srcDir := filepath.Dir(src)
		relDir, err := filepath.Rel(goModDir, srcDir)
		if err != nil {
			return err
		}
		oldImport := path.Join(modulePath, filepath.ToSlash(relDir))
		newImport := path.Join(oldImport, "gen")
		importMap[oldImport] = newImport

		dstDir := filepath.Join(srcDir, "gen")
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return err
		}
		dst := filepath.Join(dstDir, filepath.Base(src))
		_ = os.Remove(dst)
		if err := os.Rename(src, dst); err != nil {
			return err
		}
		movedFiles = append(movedFiles, dst)
	}

	for _, file := range movedFiles {
		b, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		content := string(b)
		for oldImport, newImport := range importMap {
			content = strings.ReplaceAll(content, `"`+oldImport+`"`, `"`+newImport+`"`)
		}
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			return err
		}
	}

	return nil
}
