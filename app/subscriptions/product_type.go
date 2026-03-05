package subscriptions

import (
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/framework/domain"
)

func ToDomainProductType(pt *paidsubscriptions.ProductType) *domain.ProductType {
	if pt == nil {
		return nil
	}
	switch pt.Value {
	case paidsubscriptions.ProductTypePro.Value:
		p := domain.ProductTypePro
		return &p
	default:
		p := domain.ProductTypeFree
		return &p
	}
}
