package ui

// SemanticRecipe names a reusable UI recipe class contract.
type SemanticRecipe string

const (
	RecipePage            SemanticRecipe = "page"
	RecipePanel           SemanticRecipe = "panel"
	RecipeTitle           SemanticRecipe = "title"
	RecipeText            SemanticRecipe = "text"
	RecipeKicker          SemanticRecipe = "kicker"
	RecipeStack           SemanticRecipe = "stack"
	RecipeMutedColor      SemanticRecipe = "muted-color"
	RecipeDangerColor     SemanticRecipe = "danger-color"
	RecipeElevationFloat  SemanticRecipe = "elevation-float"
	RecipeButtonBase      SemanticRecipe = "button-base"
	RecipeButtonPrimary   SemanticRecipe = "button-primary"
	RecipeButtonSecondary SemanticRecipe = "button-secondary"
	RecipeFieldInput      SemanticRecipe = "field-input"
	RecipeFieldHint       SemanticRecipe = "field-hint"
	RecipeFieldError      SemanticRecipe = "field-error"
	RecipeFieldSuccess    SemanticRecipe = "field-success"
)

var semanticRecipeClasses = map[SemanticRecipe]string{
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

func recipeClass(recipe SemanticRecipe) string {
	return semanticRecipeClasses[recipe]
}
