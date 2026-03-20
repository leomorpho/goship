package runtime

import (
	"errors"
	"os/exec"
)

func ResolveComposeCommand() ([]string, error) {
	return ResolveComposeCommandWith(exec.LookPath, func() error {
		cmd := exec.Command("docker", "compose", "version")
		return cmd.Run()
	})
}

func ResolveComposeCommandWith(lookPath func(string) (string, error), dockerComposeVersion func() error) ([]string, error) {
	if _, err := lookPath("docker-compose"); err == nil {
		return []string{"docker-compose"}, nil
	}
	if _, err := lookPath("docker"); err == nil {
		if err := dockerComposeVersion(); err == nil {
			return []string{"docker", "compose"}, nil
		}
	}
	return nil, errors.New("no docker compose command found (docker-compose or docker compose)")
}
