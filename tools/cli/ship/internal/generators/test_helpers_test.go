package generators

import "os"

type fakeCall struct {
	name string
	args []string
}

type fakeRunner struct {
	calls []fakeCall
	code  int
}

func (f *fakeRunner) RunCode(name string, args ...string) int {
	f.calls = append(f.calls, fakeCall{name: name, args: args})
	return f.code
}

func testHasFile(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
