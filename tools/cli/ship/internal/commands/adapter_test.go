package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leomorpho/goship/config"
)

func TestRunAdapterSet_WritesCanonicalSelection(t *testing.T) {
	root := t.TempDir()
	envPath := filepath.Join(root, ".env")
	initial := strings.Join([]string{
		"PAGODA_RUNTIME_PROFILE=single-node",
		"PAGODA_PROCESSES_WEB=true",
		"PAGODA_PROCESSES_WORKER=true",
		"PAGODA_PROCESSES_SCHEDULER=true",
		"PAGODA_PROCESSES_COLOCATED=true",
		"PAGODA_ADAPTERS_DB=sqlite",
		"PAGODA_DATABASE_DRIVER=sqlite",
		"PAGODA_DB_DRIVER=sqlite",
		"PAGODA_DATABASE_DBMODE=embedded",
		"PAGODA_ADAPTERS_CACHE=otter",
		"PAGODA_CACHE_DRIVER=otter",
		"PAGODA_ADAPTERS_JOBS=backlite",
		"PAGODA_JOBS_DRIVER=backlite",
		"",
	}, "\n")
	if err := os.WriteFile(envPath, []byte(initial), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir %s: %v", root, err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunAdapter([]string{"set", "db=postgres", "cache=redis", "jobs=asynq"}, AdapterDeps{
		Out: out,
		Err: errOut,
		LoadConfig: func() (config.Config, error) {
			return config.GetConfig()
		},
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	body, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read env: %v", err)
	}
	for _, want := range []string{
		"PAGODA_ADAPTERS_DB=postgres",
		"PAGODA_DATABASE_DRIVER=postgres",
		"PAGODA_DB_DRIVER=postgres",
		"PAGODA_DATABASE_DBMODE=standalone",
		"PAGODA_ADAPTERS_CACHE=redis",
		"PAGODA_CACHE_DRIVER=redis",
		"PAGODA_ADAPTERS_JOBS=asynq",
		"PAGODA_JOBS_DRIVER=asynq",
	} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("env missing %q:\n%s", want, string(body))
		}
	}
	if !strings.Contains(out.String(), "adapter selection applied") {
		t.Fatalf("stdout missing summary:\n%s", out.String())
	}
}

func TestRunAdapterSet_RejectsRedisPubSubWithoutRedisCache(t *testing.T) {
	root := t.TempDir()
	envPath := filepath.Join(root, ".env")
	if err := os.WriteFile(envPath, []byte("PAGODA_ADAPTERS_CACHE=otter\n"), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir %s: %v", root, err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })

	errOut := &bytes.Buffer{}
	code := RunAdapter([]string{"set", "pubsub=redis"}, AdapterDeps{
		Out: &bytes.Buffer{},
		Err: errOut,
		LoadConfig: func() (config.Config, error) {
			return config.GetConfig()
		},
	})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(errOut.String(), "pubsub adapter \"redis\" requires cache adapter \"redis\"") {
		t.Fatalf("stderr missing pubsub/cache validation error:\n%s", errOut.String())
	}
}

func TestRunAdapterSet_RejectsDistributedInprocJobs(t *testing.T) {
	root := t.TempDir()
	envPath := filepath.Join(root, ".env")
	initial := strings.Join([]string{
		"PAGODA_RUNTIME_PROFILE=distributed",
		"PAGODA_PROCESSES_WEB=true",
		"PAGODA_PROCESSES_WORKER=true",
		"PAGODA_PROCESSES_SCHEDULER=true",
		"PAGODA_PROCESSES_COLOCATED=false",
		"PAGODA_ADAPTERS_JOBS=backlite",
		"",
	}, "\n")
	if err := os.WriteFile(envPath, []byte(initial), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir %s: %v", root, err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })

	errOut := &bytes.Buffer{}
	code := RunAdapter([]string{"set", "jobs=inproc"}, AdapterDeps{
		Out: &bytes.Buffer{},
		Err: errOut,
		LoadConfig: func() (config.Config, error) {
			return config.GetConfig()
		},
	})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(errOut.String(), "invalid runtime combination") {
		t.Fatalf("stderr missing runtime validation prefix:\n%s", errOut.String())
	}
	if !strings.Contains(errOut.String(), "invalid distributed jobs backend: inproc") {
		t.Fatalf("stderr missing distributed jobs rejection:\n%s", errOut.String())
	}
}
