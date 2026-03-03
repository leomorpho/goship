package capabilities

import "github.com/leomorpho/goship/app/goship/types"

func LandingSections() []types.CapabilitySection {
	return []types.CapabilitySection{
		{
			Key:         "routing",
			Title:       "Routing and Controllers",
			Description: "Define routes in one canonical router and wire them to handlers that own request flow, params, responses, and page rendering.",
			Docs: []types.CapabilityDocLink{
				{Path: "docs/architecture/04-http-routes.md", Label: "HTTP Route Map"},
			},
		},
		{
			Key:         "models",
			Title:       "Models and ORM (Ent)",
			Description: "Use Ent schemas to model entities, relationships, and constraints. Keep domain logic close to your data model without ad hoc SQL everywhere.",
			Docs: []types.CapabilityDocLink{
				{Path: "docs/architecture/05-data-model.md", Label: "Data Model"},
			},
		},
		{
			Key:         "views",
			Title:       "Views and Server UI",
			Description: "Build interfaces with Templ and HTML-first patterns. Use HTMX for interactive flows while keeping rendering and routing server-driven.",
			Docs: []types.CapabilityDocLink{
				{Path: "docs/architecture/02-structure-and-boundaries.md", Label: "Structure and Boundaries"},
			},
		},
		{
			Key:         "auth",
			Title:       "Authentication and Authorization",
			Description: "Start with practical auth defaults, session handling, and protected areas. Authorization is explicit and extendable by policy and route-level checks.",
			Docs: []types.CapabilityDocLink{
				{Path: "docs/architecture/03-project-scope-analysis.md", Label: "Project Scope Analysis"},
			},
		},
		{
			Key:         "migrations",
			Title:       "Database and Migrations",
			Description: "Manage schema evolution through migrations and keep environments reproducible. External DB first, with modular support for embedded modes later.",
			Docs: []types.CapabilityDocLink{
				{Path: "docs/guides/02-development-workflows.md", Label: "Development Workflows"},
			},
		},
		{
			Key:         "mail",
			Title:       "Notifications and Mail",
			Description: "Trigger transactional emails and notifications from well-defined app flows. Providers are adapter-driven and can be swapped by environment.",
			Docs: []types.CapabilityDocLink{
				{Path: "docs/architecture/03-project-scope-analysis.md", Label: "Project Scope Analysis"},
			},
		},
		{
			Key:         "storage",
			Title:       "File Storage",
			Description: "Store uploads through storage adapters with S3-compatible support and URL helpers so application code stays provider-agnostic.",
			Docs: []types.CapabilityDocLink{
				{Path: "docs/architecture/03-project-scope-analysis.md", Label: "Project Scope Analysis"},
			},
		},
		{
			Key:         "jobs",
			Title:       "Jobs and Scheduling",
			Description: "Run background work and scheduled tasks through a stable interface, with pluggable backends for in-process, Redis, database, or cloud queue strategies.",
			Docs: []types.CapabilityDocLink{
				{Path: "docs/architecture/01-architecture.md", Label: "Architecture"},
			},
		},
		{
			Key:         "testing",
			Title:       "Testing",
			Description: "Bias toward fast table-driven unit tests, keep integration tests focused on happy-path confidence, and maintain high coverage without Docker-heavy loops.",
			Docs: []types.CapabilityDocLink{
				{Path: "docs/guides/02-development-workflows.md", Label: "Development Workflows"},
				{Path: "docs/policies/01-engineering-standards.md", Label: "Engineering Standards"},
			},
		},
		{
			Key:         "realtime",
			Title:       "Events and Realtime",
			Description: "Support live updates with adapter-backed pub/sub for multi-process deployments, while preserving a simpler single-node mode for local and small installs.",
			Docs: []types.CapabilityDocLink{
				{Path: "docs/architecture/01-architecture.md", Label: "Architecture"},
				{Path: "docs/architecture/06-known-gaps-and-risks.md", Label: "Known Gaps and Risks"},
			},
		},
	}
}

func DocsSections() []types.CapabilitySection {
	return []types.CapabilitySection{
		{Key: "routing", Title: "Routing and Controllers", Description: "Define routes from one canonical router and keep handlers focused on request parsing, orchestration, and response rendering."},
		{Key: "models", Title: "Models and ORM (Ent)", Description: "Model entities and relations using Ent schemas, then query through typed APIs that keep data logic explicit and testable."},
		{Key: "views", Title: "Views and Server UI", Description: "Compose HTML with Templ, add HTMX where needed, and keep page behavior server-driven for simpler flow and clearer ownership."},
		{Key: "auth", Title: "Authentication and Authorization", Description: "Use built-in auth defaults and extend authorization policies explicitly at route and service boundaries."},
		{Key: "migrations", Title: "Database and Migrations", Description: "Run repeatable schema migrations and keep environments aligned. Start server-db first, then opt into embedded modes when needed."},
		{Key: "mail", Title: "Notifications and Mail", Description: "Send transactional emails and notifications through adapter-based integrations with a consistent application-facing API."},
		{Key: "storage", Title: "File Storage", Description: "Handle uploads via storage adapters and S3-compatible providers without coupling product code to one cloud vendor."},
		{Key: "jobs", Title: "Jobs and Scheduling", Description: "Queue background work and schedule recurring tasks through a stable interface that can target in-process, Redis, DB, or cloud backends."},
		{Key: "testing", Title: "Testing", Description: "Prioritize fast table-driven tests, add focused integration checks, and maintain high coverage with low local setup friction."},
		{Key: "realtime", Title: "Events and Realtime", Description: "Support live updates with pub/sub abstractions that work in distributed deployments while preserving simple local behavior."},
	}
}
