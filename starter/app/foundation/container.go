package foundation

type Container struct {
	AppName        string
	EnabledModules []string
}

func NewContainer() *Container {
	return &Container{
		AppName:        "GoShip Starter",
		EnabledModules: []string{"auth", "profile"},
	}
}

func (c *Container) SupportsModule(name string) bool {
	for _, enabled := range c.EnabledModules {
		if enabled == name {
			return true
		}
	}
	return false
}
