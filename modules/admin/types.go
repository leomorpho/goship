package admin

type FieldType string

const (
	FieldTypeString   FieldType = "string"
	FieldTypeInt      FieldType = "int"
	FieldTypeBool     FieldType = "bool"
	FieldTypeTime     FieldType = "time"
	FieldTypeText     FieldType = "text"
	FieldTypeEmail    FieldType = "email"
	FieldTypePassword FieldType = "password"
	FieldTypeReadOnly FieldType = "readonly"
)

type AdminField struct {
	Name      string
	Label     string
	Type      FieldType
	Value     any
	Required  bool
	Sensitive bool
}

type AdminResource struct {
	Name       string
	PluralName string
	TableName  string
	Fields     []AdminField
	IDField    string
}

type AdminRow map[string]any

type Pager struct {
	Page    int
	PerPage int
	Total   int
}
