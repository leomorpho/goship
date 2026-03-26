package storage

import "testing"

func TestContract_HasCanonicalSurfaces(t *testing.T) {
	c := Contract()
	if len(c.Config) == 0 || len(c.Migrations) == 0 || len(c.Jobs) == 0 {
		t.Fatalf("storage contract missing required surfaces: %#v", c)
	}
}
