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
	RecipeForm            SemanticRecipe = "form"
	RecipeAlert           SemanticRecipe = "alert"
	RecipeAlertInfo       SemanticRecipe = "alert-info"
	RecipeAlertSuccess    SemanticRecipe = "alert-success"
	RecipeAlertDanger     SemanticRecipe = "alert-danger"
	RecipeCard            SemanticRecipe = "card"
	RecipeNav             SemanticRecipe = "nav"
	RecipeNavItem         SemanticRecipe = "nav-item"
	RecipeNavItemActive   SemanticRecipe = "nav-item-active"
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
	RecipeForm:            "gs-form",
	RecipeAlert:           "gs-alert",
	RecipeAlertInfo:       "gs-alert-info",
	RecipeAlertSuccess:    "gs-alert-success",
	RecipeAlertDanger:     "gs-alert-danger",
	RecipeCard:            "gs-card",
	RecipeNav:             "gs-nav",
	RecipeNavItem:         "gs-nav-item",
	RecipeNavItemActive:   "gs-nav-item-active",
}

func recipeClass(recipe SemanticRecipe) string {
	return semanticRecipeClasses[recipe]
}

func buttonClass(primary bool) string {
	variant := RecipeButtonSecondary
	if primary {
		variant = RecipeButtonPrimary
	}
	return recipeClass(RecipeButtonBase) + " " + recipeClass(variant)
}

func inputClass(statusClass string) string {
	base := recipeClass(RecipeFieldInput)
	if statusClass == "" {
		return base
	}
	return base + " " + statusClass
}

func formClass() string {
	return recipeClass(RecipeForm)
}

func alertClass(variant string) string {
	base := recipeClass(RecipeAlert)
	switch variant {
	case "success":
		return base + " " + recipeClass(RecipeAlertSuccess)
	case "danger":
		return base + " " + recipeClass(RecipeAlertDanger)
	default:
		return base + " " + recipeClass(RecipeAlertInfo)
	}
}

func cardClass() string {
	return recipeClass(RecipeCard)
}

func navClass() string {
	return recipeClass(RecipeNav)
}

func navItemClass(active bool) string {
	item := recipeClass(RecipeNavItem)
	if !active {
		return item
	}
	return item + " " + recipeClass(RecipeNavItemActive)
}
