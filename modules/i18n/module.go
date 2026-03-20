package i18n

type Module struct {
	service *Service
}

func NewModule(service *Service) *Module {
	return &Module{service: service}
}

func (m *Module) ID() string {
	return "i18n"
}

func (m *Module) Service() *Service {
	if m == nil {
		return nil
	}
	return m.service
}
