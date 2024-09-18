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

> **Warning alert!** this project is in active development as I am adding things after first trying them out in prod for [Chérie](https://cherie.chatbond.app/), a relationship app to grow your couple. Note that I would welcome any help to develop this boilerplate ❤️.

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
>>>>>>> main

## WIP Documentation
 
See [goship.run](http://goship.run). NOTE: it's currently being actively developed! Feel free to help ❤️.