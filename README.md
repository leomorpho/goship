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


## WIP Documentation
 
See [goship.run](http://goship.run). NOTE: it's currently being actively developed! Feel free to help ❤️.