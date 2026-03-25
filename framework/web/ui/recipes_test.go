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
	}

	for recipe, want := range cases {
		if got := recipeClass(recipe); got != want {
			t.Fatalf("recipe %q class = %q, want %q", recipe, got, want)
		}
	}
}
