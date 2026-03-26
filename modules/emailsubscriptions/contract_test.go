package emailsubscriptions

import "testing"

func TestContract_HasCanonicalSurfaces(t *testing.T) {
	c := Contract()
	if len(c.Config) == 0 || len(c.Migrations) == 0 || len(c.Jobs) == 0 || len(c.Routes) == 0 {
		t.Fatalf("emailsubscriptions contract missing required surfaces: %#v", c)
	}
}
