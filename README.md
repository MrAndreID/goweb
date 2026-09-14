# MrAndreID / Go Website (Web)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

The `MrAndreID/GoWeb` is a skeleton uses the Go Programming Language (GoLang) with The Echo Framework for The Frontend that renders a Server-Side Rendered (SSR) Website.

## Table of Contents

* [Requirements](#requirements)
* [Installation](#installation)
* [Unit Test](#unit-test)
* [Usage](#usage)
* [Versioning](#versioning)
* [Authors](#authors)
* [Contributing](#contributing)
* [Official Documentation for Go Language](#official-documentation-for-go-language)
* [License](#license)

## Requirements

To use The `MrAndreID/GoWeb`, you must ensure that you meet the following requirements:
- [Go](https://golang.org/) >= 1.27

## Installation

To use The `MrAndreID/GoWeb`, you must follow the steps below:
- Clone a Repository
```git
# git clone https://github.com/MrAndreID/goweb.git
```
- Get Dependancies
```go
# go mod download
# go mod tidy
```
- Create .env file from .env.example (Linux)
```sh
# cp .env.example .env
```
- Configuring .env file, including the Backend connection:
  - `BACKEND_BASE_URL` - base URL of the Backend API (default `http://127.0.0.1:10001`)
  - `BACKEND_APP_KEY` - sent as the `X-App-Key` header; leave empty to omit the header
  - `BACKEND_TIMEOUT` - total request timeout in seconds (default `10`)
  - `BACKEND_MAX_RESPONSE_BYTES` - max response body size in bytes read into memory (default `2097152`)
  - `BACKEND_MAX_IDLE_CONNS` - max idle connections in the pool (default `100`)
  - `BACKEND_MAX_CONNS_PER_HOST` - max connections per host (default `100`)
  - `BACKEND_RETRY_COUNT` - retry attempts for transient failures (default `2`)

## Unit Test

To Run Unit Test for The `MrAndreID/GoWeb`, you must ensure that you meet the following requirements:
- Prepare a stub or mock of the Backend for the repository layer
- Run Unit Test for The `MrAndreID/GoWeb`
```go
# go test -v -cover -coverpkg=./internal/feature/... ./internal/feature/... -count=1
```
- Run Unit Test for The `MrAndreID/GoWeb` with Coverage Profile
```go
# go test -v -cover -coverpkg=./internal/feature/... -coverprofile=coverage.out ./internal/feature/... -count=1
# go tool cover -func=coverage.out
```

## Usage

To use The `MrAndreID/GoWeb`, you must ensure that you meet the following requirements:
- Directory Structure The `MrAndreID/GoWeb`

| Name                                       | Description                                               |
| :----------------------------------------- | :-------------------------------------------------------- |
| `cmd/web`                                  | Entry Point for The Application                           |
| `internal/application`                     | Initialization of Echo Framework, Middleware, Renderer, and Routes. |
| `internal/application/config`              | Configuration from Env File                               |
| `internal/entity`                          | Shared Struct Data (incl. Backend Response Envelope)      |
| `internal/feature/user`                    | User Feature (SSR Pages under `/users`)                   |
| `internal/feature/user/templates`          | HTML Templates for The User Feature                       |
| `storage`                                  | Folder for Add Maintenance Flag File                      |
| `storage/log`                              | Folder for Log File                                       |

- Run The `MrAndreID/GoWeb`
```go
# go run ./cmd/web
```
- Run The `MrAndreID/GoWeb` with Docker
```docker
# docker build --no-cache -t goweb:1.0.0 .
# docker run --name goweb --restart=always -d -p 10101:10101 -v /path/to/.env:/app/.env:ro -v /path/to/folder:/app/storage goweb:1.0.0
```
- Set The `MrAndreID/GoWeb` to Maintenance Mode in Storage Folder
```sh
# touch storage/maintenance.flag
```

## Versioning

I use [Semanting Versioning](https://semver.org/). For the versions available, see the tags on this repository. 

## Authors

- **Andrea Adam** - [MrAndreID](https://github.com/MrAndreID)

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.
Please make sure to update tests as appropriate.

## Official Documentation for Go Language

Documentation for Go Language can be found on the [Go Package website](https://pkg.go.dev/).

## License

The `MrAndreID/GoWeb` is released under the [MIT License](https://opensource.org/licenses/MIT). See the `LICENSE` file for more information.
