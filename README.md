# cohabitaters

A tool to generate a Christmas card address list from [Google Contacts](https://contacts.google.com/).

## Usage

This follows the [People API Quickstart example](https://developers.google.com/people/quickstart/go).

First, download the credentials.json to this directory. Then

```
  $ go run main.go
  Go to the following link in your browser then type the authorization code:
https://accounts.google.com/o/oauth2/auth?access_type=offline&client_id=1048297799487-jd1j1q4bbspmgf9b71h5skkj6amfl1ob.apps.googleusercontent.com&redirect_uri=http%3A%2F%2Flocalhost&response_type=code&scope=https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fcontacts.readonly&state=state-token
```
and follow the instructions to open the link in your browser. After the authorization process, Google will attempt to redirect you to a local webserver that isn't listening. Simply copy the `code=XXXX` authorization code into the console:

```
XXXX
Saving credential file to: token.json
List 10 connection names:
Jennie May
...
```

## Setup

This tool uses GCP to enable the People API in a project. For simplicity, that configuration is tracked in [Terraform](https://terraform.io).

### Initial Import

```
$ cd terraform && terraform import google_project_service.people_api xmas-card-addresses/people.googleapis.com
```
