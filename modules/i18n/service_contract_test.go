package i18n

import (
	"testing"

	"github.com/leomorpho/goship/framework/core"
)

var _ core.I18n = (*Service)(nil)

func TestServiceImplementsCoreI18nContract(t *testing.T) {}
