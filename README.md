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

> **Warning alert!** this project is in active development as I am adding things after first trying them out in prod for [Goship](https://cherie.chatbond.app/), a relationship app to grow your couple. Note that I would welcome any help to develop this boilerplate ❤️.

### Features

#### 🌩 Realtime
- Support for HTMX SSE extension
- Can be used with vanilla JS

#### 📬 Email Sending
- Support for SMTP and Resend API
- Pre-made templates for account activation, password reset, and newsletter.

#### 💸 Payments
- Stripe integration for monthly subscriptions
- Internal subscription management

#### 🏗 Background Tasks
- Offload heavy tasks to background
- Realtime or scheduled

#### 🔔 Notifications
- Real-time or scheduled
- Supports push notifications to PWA, native iOS, and native Android

#### 🔐 Auth Done For You
- Email/Password logins
- Ready-made private user area

#### 📂 File Uploads with AWS APIs
- Internal management of uploaded files
- Host files and images on any S3 compatible service (e.g. Backblaze)
- Pre-signed URLs!

#### 📱 Mobile Ready App
- Fully PWA-ready with internal FCM and push subscriptions management
- IOS native wrapper with push notifications and payments
- Pre-signed URLs!
- Styled with mobile/tablet/desktop in mind

#### 💅 Components and Styles
- Light + Dark mode
- Many components available (HTMX, AlpineJS, Hyperscript)
- 20+ themes with DaisyUI

#### 🪂 Drop-in any JS App
- Designed for island architecture. Drop in any JS app and take advantage of already built infra
- Currently has SvelteJS and VanillaJS build step and static file serving

#### 🛢 AI-ready Database Layer
- Postgres support (i.e. Supabase, Neon etc)
- Vector-ready (PGVector integrated) for your AI/ML applications!

#### 🧪 Go Tests and E2E Tests with Playwright
- Go tests with automatic setup/teardown of DB container
- Playwright integration tests to make sure you don't break your previously working UIs!

#### 🚀 Deploy Anywhere. Easily.
- Deploy from bare metal to Cloud VMs with Kamal
- Single-command deploy after quick setup

### Tech Stack

#### Backend
- **[Echo](https://echo.labstack.com/)** - High-performance, extensible, minimalist Go web framework.
- **[Ent](https://entgo.io/)** - Simple yet powerful ORM for modeling and querying data.
- **[Asynq](https://github.com/hibiken/asynq)** - Simple, reliable, and efficient distributed task queue in Go.
- **[Stripe](https://stripe.com/)** - Payments solution.

#### Frontend
- **[HTMX](https://htmx.org/)** - Build modern user interfaces with minimal JavaScript.
- **[Templ](https://templ.build/)** - A powerful type-safe Go templating language.
- **[Tailwind CSS](https://tailwindcss.com/)** - A utility-first CSS framework for rapid implementation.
- **[Hyperscript](https://hyperscript.org/)** - A lightweight JavaScript framework to sprinkle localized logic and state.
#### Storage
- **Postgres** - Host your DB on Supabase or any other hosting platform compatible with Postgres.
  - Currently making optional, with **SQLite** as replacement for single binary deployments
- **[S3](https://aws.amazon.com/s3/)** - Host files and images on any S3-compatible service (e.g., Backblaze). 
- **Redis** - used for task queuing, caching, and SSE events.
  - Currently making optional for single binary deployments

## WIP Documentation
 
See [goship.run](https://goship.run). NOTE: it's currently being actively developed! Feel free to help ❤️.

---

# Temporary Documentation

This documentation will eventually be moved to [goship.run](https://goship.run).

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


## Add a route

Create a new file in `routes/` and add your route. A route is a standard Echo handler with some added goodies. Once you've added handlers for your route, you can hook it up to the router in `routes/routes.go`, where the route should be registered to be reachable from the web.


## Realtime

There is a `realtime` route that is setup to handle SSE connections to any client desiring real-time data. Realtime data is sent in "notifications" which are just custom events with a notification type, some data, and a profile id. The `NotifierRepo` handles subscribing the client to the right channels and pushing new notifications to the client. Notifications can be stored in the DB in case the client is offline and needs to be picked up later when they reconnect - these will be shown in the notification center UI.

Methods for interacting with notifications:

- `PublishNotification` to send a notification to a user. This can optionally store the notification in the DB.
- `MarkNotificationUnread` to mark a notification as unread.
- `MarkNotificationRead` to mark a notification as read.
- `DeleteNotification` to delete a notification.
- `GetNotifications` to get all notifications for a user.

## PWA Notifications

There are 2 push notification repos for different use cases: 
- `PwaPushNotificationsRepo`: for sending push notifications to PWAs.
- `FcmPushNotificationsRepo`: for sending push notifications to native Android and iOS apps.

Both have similar interfaces:
- `AddPushSubscription`: to add a new push subscription, triggered when the profile turns on PWA notifications in their profile settings.
- `SendPushNotifications`: to send a push notification to a user. This is generally handled by the `NotifierRepo` after storing a notification in the DB using the `PublishNotification` method. 
- `DeletePushSubscriptionByEndpoint`: to delete a push subscription by endpoint.

## Deployment

First, make sure all your env vars in the Kamal file `deploy.yml` are correct. All your vars should be set either in:
 
- `config/config.yml`: only non-secret ones
- `deploy.yml`: only non-secret ones
- `.env`: all secret vars

Then, set the IP of the server host in `deploy.yml`, as well as your image and registry details. Read up on the [kamal documentation](https://kamal-deploy.org) if you get stuck anywhere here. 

### Set up live server

The below command will install docker, build your image, push it to your registry, and then pull it on your remote VPS. If you set up any accessory (cache, standalone DB that is not hosted etc), these will also be deployed.

```bash
kamal setup -c deploy.yml
``` 

At this point, your project should be live, and if `128.0.0.1111` is the IP of your VPS, entering that IP in the search bar on your browser should bring up your site.

### HTTPS

Hop into your VPS console.

```bash
mkdir -p /letsencrypt && touch /letsencrypt/acme.json && chmod 600 /letsencrypt/acme.json
```

Then locally, run

```bash
kamal traefik reboot
```

Your site should now have TLS enabled and you should see the lock icon the search bar.

For reference, the above procedure was taken from [this Kamal issue](https://github.com/basecamp/kamal/discussions/112).
