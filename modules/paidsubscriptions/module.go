package paidsubscriptions

// New is the module entrypoint used by app wiring.
func New(store Store) *Service {
	return NewService(store)
}
