package types

// CapabilityDocLink points to relevant repository documentation for a capability section.
type CapabilityDocLink struct {
	Path  string
	Label string
}

// CapabilitySection defines one item in the reusable capability explorer UI.
type CapabilitySection struct {
	Key         string
	Title       string
	Description string
	Docs        []CapabilityDocLink
}

