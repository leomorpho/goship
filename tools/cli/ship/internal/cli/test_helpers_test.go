package ship

import "strings"

const testDBURL = "postgres://test-user:test-pass@localhost:5432/test_db?sslmode=disable"

type fakeCall struct {
	name string
	args []string
}

type fakeRunner struct {
	calls    []fakeCall
	code     int
	err      error
	nextCode map[string]int
	nextErr  map[string]error
}

func (f *fakeRunner) Run(name string, args ...string) (int, error) {
	f.calls = append(f.calls, fakeCall{name: name, args: args})
	key := name + " " + strings.Join(args, " ")
	if err, ok := f.nextErr[key]; ok {
		return 1, err
	}
	if code, ok := f.nextCode[key]; ok {
		return code, nil
	}
	return f.code, f.err
}
