# GoShip

Rails-inspired Go web framework + example app for shipping production-ready projects quickly.

Last updated: 2026-03-03

[![Test](https://github.com/leomorpho/GoShip/actions/workflows/test.yml/badge.svg)](https://github.com/leomorpho/GoShip/actions/workflows/test.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Start Here

Requirements:

- Go
- Make
- Docker

Quick start:

```bash
# web-only local dev (recommended default)
make dev

# or via CLI
go run ./cli/ship/cmd/ship dev
```

Common commands:

- `make dev`: infra + web server
- `make dev-worker`: infra + worker
- `make dev-full`: infra + all watchers
- `make test`: unit package set
- `make test-integration`: integration package set
- `make templ-gen`: generate templ Go files into sibling `gen/` dirs

## Motivation

GoShip exists because I kept re-building the same production foundation for every new project.

I want a Go-first framework that is highly ergonomic, fast to develop with, and efficient to run on small hardware. The inspiration is developer experience from frameworks like Rails and Laravel, adapted to Go + HTMX workflows.

This repository is not just a demo; it is a starter I have used repeatedly to launch real projects faster. The goal is to keep pushing that speed:

- strong conventions and batteries-included defaults
- a real `ship` CLI for repetitive/generator workflows
- LLM-first documentation + structure so humans and agents can move faster together

## Documentation

Use docs as the source of truth for architecture, workflows, and plans:

- [`docs/00-index.md`](docs/00-index.md)
- [`docs/guides/02-development-workflows.md`](docs/guides/02-development-workflows.md)
- [`docs/guides/04-deployment-kamal.md`](docs/guides/04-deployment-kamal.md)
- [`docs/reference/01-cli.md`](docs/reference/01-cli.md)
- [`docs/roadmap/01-framework-plan.md`](docs/roadmap/01-framework-plan.md)

## Repository Shape

- `app/goship/`: app-specific code (routes, views, app router)
- `pkg/`: reusable framework-level packages
- `cmd/`: process entrypoints (`web`, `worker`, `seed`)
- `cli/ship/`: standalone `ship` CLI module
- `mcp/ship/`: standalone MCP module
- `docs/`: maintained engineering documentation

## Historical Note

GoShip originally started from Pagoda and has since diverged significantly in structure and goals.
- `IsProfileFullyOnboarded`: to check if a profile is fully onboarded. This is used in the onboarding flow to check if the profile has completed the onboarding process. Edit as needed. On startup, a non-onboarded profile is redirected to the onboarding page.

Note that a method `EntProfileToDomainObject` is used to convert the ent profile object to a domain profile object, which is a more generic object that is used throughout the app. Generally, domain objects are preferred over ent objects as they are more generic and are not tied to a specific ORM.


## File Uploads

The `StorageClient` handles all the file storage logic and lives at `pkg/repos/storage/storagerepo.go`. It uses minio under the hood to handle the file uploads with AWS S3 API, which means you can easily swap out the storage backend to any S3-compatible service. 

The following methods are available:
- `CreateBucket`: to create a new bucket.
- `UploadFile`: to upload a new file.
- `DeleteFile`: to delete a file.
- `GetPresignedURL`: to get a presigned URL for a file.
- `GetImageObjectFromFile`: to get an image object from a file.
- `GetImageObjectsFromFiles`: to get image objects from a list of files.

## Paid/Free Subscriptions

The `SubscriptionsRepo` handles the subscription logic and lives at `pkg/repos/subscriptions/subscriptions.go`. It uses Stripe under the hood to handle the subscription logic. If you'd like to see the stripe webhooks, they live at `pkg/routes/payments.go`.

**Note:** currently, the only type of subscription implemented is a monthly subscription that is either paid or free. Feel free to expand on this!

The following methods are available:
- `CreateSubscription`: to create a new subscription.
- `DeactivateExpiredSubscriptions`: to deactivate all expired monthly subscriptions.
- `UpdateToPaidPro`: to update a subscription to the pro plan.
- `UpdateToFree`: to update a subscription to the free plan.
- `GetCurrentlyActiveProduct`: to get the currently active product for a profile.
- `CancelWithGracePeriod`: to cancel a subscription with a grace period.
- `CancelOrRenew`: to cancel a subscription or renew it.


## Regenerate Logo Image Assets 

There is a python script in `scripts/regen_logo_images.py` that should be run when the logo in `static/logo.png` is updated. 
This will regenerate the logo assets for different app icons and the favicon. It will also regenerate the correct iOS and Android app icons and place them in the `static/ios-wrapper/` and `static/android-wrapper/` directories. Note that for iOS it will remove alpha transparency and make the background black (as apple requires).

```bash
cd scripts
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt

python3 scripts/regen_logo_images.py
```

## Run Tasks

Currently, tasks are run using [asynq](https://github.com/hibiken/asynq). This unfortunately requires [redis](https://redis.io/) to be running. This can make deployment a bit trickier as it means you will need at least 3 VPS with Kamal (except if I'm missing something), as you will need one for the web app, one for the worker, and one for the cache/queue. This is far from ideal for small projects, and [pagoda](https://github.com/leomorpho/goship)'s author decided to use [backlite](https://github.com/mikestefanello/backlite), a tool he created to use SQLite as the task queue. I have not gone around to pulling these changes in yet, and I am hesitant at this point as I have multiple projects running in prod, and only 1 VPS running a cache that is serving all my projects...which means that I don't have a huge incentive to add this in. 

If you'd like to change asynq to backlite, you can refer [to this pagoda PR](https://github.com/leomorpho/goship/pull/72/files) to bring the changes in your goship instance.

## Drop in any JS App

While the project primarily uses HTMX, it also supports integrating JavaScript applications. The current build process creates two separate bundles:
1. A single Vanilla JavaScript bundle
2. A single Svelte bundle
This approach allows you to incorporate JavaScript functionality alongside the HTMX-driven parts of your application. Here's how it works:
- The build.mjs script handles the bundling process for both Vanilla JS and Svelte components.
-Each framework (Vanilla JS and Svelte) is compiled into its own single file bundle.
-These bundles can be served to the frontend and used where needed in your application.

**Note:** While this method allows for easy integration, it does come with the trade-off of potentially large bundle sizes. Future improvements could involve optimizing the build process to create smaller, component-specific bundles for more efficient loading.

This setup provides flexibility to use JavaScript frameworks alongside HTMX, leveraging the strengths of both approaches in different parts of your application.

Note that any JS framework could be used.

**Note:** Svelte is used for highly interactive components, although I've come to regret this as it is a large framework to bundle and slow down the initial page load. In the future, I plan to remove Svelte and only use HTMX for all components. This would not impact the ability to drop in any JS app, however.

## Playwright E2E Tests

TODO: the test file can be found at `e2e_tests/tests/goship.spec.ts` and is currently still the one from [chérie](https://cherie.chatbond.app/)...I will update it soon!

You can run the Playwright tests with:
```bash
make e2eui
```

NOTE: on older/slower machines, the tests may time out. If so, you can increase the timeout in the test file. I was facing that issue when testing locally on a 2014 Macbook Pro, though have not faced it since running the tests on my M2 Mac. I am no playwright expert too, so perhaps I am missing something.

## Deployment

I currently only use Kamal for deployment. Should you want to contribute in adding other deployment methods, please create a subdirectory in `deploy` and add it there, so that it's well organized.

### Kamal

First, make sure all your env vars in the Kamal file `deploy/kamal/deploy.yml` are correct. All your vars should be set either in:
 
- `config/config.yml`: only non-secret ones
- `deploy/kamal/deploy.yml`: only non-secret ones
- `.env`: all secret vars

Then, set the IP of the server host in `deploy/kamal/deploy.yml`, as well as your image and registry details. Read up on the [kamal documentation](https://kamal-deploy.org) if you get stuck anywhere here. 

### Set up live server

The below command will install docker, build your image, push it to your registry, and then pull it on your remote VPS. If you set up any accessory (cache, standalone DB that is not hosted etc), these will also be deployed.

```bash
kamal setup -c deploy/kamal/deploy.yml
``` 

At this point, your project should be live, and if `128.0.0.1111` is the IP of your VPS, entering that IP in the search bar on your browser should bring up your site.

### HTTPS

Hop into your VPS console.

```bash
mkdir -p /letsencrypt && touch /letsencrypt/acme.json && chmod 600 /letsencrypt/acme.json
```

Then locally, run

```bash
kamal traefik reboot -c deploy/kamal/deploy.yml
```

Your site should now have TLS enabled and you should see the lock icon the search bar.

For reference, the above procedure was taken from [this Kamal issue](https://github.com/basecamp/kamal/discussions/112).

### Firewall

There are some sample firewall scripts in `config/firewalls/` to help you get started. They make use of `ufw` so make sure that is installed on your system. 

The worker firewall should block all ports by default except for SSH and internal network traffic.

The web app firewall should block all ports by default except for SSH, HTTPS, and internal network traffic.

The accessories firewall should block all ports by default, though if using Asynq, you should allow 8080 (Asynq UI) to your specific IP.

## Future Work

### Environment Management

Improve the experience with handling config and environment variables. Currently, there is an `.env` file with secrets, which can be of the form `PAGODA_STORAGE_S3ACCESSKEY=123` and then in `config.yml` it is under:
```yml
pagoda:
  storage:
    s3accesskey: 123
```

And in `config.go` it is defined as:
```go
type Config struct {
  Pagoda struct {
    Storage struct {
      S3accesskey string 
    }
  }
}
```

This is fine for simple cases but can quickly be confusing. It would be nice to have a more robust env management system, perhaps one that can auto-generate env vars for you in the `.env` and `config.yml` files, so that no human error is introduced, leading the developer to confusion trying to figure out what is going on.

### Code generation

#### Scaffold Generator

This is a CLI command that will generate a route, model, and view for you. It's just at the idea stage and would be a great feature to have. A lot of time goes into writing boilerplate code for each new route. Ideally, it supports generating templ/htmx routes and JSON routes.

Example:

```bash
ship make:scaffold Post title:string content:text --migrate
```

##### Generated Model:
```go
// ent/schema/post.go
package schema

type Post struct {
  ent.Schema
}

func (Post) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

func (Post) Fields() []ent.Field {
  return []ent.Field{
		field.String("title"),
    field.String("content"),
	}
}

func (UsPoster) Edges() []ent.Edge {
	return nil
}
```

##### Route
```go
// app/controllers/post_controller.go
package routes

type postRoute struct {}

func NewPostRoute(
	ctr controller.Controller,
) postRoute {
	return postRoute{
		ctr:                            ctr,
	}
}

func (p *postRoute) Index(ctx echo.Context) {
    // List all posts
}

func (p *postRoute) Show(ctx echo.Context) {
    // Show a specific post
}

func (p *postRoute) New(ctx echo.Context) {
    // Render form to create new post
}

func (p *postRoute) Create(ctx echo.Context) {
    // Logic to create a new post
}

func (p *postRoute) Edit(ctx echo.Context) {
    // Render form to edit post
}

func (p *postRoute) Update(ctx echo.Context) {
    // Logic to update a post
}

func (p *postRoute) Destroy(ctx echo.Context) {
    // Logic to delete a post
}
```

##### Generated Routes:

The routes will be automatically added to the router:
```go
postRoute := NewPostRoute(ctr)
g.GET("/posts", postRoute.Index).Name = "posts.index"
g.GET("/posts/:id", postRoute.Show).Name = "posts.show"
g.GET("/posts/new", postRoute.New).Name = "posts.new"
g.POST("/posts", postRoute.Create).Name = "posts.create"
g.GET("/posts/:id/edit", postRoute.Edit).Name = "posts.edit"
g.POST("/posts/:id", postRoute.Update).Name = "posts.update"
g.DELETE("/posts/:id", postRoute.Destroy).Name = "posts.destroy"
```

##### Generated views
```go
// app/goship/views/posts.templ
package pages

import (
	"github.com/leomorpho/goship/app/goship/controller"
	"github.com/leomorpho/goship/app/goship/types"
	"github.com/leomorpho/goship/app/goship/views/web/components"
)

templ PostsIndex(page *controller.Page) {
}

templ PostsShow(page *controller.Page) {
}

templ PostsNew(page *controller.Page) {
}

templ PostsEdit(page *controller.Page) {
}

templ PostsCreate(page *controller.Page) {
}

templ PostsUpdate(page *controller.Page) {
}

templ PostsDestroy(page *controller.Page) {
}
```
##### Generate Type Data Struct

```go
// types/post.go
package types

type Post struct {
  Title string
  Content string
}
```
