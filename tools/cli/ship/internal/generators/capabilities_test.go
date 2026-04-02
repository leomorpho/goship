package generators

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseGeneratorCRUDActionNames(t *testing.T) {
	actions, err := ParseGeneratorCRUDActionNames("index, show, create, update, destroy,show")
	if err != nil {
		t.Fatalf("ParseGeneratorCRUDActionNames() error = %v", err)
	}
	want := []string{"index", "show", "create", "update", "destroy"}
	if !reflect.DeepEqual(actions, want) {
		t.Fatalf("ParseGeneratorCRUDActionNames() = %v, want %v", actions, want)
	}
}

func TestParseGeneratorCRUDActionNamesRejectsUnsupportedAction(t *testing.T) {
	if _, err := ParseGeneratorCRUDActionNames("index,publish"); err == nil {
		t.Fatal("ParseGeneratorCRUDActionNames() error = nil, want unsupported action error")
	}
}

func TestGeneratorCapabilityModelForStarterScaffold(t *testing.T) {
	starterRoot := filepath.Join("..", "templates", "starter", "testdata", "scaffold")
	model := CapabilityModelForRoot(starterRoot)
	if model.Workspace != GeneratorWorkspaceStarterScaffold {
		t.Fatalf("Workspace = %q, want %q", model.Workspace, GeneratorWorkspaceStarterScaffold)
	}
	if model.SupportsControllerGeneration {
		t.Fatal("starter scaffold should not yet support controller generation")
	}
	if model.SupportsScaffoldGeneration {
		t.Fatal("starter scaffold should not yet support scaffold generation")
	}
	if model.ResourceRouteStyle != GeneratorResourceRouteSinglePath {
		t.Fatalf("ResourceRouteStyle = %q, want %q", model.ResourceRouteStyle, GeneratorResourceRouteSinglePath)
	}
	if model.ResourcePersistence != GeneratorResourcePersistenceStarterRuntime {
		t.Fatalf("ResourcePersistence = %q, want %q", model.ResourcePersistence, GeneratorResourcePersistenceStarterRuntime)
	}
	wantActions := []string{"index", "show", "create", "update", "destroy"}
	if !reflect.DeepEqual(model.SupportedResourceActions, wantActions) {
		t.Fatalf("SupportedResourceActions = %v, want %v", model.SupportedResourceActions, wantActions)
	}
}

func TestGeneratorCapabilityModelForFrameworkWorkspace(t *testing.T) {
	root := t.TempDir()
	model := CapabilityModelForRoot(root)
	if model.Workspace != GeneratorWorkspaceFramework {
		t.Fatalf("Workspace = %q, want %q", model.Workspace, GeneratorWorkspaceFramework)
	}
	if !model.SupportsControllerGeneration {
		t.Fatal("framework workspace should support controller generation")
	}
	if !model.SupportsScaffoldGeneration {
		t.Fatal("framework workspace should support scaffold generation")
	}
	if model.ResourceRouteStyle != GeneratorResourceRouteREST {
		t.Fatalf("ResourceRouteStyle = %q, want %q", model.ResourceRouteStyle, GeneratorResourceRouteREST)
	}
	if model.ResourcePersistence != GeneratorResourcePersistenceModelBacked {
		t.Fatalf("ResourcePersistence = %q, want %q", model.ResourcePersistence, GeneratorResourcePersistenceModelBacked)
	}
}
