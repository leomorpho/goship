package cmd

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// scaffoldCmd represents the scaffold command
var scaffoldCmd = &cobra.Command{
	Use:     "scaffold [model name]",
	Short:   "Generate a scaffold with controller and views for an existing model",
	Long:    `This command generates the necessary scaffold for controller and views with the given fields from an existing schema`,
	Example: `goship generate scaffold User`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		modelName := args[0]

		fmt.Println("Generating scaffold for model:", modelName)

		// TODO: convert fields to be a list of structs instead of list of map
		// Get fields from existing schema
		fields, err := parseFields(modelName)
		if err != nil {
			fmt.Printf("Error reading schema: %v\n", err)
			return
		}

		// Generate form struct
		generateTypeStructs(modelName, fields)

		// Generate controller routes
		generateRoutes(modelName, fields)

		// Generate templ templates
		generateTemplates(modelName, fields)

		// After the scaffolding logic, run the `templ generate` command
		if err := runTemplGenerate(); err != nil {
			fmt.Println("Error running templ generate:", err)
		}

		fmt.Println("Scaffold generation complete.")
	},
}

func init() {
	// Attach scaffold command to the generate command
	generateCmd.AddCommand(scaffoldCmd)
}

type FieldType struct {
	Name      string
	EntType   string
	GoType    string
	InputType string
}

// parseFields now takes a modelName and returns the parsed fields from the Ent schema
func parseFields(modelName string) ([]FieldType, error) {
	schemaDir := "ent/schema" // Adjust this path if your schema is located elsewhere
	schemaFile := filepath.Join(schemaDir, strings.ToLower(modelName)+".go")

	// Read the schema file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, schemaFile, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema file: %v", err)
	}

	var fields []FieldType

	// Find the Fields() method in the schema
	ast.Inspect(node, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Name.Name != "Fields" {
			return true
		}

		// Parse the fields within the Fields() method
		ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
			callExpr, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			// Check if it's a field definition (e.g., field.String("name"))
			selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
			if !ok || selectorExpr.Sel.Name != "String" && selectorExpr.Sel.Name != "Int" && selectorExpr.Sel.Name != "Bool" && selectorExpr.Sel.Name != "Time" {
				return true
			}

			// Extract field name and type
			if len(callExpr.Args) > 0 {
				fieldName, ok := callExpr.Args[0].(*ast.BasicLit)
				if ok {
					entType := selectorExpr.Sel.Name
					goType := mapEntTypeToGoType(entType)
					inputType := mapGoTypeToHtmlInputType(goType)

					fields = append(fields, FieldType{
						Name:      strings.Trim(fieldName.Value, "\""),
						EntType:   entType,
						GoType:    goType,
						InputType: inputType,
					})
				}
			}

			return true
		})

		return false
	})

	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields found in schema for model %s", modelName)
	}

	return fields, nil
}

// mapEntTypeToGoType converts Ent types to Go types
func mapEntTypeToGoType(entType string) string {
	switch entType {
	case "String":
		return "string"
	case "Int":
		return "int"
	case "Bool":
		return "bool"
	case "Time":
		return "time.Time"
	default:
		return "string" // Default to string for unknown types
	}
}

func mapGoTypeToHtmlInputType(goType string) string {
	switch goType {
	case "string":
		return "text" // default input for strings
	case "int", "int8", "int16", "int32", "int64":
		return "number" // integer types
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "number" // unsigned integers
	case "float32", "float64":
		return "number" // float types can be input as numbers
	case "bool":
		return "checkbox" // boolean maps to checkbox
	case "time.Time":
		return "datetime-local" // Go's time.Time type mapped to datetime-local
	case "[]byte":
		return "file" // byte slice often represents file upload
	case "complex64", "complex128":
		return "text" // complex numbers are rare and can be handled as text
	case "interface{}":
		return "text" // fallback for any generic type
	default:
		return "text" // fallback for other unmapped Go types
	}
}

// Function to generate form struct for the model
func generateTypeStructs(modelName string, fieldsData []FieldType) {
	// Convert modelName to PascalCase for struct naming using cases.Title
	modelNamePascal := getPascalCase(modelName)

	// Define the form file path
	formDir := "pkg/types"
	filePath := filepath.Join(formDir, strings.ToLower(modelName)+".go")

	// Create a map to track required imports
	imports := map[string]bool{
		"github.com/mikestefanello/pagoda/pkg/controller": true,
		"github.com/mikestefanello/pagoda/ent":            true,
	}

	// Check field types and add required imports
	for _, field := range fieldsData {
		switch field.GoType {
		case "time.Time":
			imports["time"] = true
			// TODO: expand for new usecases
		}
	}

	// Collect the list of imports from the map
	var importList []string
	for imp := range imports {
		importList = append(importList, imp)
	}

	// Create the form struct template
	formTemplate := `package types

{{- if .Imports }}
import (
	{{- range .Imports }}
	"{{ . }}"
	{{- end }}
)
{{- end }}

// {{.ModelName}}Form holds the form fields for {{.ModelName}}.
type {{.ModelName}}Form struct {
	{{- range .Fields}}
	{{getPascalCase .Name}} {{.GoType}} ` + "`" + `form:"{{.Name}}"` + "`" + `
	{{- end}}
	Submission         controller.FormSubmission
}

// {{.ModelName}}IndexData holds multiple instances of {{.ModelName}} for the Index view.
type {{.ModelName}}IndexData struct {
	Items []*ent.{{.ModelName}}
}

// {{.ModelName}}ViewData holds a single instance of {{.ModelName}} for the Show and Edit views.
type {{.ModelName}}ViewData struct {
	Item *ent.{{.ModelName}}
}
`

	// Create a function map to pass to the template
	funcMap := template.FuncMap{
		"getPascalCase": getPascalCase,
	}

	// Use Go templating to inject data into the form struct template
	tmpl, _ := template.New("formStruct").Funcs(funcMap).Parse(formTemplate)
	file, _ := os.Create(filePath)
	defer file.Close()

	tmpl.Execute(file, map[string]interface{}{
		"ModelName": modelNamePascal,
		"Fields":    fieldsData,
		"Imports":   importList,
	})

	fmt.Println("Form and Data structs generated:", filePath)
}

// Function to generate a routes file for a model and hook into Templ components
func generateRoutes(modelName string, fieldsData []FieldType) {
	// Convert modelName to lowercase for filenames and routing paths
	modelNameLower := strings.ToLower(modelName)
	modelNamePascal := getPascalCase(modelName)

	// Define the directory and file for the routes
	routeDir := "pkg/routes/"
	routeFile := filepath.Join(routeDir, modelNameLower+".go")

	// Create the routes directory if it doesn't exist
	os.MkdirAll(routeDir, os.ModePerm)

	// Define the template for the routes file
	const routeTemplate = `package routes

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/mikestefanello/pagoda/pkg/context"
	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/pkg/repos/msg"
	"github.com/mikestefanello/pagoda/pkg/types"
	"github.com/mikestefanello/pagoda/templates/layouts"
	"github.com/mikestefanello/pagoda/templates/pages"
)

type {{.ModelNameLower}}Route struct {
	ctr controller.Controller
}

func New{{.ModelName}}Route(ctr controller.Controller) *{{.ModelNameLower}}Route {
	return &{{.ModelNameLower}}Route{
		ctr: ctr,
	}
}

// GET /{{.ModelNameLower}}
func (c *{{.ModelNameLower}}Route) Index(ctx echo.Context) error {
	// Query all entities
	{{.ModelNameLower}}s, err := c.ctr.Container.ORM.{{.ModelName}}.Query().All(ctx.Request().Context())
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	
	page := controller.NewPage(ctx)
	page.Layout = layouts.Main // TODO: update if necessary
	page.Name = "Index" // TODO: rename
	page.Component = pages.{{.ModelName}}Index(&page)
	page.Data = types.{{.ModelName}}IndexData{
		Items: {{.ModelNameLower}}s,
	}
	return c.ctr.RenderPage(ctx, page)
}

// GET /{{.ModelNameLower}}/:id
func (c *{{.ModelNameLower}}Route) Show(ctx echo.Context) error {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return err
	}

	// Query the entity by ID
	{{.ModelNameLower}}, err := c.ctr.Container.ORM.{{.ModelName}}.Get(ctx.Request().Context(), id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": "Entity not found"})
	}

	page := controller.NewPage(ctx)
	page.Layout = layouts.Main // TODO: update if necessary
	page.Name = "Show" // TODO: rename
	page.Component = pages.{{.ModelName}}Show(&page)
	page.Form = &types.{{.ModelName}}Form{}
	page.Data = types.{{.ModelName}}ViewData{
		Item: {{.ModelNameLower}},
	}

	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*types.{{.ModelName}}Form)
	}

	return c.ctr.RenderPage(ctx, page)
}

// GET /{{.ModelNameLower}}/create
func (c *{{.ModelNameLower}}Route) Create(ctx echo.Context) error {
	page := controller.NewPage(ctx)
	page.Name = "Create"
	page.Component = pages.{{.ModelName}}Create(&page)
	page.Form = &types.{{.ModelName}}Form{}
	return c.ctr.RenderPage(ctx, page)
}

// POST /{{.ModelNameLower}} (for creating a new entity)
func (c *{{.ModelNameLower}}Route) Store(ctx echo.Context) error {
	var form types.{{.ModelName}}Form
	ctx.Set(context.FormKey, &form)

	// Parse the form values
	if err := ctx.Bind(&form); err != nil {
		return c.ctr.Fail(err, "unable to parse form")
	}

	if err := form.Submission.Process(ctx, form); err != nil {
		return c.ctr.Fail(err, "unable to process form submission")
	}
	
	if form.Submission.HasErrors() {
		return c.Create(ctx)
	}

	// Create a new entity
	creator := c.ctr.Container.ORM.{{.ModelName}}.Create()
	{{range .Fields}}
	creator.Set{{getPascalCase .Name}}(form.{{getPascalCase .Name}})
	{{- end}}
	
	_, err := creator.Save(ctx.Request().Context())
	if err != nil {
		return err
	}

	msg.Success(ctx, "New {{.ModelName}} created successfully. 👌")
	return c.Index(ctx)
}


// GET /{{.ModelNameLower}}/:id/edit
func (c *{{.ModelNameLower}}Route) Edit(ctx echo.Context) error {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return err
	}

	{{.ModelNameLower}}, err := c.ctr.Container.ORM.{{.ModelName}}.Get(ctx.Request().Context(), id)
	if err != nil {
		return err
	}

	page := controller.NewPage(ctx)
	page.Name = "Edit"
	page.Component = pages.{{.ModelName}}Edit(&page)
	page.Form = &types.{{.ModelName}}Form{
		{{- range .Fields}}
		{{getPascalCase .Name}}: {{$.ModelNameLower}}.{{getPascalCase .Name}},
		{{- end}}
	}
	page.Data = types.{{.ModelName}}ViewData{
		Item: {{.ModelNameLower}},
	}

	return c.ctr.RenderPage(ctx, page)
}


// POST /{{.ModelNameLower}}/:id (for updating an existing entity)
func (c *{{.ModelNameLower}}Route) Update(ctx echo.Context) error {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return err
	}

	var form types.{{.ModelName}}Form
	ctx.Set(context.FormKey, &form)

	// Parse the form values
	if err := ctx.Bind(&form); err != nil {
		return c.ctr.Fail(err, "unable to parse form")
	}

	if err := form.Submission.Process(ctx, form); err != nil {
		return c.ctr.Fail(err, "unable to process form submission")
	}
	
	if form.Submission.HasErrors() {
		return c.Edit(ctx)
	}
			
	// Update the entity
	updater := c.ctr.Container.ORM.{{.ModelName}}.UpdateOneID(id)
	{{range .Fields}}
	updater.Set{{getPascalCase .Name}}(form.{{getPascalCase .Name}})
	{{- end}}
	
	_, err = updater.Save(ctx.Request().Context())
	if err != nil {
		return err
	}

	msg.Success(ctx, "{{.ModelName}} updated successfully. 👌")

	return c.Show(ctx)
}
`

	// Create a function map to pass to the template
	funcMap := template.FuncMap{
		"getPascalCase": getPascalCase,
	}

	// Create a template instance and parse the routeTemplate
	tmpl, err := template.New("route").Funcs(funcMap).Parse(routeTemplate)
	if err != nil {
		fmt.Println("Error parsing template:", err)
		return
	}

	// Create the file
	file, err := os.Create(routeFile)
	if err != nil {
		fmt.Println("Error creating route file:", err)
		return
	}
	defer file.Close()

	// Data for the template
	data := map[string]interface{}{
		"ModelName":      modelNamePascal, // PascalCase model name
		"ModelNameLower": modelNameLower,  // Lowercase model name for paths
		"Fields":         fieldsData,
	}

	// Execute the template with data and write to file
	err = tmpl.Execute(file, data)
	if err != nil {
		fmt.Println("Error writing to route file:", err)
		return
	}

	// Define the router template
	const routerTemplate = `
# Move the below to where you store your route names
routeName{{.ModelName}}Index = "{{.ModelNameLower}}.index"
routeName{{.ModelName}}Show = "{{.ModelNameLower}}.show"
routeName{{.ModelName}}Create = "{{.ModelNameLower}}.create"
routeName{{.ModelName}}Store = "{{.ModelNameLower}}.store"
routeName{{.ModelName}}Edit = "{{.ModelNameLower}}.edit"
routeName{{.ModelName}}Update = "{{.ModelNameLower}}.update"
routeName{{.ModelName}}Delete = "{{.ModelNameLower}}.delete"

# Attach the below to the correct router
{{.ModelNameLower}}Routes := New{{.ModelName}}Route(ctr)
onboardedGroup.GET("/{{.ModelNameLower}}", {{.ModelNameLower}}Routes.Index).Name = routeName{{.ModelName}}Index
onboardedGroup.GET("/{{.ModelNameLower}}/:id", {{.ModelNameLower}}Routes.Show).Name = routeName{{.ModelName}}Show
onboardedGroup.GET("/{{.ModelNameLower}}/create", {{.ModelNameLower}}Routes.Create).Name = routeName{{.ModelName}}Create
onboardedGroup.POST("/{{.ModelNameLower}}", {{.ModelNameLower}}Routes.Store).Name = routeName{{.ModelName}}Store
onboardedGroup.GET("/{{.ModelNameLower}}/:id/edit", {{.ModelNameLower}}Routes.Edit).Name = routeName{{.ModelName}}Edit
onboardedGroup.POST("/{{.ModelNameLower}}/:id", {{.ModelNameLower}}Routes.Update).Name = routeName{{.ModelName}}Update
onboardedGroup.DELETE("/{{.ModelNameLower}}/:id", {{.ModelNameLower}}Routes.Delete).Name = routeName{{.ModelName}}Delete
`
	// Prepare the template data
	routerData := map[string]string{
		"ModelName":      modelNamePascal,
		"ModelNameLower": strings.ToLower(modelName),
	}

	// Parse and execute the template
	tmpl, err = template.New("router").Parse(routerTemplate)
	if err != nil {
		fmt.Println("Error parsing template:", err)
		return
	}

	// Create a buffer to capture the template output
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, routerData)
	if err != nil {
		fmt.Println("Error executing template:", err)
	}
	fmt.Println()
	fmt.Println("Add the following lines to your echo router to hook up the routes:")
	fmt.Println(buf.String())

	fmt.Println("Routes generated:", routeFile)
}

// Function to generate a single .templ scaffold
func generateTemplates(modelName string, fieldsData []FieldType) {
	// Convert modelName to lowercase for directory and file naming
	modelNameLower := strings.ToLower(modelName)

	// Convert modelName to PascalCase for struct naming
	modelNamePascal := getPascalCase(modelName)

	templateDir := "templates/pages/"
	filePath := filepath.Join(templateDir, modelNameLower+".templ")

	// Create the templates directory if it doesn't exist
	os.MkdirAll(templateDir, os.ModePerm)

	// Create a map to track required imports
	imports := map[string]bool{
		"fmt": true,
		"github.com/mikestefanello/pagoda/pkg/controller":       true,
		"github.com/mikestefanello/pagoda/pkg/types":            true,
		"github.com/mikestefanello/pagoda/templates/components": true,
	}

	// Check field types and add required imports
	for _, field := range fieldsData {
		switch field.GoType {
		case "time.Time":
			imports["time"] = true
			// TODO: expand for new usecases
		}
	}

	// Collect the list of imports from the map
	var importList []string
	for imp := range imports {
		importList = append(importList, imp)
	}

	// Define the template for the .templ scaffold
	const templateContent = `package pages

{{- if .Imports }}
import (
	{{- range .Imports }}
	"{{ . }}"
	{{- end }}
)
{{- end }}

// Template for {{.ModelNamePascal}} views

templ {{.ModelNamePascal}}Index(page *controller.Page) {
	if data, ok := page.Data.(types.{{.ModelNamePascal}}IndexData); ok {
		<h1>{{.ModelNamePascal}} Index</h1>
		<a hx-get={ page.ToURL("{{.ModelNameLower}}.create") }>Create New {{.ModelNamePascal}}</a>
		<table>
			<thead>
				<tr>
					{{range .Fields}}
					<th>{{getPascalCase .Name}}</th>
					{{end}}
					<th>Actions</th>
				</tr>
			</thead>
			<tbody>
				for _, item := range data.Items {
					<tr>
						{{range .Fields}}
						<td>{ {{toStringRepresentation "item" .GoType .Name}} }</td>
						{{end}}
						<td>
							<a hx-get={ page.ToURL("{{.ModelNameLower}}.show", item.ID) }>View</a>
							<a hx-get={ page.ToURL("{{.ModelNameLower}}.edit", item.ID) }>Edit</a>
						</td>
					</tr>
				}
			</tbody>
		</table>
	}
}

templ {{.ModelNamePascal}}Show(page *controller.Page) {
	if data, ok := page.Data.(types.{{.ModelNamePascal}}ViewData); ok {
		<h1>{{.ModelNamePascal}} Details</h1>
		{{range .Fields}}
		<p>{{.Name}}: { {{toStringRepresentation "data.Item" .GoType .Name}} }</p>
		{{end}}
		<a hx-get={ page.ToURL("{{.ModelNameLower}}.edit", data.Item.ID) }>Edit</a>
		<a hx-get={ page.ToURL("{{.ModelNameLower}}.index") }>Back to List</a>
	}
}
templ {{.ModelNamePascal}}Create(page *controller.Page) {
	<h1>Create {{.ModelNamePascal}}</h1>
	@{{.ModelNamePascal}}Form(page, "/{{.ModelNameLower}}")
}

templ {{.ModelNamePascal}}Edit(page *controller.Page) {
	if data, ok := page.Data.(types.{{.ModelNamePascal}}ViewData); ok {
		<h1>Edit {{.ModelNamePascal}}</h1>
		@{{.ModelNamePascal}}Form(page, page.ToURL("{{.ModelNameLower}}.update", data.Item.ID))
	}
}

templ {{.ModelNamePascal}}Form(page *controller.Page, route string) {
	if form, ok := page.Form.(*types.{{.ModelNamePascal}}Form); ok {
		<form 
			method="POST"
			action={ templ.URL(route) }
		>
			{{range .Fields}}
			<div>
				<label for="{{.Name}}">{{.Name}}:</label>
				<input type="{{.InputType}}" id="{{.Name}}" name="{{.Name}}" value={ {{toStringRepresentation "form" .GoType .Name}} } required />
			</div>
			{{end}}
			<button type="submit">Submit</button>
			@components.FormCSRF(page.CSRF)
		</form>
	}
}
`
	// Create a function map to pass to the template
	funcMap := template.FuncMap{
		"getPascalCase":          getPascalCase,
		"toStringRepresentation": toStringRepresentation,
	}

	// Create a template instance and parse the template content
	tmpl, err := template.New("templ").Funcs(funcMap).Parse(templateContent)
	if err != nil {
		fmt.Println("Error parsing template:", err)
		return
	}

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error creating template file:", err)
		return
	}
	defer file.Close()

	// Data for the template
	data := map[string]interface{}{
		"ModelNamePascal": modelNamePascal, // PascalCase model name
		"ModelNameLower":  strings.ToLower(modelName),
		"Fields":          fieldsData,
		"Imports":         importList,
	}

	// Execute the template with data and write to file
	err = tmpl.Execute(file, data)
	if err != nil {
		fmt.Println("Error writing to template file:", err)
		return
	}

	fmt.Println("Template generated:", filePath)
}

// Function to run the `templ generate` command
func runTemplGenerate() error {
	// Create the command
	cmd := exec.Command("templ", "generate")

	// Set the command's output to the standard output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	return cmd.Run()
}

// getPascalCase converts text to PascalCase from various formats including snake_case
func getPascalCase(text string) string {
	// First, split the text by underscores and spaces
	words := strings.FieldsFunc(text, func(r rune) bool {
		return r == '_' || unicode.IsSpace(r)
	})

	// Capitalize each word
	for i, word := range words {
		words[i] = cases.Title(language.English).String(word)
	}

	// Join the words back together
	return strings.Join(words, "")
}

// toStringRepresentation converts an interface{} to a Go code string representation based on the provided goType.
func toStringRepresentation(prefix string, goType string, name string) string {
	var formatted string
	pascalCaseName := getPascalCase(name)

	switch goType {
	case "bool":
		formatted = fmt.Sprintf("fmt.Sprintf(\"%%t\", %s.%s)", prefix, pascalCaseName)
	case "int", "int8", "int16", "int32", "int64":
		formatted = fmt.Sprintf("fmt.Sprintf(\"%%d\", %s.%s)", prefix, pascalCaseName)
	case "uint", "uint8", "uint16", "uint32", "uint64":
		formatted = fmt.Sprintf("fmt.Sprintf(\"%%d\", %s.%s)", prefix, pascalCaseName)
	case "float32", "float64":
		formatted = fmt.Sprintf("fmt.Sprintf(\"%%f\", %s.%s)", prefix, pascalCaseName)
	case "string":
		formatted = fmt.Sprintf("%s.%s", prefix, pascalCaseName) // no formatting needed for strings
	case "time.Time":
		formatted = fmt.Sprintf("%s.%s.Format(time.RFC3339)", prefix, pascalCaseName) // time format
	default:
		formatted = fmt.Sprintf("fmt.Sprintf(\"%%v\", %s.%s)", prefix, pascalCaseName) // fallback for other types
	}

	if prefix != "" {
		return formatted
	}
	return name
}
