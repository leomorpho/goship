package capabilities

import "github.com/leomorpho/goship/app/web/viewmodels"

func LandingSections() []viewmodels.CapabilitySection {
	return []viewmodels.CapabilitySection{
		newSection("routing", "Routing and Controllers", "Define routes in one canonical router and wire them to handlers that own request flow, params, responses, and page rendering.", newDoc("docs/architecture/04-http-controllers.md", "HTTP Route Map")),
		newSection("models", "Models and ORM (Bob)", "Define schema and query logic through SQL-first Bob tooling with generated, typed accessors. Keep domain logic close to your data model without ad hoc runtime SQL sprawl.", newDoc("docs/architecture/05-data-model.md", "Data Model")),
		newSection("views", "Views and Server UI", "Build interfaces with Templ and HTML-first patterns. Use HTMX for interactive flows while keeping rendering and routing server-driven.", newDoc("docs/architecture/02-structure-and-boundaries.md", "Structure and Boundaries")),
		newSection("auth", "Authentication and Authorization", "Start with practical auth defaults, session handling, and protected areas. Authorization is explicit and extendable by policy and route-level checks.", newDoc("docs/architecture/03-project-scope-analysis.md", "Project Scope Analysis")),
		newSection("migrations", "Database and Migrations", "Manage schema evolution through migrations and keep environments reproducible. External DB first, with modular support for embedded modes later.", newDoc("docs/guides/02-development-workflows.md", "Development Workflows")),
		newSection("mail", "Notifications and Mail", "Trigger transactional emails and notifications from well-defined app flows. Providers are adapter-driven and can be swapped by environment.", newDoc("docs/architecture/03-project-scope-analysis.md", "Project Scope Analysis")),
		newSection("storage", "File Storage", "Store uploads through storage adapters with S3-compatible support and URL helpers so application code stays provider-agnostic.", newDoc("docs/architecture/03-project-scope-analysis.md", "Project Scope Analysis")),
		newSection("jobs", "Jobs and Scheduling", "Run background work and scheduled tasks through a stable interface, with pluggable backends for in-process, Redis, database, or cloud queue strategies.", newDoc("docs/architecture/01-architecture.md", "Architecture")),
		newSection("testing", "Testing", "Bias toward fast table-driven unit tests, keep integration tests focused on happy-path confidence, and maintain high coverage without Docker-heavy loops.", newDoc("docs/guides/02-development-workflows.md", "Development Workflows"), newDoc("docs/policies/01-engineering-standards.md", "Engineering Standards")),
		newSection("realtime", "Events and Realtime", "Support live updates with adapter-backed pub/sub for multi-process deployments, while preserving a simpler single-node mode for local and small installs.", newDoc("docs/architecture/01-architecture.md", "Architecture"), newDoc("docs/architecture/06-known-gaps-and-risks.md", "Known Gaps and Risks")),
	}
}

func DocsSections() []viewmodels.CapabilitySection {
	return []viewmodels.CapabilitySection{
		newSection("routing", "Routing and Controllers", "Define routes from one canonical router and keep handlers focused on request parsing, orchestration, and response rendering."),
		newSection("models", "Models and ORM (Bob)", "Model entities and relations with SQL-first Bob generation, then query through typed APIs that keep data logic explicit and testable."),
		newSection("views", "Views and Server UI", "Compose HTML with Templ, add HTMX where needed, and keep page behavior server-driven for simpler flow and clearer ownership."),
		newSection("auth", "Authentication and Authorization", "Use built-in auth defaults and extend authorization policies explicitly at route and service boundaries."),
		newSection("migrations", "Database and Migrations", "Run repeatable schema migrations and keep environments aligned. Start server-db first, then opt into embedded modes when needed."),
		newSection("mail", "Notifications and Mail", "Send transactional emails and notifications through adapter-based integrations with a consistent application-facing API."),
		newSection("storage", "File Storage", "Handle uploads via storage adapters and S3-compatible providers without coupling product code to one cloud vendor."),
		newSection("jobs", "Jobs and Scheduling", "Queue background work and schedule recurring tasks through a stable interface that can target in-process, Redis, DB, or cloud backends."),
		newSection("testing", "Testing", "Prioritize fast table-driven tests, add focused integration checks, and maintain high coverage with low local setup friction."),
		newSection("realtime", "Events and Realtime", "Support live updates with pub/sub abstractions that work in distributed deployments while preserving simple local behavior."),
	}
}

func newSection(key, title, description string, docs ...viewmodels.CapabilityDocLink) viewmodels.CapabilitySection {
	section := viewmodels.NewCapabilitySection()
	section.Key = key
	section.Title = title
	section.Description = description
	section.Docs = docs
	return section
}

func newDoc(path, label string) viewmodels.CapabilityDocLink {
	doc := viewmodels.NewCapabilityDocLink()
	doc.Path = path
	doc.Label = label
	return doc
}
