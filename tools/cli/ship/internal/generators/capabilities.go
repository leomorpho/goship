package generators

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

type GeneratorWorkspace string

const (
	GeneratorWorkspaceFramework       GeneratorWorkspace = "framework"
	GeneratorWorkspaceStarterScaffold GeneratorWorkspace = "starter-scaffold"
)

type GeneratorResourceRouteStyle string

const (
	GeneratorResourceRouteREST       GeneratorResourceRouteStyle = "rest"
	GeneratorResourceRouteSinglePath GeneratorResourceRouteStyle = "single-path"
)

type GeneratorResourcePersistence string

const (
	GeneratorResourcePersistenceModelBacked    GeneratorResourcePersistence = "model-backed"
	GeneratorResourcePersistenceStarterRuntime GeneratorResourcePersistence = "starter-runtime"
)

type GeneratorCapabilityModel struct {
	Workspace                    GeneratorWorkspace
	ResourceRouteStyle           GeneratorResourceRouteStyle
	ResourcePersistence          GeneratorResourcePersistence
	SupportedResourceActions     []string
	SupportsControllerGeneration bool
	SupportsScaffoldGeneration   bool
}

var generatorCRUDActionNames = []string{"index", "show", "create", "update", "destroy"}

func ParseGeneratorCRUDActionNames(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("actions cannot be empty")
	}
	var out []string
	for _, token := range strings.Split(raw, ",") {
		action := strings.ToLower(strings.TrimSpace(token))
		if action == "" {
			continue
		}
		if !slices.Contains(generatorCRUDActionNames, action) {
			return nil, fmt.Errorf("unsupported action %q", action)
		}
		if !slices.Contains(out, action) {
			out = append(out, action)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("actions cannot be empty")
	}
	return out, nil
}

func DefaultGeneratorCRUDActionNames() []string {
	return append([]string(nil), generatorCRUDActionNames...)
}

func CapabilityModelForRoot(root string) GeneratorCapabilityModel {
	if looksLikeStarterScaffoldRoot(root) {
		return GeneratorCapabilityModel{
			Workspace:                    GeneratorWorkspaceStarterScaffold,
			ResourceRouteStyle:           GeneratorResourceRouteSinglePath,
			ResourcePersistence:          GeneratorResourcePersistenceStarterRuntime,
			SupportedResourceActions:     DefaultGeneratorCRUDActionNames(),
			SupportsControllerGeneration: false,
			SupportsScaffoldGeneration:   false,
		}
	}
	return GeneratorCapabilityModel{
		Workspace:                    GeneratorWorkspaceFramework,
		ResourceRouteStyle:           GeneratorResourceRouteREST,
		ResourcePersistence:          GeneratorResourcePersistenceModelBacked,
		SupportedResourceActions:     DefaultGeneratorCRUDActionNames(),
		SupportsControllerGeneration: true,
		SupportsScaffoldGeneration:   true,
	}
}

func CapabilityModelForPath(path string) GeneratorCapabilityModel {
	root := filepath.Dir(path)
	if root == "" {
		root = "."
	}
	return CapabilityModelForRoot(root)
}
