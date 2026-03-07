package paidsubscriptions

import "strings"

// ProductType is the module-local plan type to keep module boundaries isolated.
type ProductType struct {
	Value string
}

var (
	ProductTypeFree = ProductType{Value: "free"}
	ProductTypePro  = ProductType{Value: "pro"}
)

func ParseProductType(v string) *ProductType {
	key := strings.ToLower(strings.TrimSpace(v))
	if key == "" {
		return nil
	}
	p := ProductType{Value: key}
	return &p
}
