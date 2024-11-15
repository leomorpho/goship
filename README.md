## GoShip: Ship in Record Time ⛵️🛟⚓️📦

### A Go + HTMX boilerplate with all the essentials for your SaaS, AI tools, or web apps. Start earning online quickly without the hassle.

🎯 **The goal of this project** is to build the most comprehensive Go-centric OSS starter boilerplate to ship projects fast.

<!-- [![Go Report Card](https://goreportcard.com/badge/github.com/mikestefanello/pagoda)](https://goreportcard.com/report/github.com/mikestefanello/pagoda) -->
[![Test](https://github.com/leomorpho/GoShip/actions/workflows/test.yml/badge.svg)](https://github.com/leomorpho/GoShip/actions/workflows/test.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)


<p align="center"><img alt="Logo" src="https://goship-static.s3.us-west-002.backblazeb2.com/assets/goship.png" height="200px"/></p>

<p align="center">
  <a href="http://www.youtube.com/watch?feature=player_embedded&v=Mnti8f-4bp0" target="_blank"><img src="https://goship-static.s3.us-west-002.backblazeb2.com/assets/git-repo-video-overview-frame.jpg" 
  alt="Rapid walktrough of project" style="max-width: 100%; height: auto; border: 10px;" /></a>
</p>

<p style="text-align:center;">Check out the video above for a rapid walkthrough of the project! 🏂</p>

This started as a fork of [pagoda](https://github.com/mikestefanello/pagoda), for which I am super grateful! Big shoutout to Mike Stefanello and team!
<p align="center"><img alt="Logo" src="https://user-images.githubusercontent.com/552328/147838644-0efac538-a97e-4a46-86a0-41e3abdf9f20.png" height="100px"/></p>

### Getting Started

Make sure you have `make` and Golang installed on your machine.

To get up and running with GoShip:
```bash
# The below command will:
# - set up the postgres/redis/mailer containers
# - build the JS/CSS assets 
# - seed the DB with test users
# - start the project in watch mode
make init

# Running init will fully scrap your state and start with fresh new containers. 
# After running `make init` the first time, just use the below for everyday work.
make watch
```

For in-depth info on the architecture of the project, please see the [mikestefanello/pagoda](https://github.com/mikestefanello/pagoda) repo. There are some key differences, but since this was originally a fork, 99% of it still applies. I am working on creating clear and actionable documentation, but that is quite time-consuming, so don't hold your socks.

### Motivation

Build the same rich interfaces you would build with Javascript frameworks, but with HTML and Go. Limit the number of tools you use. Develop rapidly.

#### Why the Hell Do We Need Another Boilerplate?

Well, I noticed that there were none for Go. Now, I know most Go folks like to build it all themselves. And while I love doing that myself, I have many project ideas for which I just want to build that specific project, not the entire infra surrounding it, like auth, notifications, payments, file uploads etc. This project has served me well in bringing to production many projects so far. It has evolved far beyond what I originally planned for, though there is still so much potentional to expand on and implement for.

If you'd like a no-nonesense (or not too much?) starter kit to get your next project to production ASAP, while also using awesome technologies like Go, you've found a suitable starting point!

> **Warning alert!** this project is in active development as I am adding things after first trying them out in prod for [Chérie](https://cherie.chatbond.app/), a relationship app to grow your couple. Note that I would welcome any help to develop this boilerplate ❤️.

### Features && Tech Stack
 
See [goship.run](https://goship.run). 

---

# Documentation (WIP)

## File Structure
```bash
|-- cmd
|   |-- web # Web server
|   |-- worker # Async worker
|   |-- seed # Seeder
|-- config # Config files where the non-secret config vars are stored and the config go struct is defined
|-- pkg # Package imports
|   |-- context # Context package to handle context across the app
|   |-- controller # Controller package to handle requests and responses
|   |-- domain # Domain objects that are used throughout the app, these should be specific to your app/project
|   |-- funcmap # Custom template functions
|   |-- htmx # HTMX lifecycle helpers
|   |-- middleware # Middleware for the app
|   |-- repos # Repositories
|   |-- routes # Think of these as the controllers in a traditional MVC framework
|   |-- services # Services on the Container struct
|   |-- tasks # Task definitions
|   |-- tests # Utility functions for testing
|   |-- types # Struct types
|-- templates # HTML templates
|-- ent # Ent ORM, contains the schema for the DB as well as the generated code from the schema. Always commit this to git.

# Everything else is not Go-specific
|-- deploy.yml # Kamal deployment file
|-- docker-compose.yml # Docker compose file for running the project locally with Docker Desktop
|-- e2e_tests # Playwright E2E tests
|-- scripts # Useful scripts
|-- static # Static files
|-- javascript # Any javascript app can be dropped here. JS and CSS will be built and bundled into a single file. It is currently set up solely for Vanilla JS and Svelte.
|-- build.mjs # Build script for the JS defined in `./javascript` 
|-- .env # Secret environment variables
|-- .kamal # Kamal hooks you can use to run commands in the project during deployment
|-- .github # Github actions and secrets
|-- .gitignore # Files to ignore when committing
|-- tailwind.config.js # Tailwind config
|-- tsconfig.json # Typescript config 
|-- Procfile # Defines the commands to run the project in watch mode
|-- service-worker.js # Service worker for the PWA
|-- pwabuilder-ios-wrapper # PWA iOS wrapper. Use as a guide for push notifications. 
```

## Makefile

The Makefile is the main entry point for the project. It is used to build the project, run the project, and deploy the project.

The following commands are the most useful ones:
```bash
make init # Initializes the project
make watch # Runs the project in watch mode, rebuilding assets as you go (JS, CSS, Templ, etc)
make test # Runs the tests
make e2eui # Runs the interactive e2e tests with Playwright # 
make cover # Shows a Go coverage report of the tests

# DB specific commands
make ent-new name=YourModelName # Creates a new ent schema file
make makemigrations # Creates a new migration file
make ent-gen # Generates the ent code from the schema
make migrate # Applies migrations
make inspecterd # Shows you a view of all your tables in a UI
make schema

# Docker commands
make up # Starts the docker containers
make down # Stops the docker containers
make down-volume # Stops the docker containers and removes the volumes
make reset # Stops the docker containers and removes the volumes, then rebuilds the docker containers

# Assets
make build-js # Builds the JS assets
make watch-js # Watches the JS assets and rebuilds them on change
make build-css # Builds the CSS assets
make watch-css # Watches the CSS assets and rebuilds them on change

# Worker commands
make worker # Starts the worker
make worker-ui # Will open the terminal to the asynq worker UI

# Stripe (payments)
make stripe-webhook # Sets up a webhook for stripe for local testing

make help # Shows all the commands you can run
```

## General Architecture

For in-depth info on the architecture of the project, please see the [mikestefanello/pagoda](https://github.com/mikestefanello/pagoda) repo. There are some key differences, but since this was originally a fork, 99% of it still applies. 

The most important aspects to note are:
- The `Container` struct is instantiated when the app starts up and is used to pass dependencies around the app, specifically core services like `Logger`, `Database`, `ORM`, `Cache`, etc.
- Routes are defined in `routes/routes.go` and are registered to the `Echo` framework. Generally, any logic that alters the DB should be done in the `repos` layer so that it is easily testable, and can be used by other routes. A route will generally have a `Component`, which is a Templ component defined in `templates/pages/` that represents the view.

## Database

The current options are:
- Standalone Postgres DB (which you can host anywhere, including locally with Docker)
  - For free deployments, see [Supabase](https://supabase.com/pricing) or [Neon](https://neon.tech/pricing). There are also other free options available, and if you host each one of your projects on a different DB, you can use the free tier for all your projects!
- Embedded SQLite DB (which is great for small projects and local development)
  

## Starting DB State

To get a look at what tables are available to start off, you can run 

```bash
make schema
```

or go to `ent/schema` and see the declared schemas. Note that ent generates a lot of code. Do not remove it from git. In fact, make sure to keep it there. 

To create a new schema, do:
```bash
make ent-new name=YourSchemaName
```

Then generate the migrations
```bash
make makemigrations
```

Then generate the ent generated code to interact with your new schema in Go:
```bash
make ent-gen
```

To apply the migrations, either run `make migrate` or do a `make reset` to start from scratch (often times easier, and your test DB should be treated as disposable).


## Add a route

Create a new file in `routes/` and add your route. A route is a standard Echo handler with some added goodies. Once you've added handlers for your route, you can hook it up to the router in `routes/routes.go`, where the route should be registered to be reachable from the web.

## Set Action Messages

Following an action (POST/DELETE/GET/etc), a msg can be shown to the user. For example, a success message can shown with `msg.Success("An email confirmation was sent!")` upon user registration. The following message types are currently available:
- success
- info
- warning
- danger

See `pkg/repos/msg/msg.go` for more info.

## Realtime and Notifications

There is a `realtime` route that is setup to handle SSE connections to any client desiring real-time data. Realtime data is sent in "notifications" which are just custom events with a notification type, some data, and a profile id. The `NotifierRepo` handles subscribing the client to the right channels and pushing new notifications to the client. Notifications can be stored in the DB in case the client is offline and needs to be picked up later when they reconnect - these will be shown in the notification center UI.

Methods for interacting with notifications:

- `PublishNotification` to send a notification to a user. This can optionally store the notification in the DB.
- `MarkNotificationUnread` to mark a notification as unread.
- `MarkNotificationRead` to mark a notification as read.
- `DeleteNotification` to delete a notification.
- `GetNotifications` to get all notifications for a user.

Note that actual storage of notifications in the DB is handled by `NotificationStorageRepo`.

## Notification Permissions

The `NotificationSendPermissionRepo` handles the permission logic for sending notifications to a user. It is used to determine if a user has granted permission to send notifications to them and lives at `pkg/repos/notifierrepo/permissions.go`.
You can mostly leave this alone, but if you need to add a new permission platform (e.g. a new push notification service), you may need to add a new permission here.

## Planned Notifications

The `PlannedNotificationsRepo` handles the logic for sending notifications at a planned time and lives at `pkg/repos/notifierrepo/planned_notifications.go`. The repo does not send any notifications, but rather sets up the DB storage for scheduled notifications. It also contains a method to clean up old notifications. But both the sending and deletion methods need to be called as tasks. Two examples are the `TypeAllDailyConvoNotifications` and `TypeEmailUpdates` tasks, as well as the `TypeDeleteStaleNotifications` task, which are commented out in the `cmd/web/main.go` file.

The algorithm used to determine best time to send notifications is very primitive. Feel free to improve it! (or I will eventually, though it's low priority)

## PWA Notifications

There are 2 push notification repos for different use cases: 
- `PwaPushNotificationsRepo`: for sending push notifications to PWAs.
- `FcmPushNotificationsRepo`: for sending push notifications to native Android and iOS apps.

Both have similar interfaces:
- `AddPushSubscription`: to add a new push subscription, triggered when the profile turns on PWA notifications in their profile settings.
- `SendPushNotifications`: to send a push notification to a user. This is generally handled by the `NotifierRepo` after storing a notification in the DB using the `PublishNotification` method. 
- `DeletePushSubscriptionByEndpoint`: to delete a push subscription by endpoint.

## Profile Repo

The `ProfileRepo` handles all the profile logic and lives at `pkg/repos/profilerepo/profilerepo.go`. It contains basic CRUD methods for profiles, as well as some helper methods for getting friends, updating profile info, etc.

There is extensive "friendship" logic in the repo, which is currently not used in the app. It is left over from [Chérie](https://cherie.chatbond.app/) as a demo. Feel free to delete these methods if you don't need them!

- `GetFriends`: to get all friends for a profile. This is a demo as there is no friends feature in the app.
- `AreProfilesFriends`: to check if two profiles are friends. This is a demo as there is no friends feature in the app.
- `LinkProfilesAsFriends`: to link two profiles as friends. This is a demo as there is no friends feature in the app.
- `UnlinkProfilesAsFriends`: to unlink two profiles as friends. This is a demo as there is no friends feature in the app.
- `GetProfileByID`: to get a profile by ID.
- `GetCountOfUnseenNotifications`: to get the count of unseen notifications for a profile.
- `GetPhotosByProfileByID`: to get the photos for a profile by ID.
- `GetProfilePhotoThumbnailURL`: to get the thumbnail URL for a profile's photo by ID.
- `SetProfilePhoto`: to set the profile photo for a profile by ID.
- `UploadPhoto`: to upload a photo for a profile by ID.
- `UploadImageSizes`: to upload image sizes for a photo by ID.
- `DeletePhoto`: to delete a photo by ID.
- `DeleteUserData`: to delete a user's data by ID. This should be updated to delete all new models that may not cascade delete and is used in the settings to delete a user's data and account.
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

Currently, tasks are run using [asynq](https://github.com/hibiken/asynq). This unfortunately requires [redis](https://redis.io/) to be running. This can make deployment a bit trickier as it means you will need at least 3 VPS with Kamal (except if I'm missing something), as you will need one for the web app, one for the worker, and one for the cache/queue. This is far from ideal for small projects, and [pagoda](https://github.com/mikestefanello/pagoda)'s author decided to use [backlite](https://github.com/mikestefanello/backlite), a tool he created to use SQLite as the task queue. I have not gone around to pulling these changes in yet, and I am hesitant at this point as I have multiple projects running in prod, and only 1 VPS running a cache that is serving all my projects...which means that I don't have a huge incentive to add this in. 

If you'd like to change asynq to backlite, you can refer [to this pagoda PR](https://github.com/mikestefanello/pagoda/pull/72/files) to bring the changes in your goship instance.

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
goship generate scaffold Post title:string content:text
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
// templates/posts.templ
package pages

import (
	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/pkg/types"
	"github.com/mikestefanello/pagoda/templates/components"
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


