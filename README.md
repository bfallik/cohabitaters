# cohabitaters

A little web app to generate a Christmas card address list from [Google Contacts](https://contacts.google.com/).

I like to store my data centrally but Google doesn't make it easy to aggregate contacts by where they live. I don't want to send multiple cards to the same address.

## Architecture

Cohabitaters uses several platforms and libraries. On the front-end:
* [htmx](https://htmx.org/) for [HATEOS](https://htmx.org/essays/hateoas/)
* [_hyperscript](https://hyperscript.org/) for client-side scripting
* [Tailwind CSS](https://tailwindcss.com/) and [Flowbite](https://flowbite.com/) for CSS and basic UI components
* [Font Awesome](https://fontawesome.com/) for icons

and on the the back-end:
* [Go](https://go.dev/) and [echo](https://echo.labstack.com/) for the web server
* [Air](https://github.com/cosmtrek/air) for live reloading

Deployment uses a healthy mix of [Google Domains](https://domains.google.com/registrar/), [GCP](https://cloud.google.com/), and [fly.io](https://fly.io/).

## Local Development

Operations are driven from the top-level Makefile:
```
❯ make bin/cohab-server
cd cmd/cohab-server && go build -o ../../bin/cohab-server
❯ make air

  __    _   ___
 / /\  | | | |_)
/_/--\ |_| |_| \_ , built with Go

watching .
!exclude air-tmp
watching bin
watching cmd
watching cmd/cohab-server
watching cmd/cohabcli
watching deploy
watching html
watching html/fontawesome-free-6.2.1-web
watching html/fontawesome-free-6.2.1-web/css
watching html/fontawesome-free-6.2.1-web/webfonts
watching html/templates
watching html/templates/partials
watching mapcache
!exclude terraform
building...
make[1]: Entering directory '/home/bfallik/sandbox/cohabitaters'
cd cmd/cohab-server && go build -o ../../bin/cohab-server
make[1]: Leaving directory '/home/bfallik/sandbox/cohabitaters'
running...
^Ccleaning...
see you again~

```

[Dockerfile.fedora](Dockerfile.fedora) contains a verifiable script to prove the build dependencies for `make check`.

```
❯ make image-fedora-test-build
podman build -f Dockerfile.fedora -t fedora-test-build .
STEP 1/6: FROM fedora:37 AS builder
STEP 2/6: WORKDIR /build
...
```

## Deployment

This tool requires a [GCP](https://cloud.google.com/) Project for access to the [People API](https://developers.google.com/people). For simplicity, that configuration is tracked in [Terraform](https://terraform.io).

### Initial Import

```
$ cd terraform && terraform import google_project_service.people_api xmas-card-addresses/people.googleapis.com
```

### Deploy Locally

`make air` will build and launch the web server and repeat when any source files have changed. By default the server will listen on `localhost:8080`.

### Deploy locally with Podman

Use `cohab-server-dev-podman.sh`:

```
  ❯ ./deploy/cohab-server-dev-podman.sh

```

### Deploy remotely

The final version gets deployed to [fly.io](https://fly.io/).

Prerequisite 1: create the DNS record and then the certificate:

```
❯ flyctl certs create cohabitaters.bfallik.net
```

Prerequisite 2: create the applications secret from the downloaded OAuth2 client credentials:

```
❯ flyctl secrets set GOOGLE_APP_CREDENTIALS="$(< path/to/downloaded/client_secret.json)"
```

To have fly.io build and deploy the image:
```
❯ flyctl deploy
```
