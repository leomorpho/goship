package ui

import "testing"

func TestSemanticRecipeClassRegistry(t *testing.T) {
	cases := map[SemanticRecipe]string{
		RecipePage:            "gs-page",
		RecipePanel:           "gs-panel",
		RecipeTitle:           "gs-title",
		RecipeText:            "gs-text",
		RecipeKicker:          "gs-kicker",
		RecipeStack:           "gs-stack",
		RecipeMutedColor:      "gs-color-muted",
		RecipeDangerColor:     "gs-color-danger",
		RecipeElevationFloat:  "gs-elevation-float",
		RecipeButtonBase:      "gs-button",
		RecipeButtonPrimary:   "gs-button-primary",
		RecipeButtonSecondary: "gs-button-secondary",
		RecipeFieldInput:      "gs-field-input",
		RecipeFieldHint:       "gs-field-hint",
		RecipeFieldError:      "gs-field-error",
		RecipeFieldSuccess:    "gs-field-success",
		RecipeForm:            "gs-form",
		RecipeAlert:           "gs-alert",
		RecipeAlertInfo:       "gs-alert-info",
		RecipeAlertSuccess:    "gs-alert-success",
		RecipeAlertDanger:     "gs-alert-danger",
		RecipeCard:            "gs-card",
		RecipeNav:             "gs-nav",
		RecipeNavItem:         "gs-nav-item",
		RecipeNavItemActive:   "gs-nav-item-active",
		RecipeLayoutShell:     "gs-layout-shell",
		RecipeLayoutHeader:    "gs-layout-header",
		RecipeLayoutContent:   "gs-layout-content",
		RecipeLayoutFooter:    "gs-layout-footer",
		RecipeIslandMount:     "gs-island-mount",
	}

	for recipe, want := range cases {
		if got := recipeClass(recipe); got != want {
			t.Fatalf("recipe %q class = %q, want %q", recipe, got, want)
		}
	}
}

func TestSemanticRecipeMarkupConsistency(t *testing.T) {
	if got := buttonClass(true); got != "gs-button gs-button-primary" {
		t.Fatalf("buttonClass(true)=%q", got)
	}
	if got := buttonClass(false); got != "gs-button gs-button-secondary" {
		t.Fatalf("buttonClass(false)=%q", got)
	}
	if got := inputClass(""); got != "gs-field-input" {
		t.Fatalf("inputClass(empty)=%q", got)
	}
	if got := inputClass("gs-field-error"); got != "gs-field-input gs-field-error" {
		t.Fatalf("inputClass(error)=%q", got)
	}
	if got := formClass(); got != "gs-form" {
		t.Fatalf("formClass()=%q", got)
	}
	if got := alertClass("info"); got != "gs-alert gs-alert-info" {
		t.Fatalf("alertClass(info)=%q", got)
	}
	if got := alertClass("success"); got != "gs-alert gs-alert-success" {
		t.Fatalf("alertClass(success)=%q", got)
	}
	if got := alertClass("danger"); got != "gs-alert gs-alert-danger" {
		t.Fatalf("alertClass(danger)=%q", got)
	}
	if got := cardClass(); got != "gs-card" {
		t.Fatalf("cardClass()=%q", got)
	}
	if got := navClass(); got != "gs-nav" {
		t.Fatalf("navClass()=%q", got)
	}
	if got := navItemClass(false); got != "gs-nav-item" {
		t.Fatalf("navItemClass(false)=%q", got)
	}
	if got := navItemClass(true); got != "gs-nav-item gs-nav-item-active" {
		t.Fatalf("navItemClass(true)=%q", got)
	}
	if got := layoutShellClass(); got != "gs-layout-shell" {
		t.Fatalf("layoutShellClass()=%q", got)
	}
	if got := layoutHeaderClass(); got != "gs-layout-header" {
		t.Fatalf("layoutHeaderClass()=%q", got)
	}
	if got := layoutContentClass(); got != "gs-layout-content" {
		t.Fatalf("layoutContentClass()=%q", got)
	}
	if got := layoutFooterClass(); got != "gs-layout-footer" {
		t.Fatalf("layoutFooterClass()=%q", got)
	}
	if got := islandMountClass(); got != "gs-island-mount" {
		t.Fatalf("islandMountClass()=%q", got)
	}
}
