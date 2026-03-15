package runtime

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	appconfig "github.com/leomorpho/goship/config"
)

func ResolveDevWebURL() (string, error) {
	cfg, err := appconfig.GetConfig()
	if err != nil {
		return "", err
	}

	domain := strings.TrimSpace(cfg.HTTP.Domain)
	if domain != "" {
		return strings.TrimRight(domain, "/"), nil
	}

	scheme := "http"
	if cfg.HTTP.TLS.Enabled {
		scheme = "https"
	}

	host := strings.TrimSpace(cfg.HTTP.Hostname)
	if host == "" {
		host = "localhost"
	}

	return fmt.Sprintf("%s://%s:%d", scheme, host, cfg.HTTP.Port), nil
}

func IsInteractiveTerminal() bool {
	in, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	out, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (in.Mode()&os.ModeCharDevice) != 0 && (out.Mode()&os.ModeCharDevice) != 0
}

func PromptOpenBrowser(url string) (bool, error) {
	fmt.Printf("Open %s in your browser? [Y/n]: ", url)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer == "" || answer == "y" || answer == "yes" {
		return true, nil
	}
	return false, nil
}

func OpenBrowserURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
