Local Router
------------

A simple to manage router for running multiple Docker Compose environments.

## Features

* Automatically provision certificates
* Certificates are all provisioned through a locally signed CA, making it easy for local installation
* Route discovery using labels

## Getting started

### Start the router

* Clone this repository
* Run `docker compose up`, alternatively `docker compose up -d` for the router to run in the background

The router is now listening on port 80 and 443.

### Configure a Docker Compose Project

Update any Docker Compose projects by:

* Removing any conflicting ports eg. `8080:8080`
* Add a label to the service which you want to route to (example below)
* Add a `/etc/hosts` record (manually) eg. `127.0.0.1 myapp.localhost` 

**Label Example**

```yaml
services:
  myapp:
    labels:
      - "skpr.host=myapp.localhost"
```

## Next steps for this project

* Testing
* Steps to install local CA to avoid certificate errors
* More documentation
* Move from a hardcoded 8080 target to a configuration file 
* Consider moving to decoupling the certificate handling to be a sidecar on the project
