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
	switch strings.ToLower(strings.TrimSpace(v)) {
	case ProductTypeFree.Value:
		p := ProductTypeFree
		return &p
	case ProductTypePro.Value:
		p := ProductTypePro
		return &p
	default:
		return nil
	}
}
