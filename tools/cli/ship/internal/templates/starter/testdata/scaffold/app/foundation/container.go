package foundation

type Container struct {
	AppName        string
	EnabledModules []string
}

func NewContainer() *Container {
	c := &Container{
		AppName:        "GoShip Starter",
		EnabledModules: []string{"auth", "profile"},
	}
	// ship:container:start
	// ship:container:end
	return c
}

func (c *Container) SupportsModule(name string) bool {
	for _, enabled := range c.EnabledModules {
		if enabled == name {
			return true
		}
	}
	return false
}
