# Service

[![License](https://img.shields.io/badge/license-MIT-blue)](https://choosealicense.com/licenses/mit/)
![go.mod Go version](https://img.shields.io/github/go-mod/go-version/krostar/service?label=go)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/krostar/service)
[![Latest tag](https://img.shields.io/github/v/tag/krostar/service)](https://github.com/krostar/service/tags)
[![Go Report](https://goreportcard.com/badge/github.com/krostar/service)](https://goreportcard.com/report/github.com/krostar/service)

A robust Go library for managing long-running services and goroutines with graceful shutdown capabilities. Perfect for building microservices, HTTP servers, background workers, and any application that needs to coordinate multiple concurrent services.

## ✨ Features

- **🔄 Concurrent service management**: Run multiple services concurrently with automatic lifecycle coordination
- **🛑 Graceful shutdown**: Context-based cancellation with proper cleanup handling
- **⚡ Fail-fast behavior**: When one service fails, all services are gracefully stopped
- **🔧 Flexible configuration**: Extensive options for customizing service behavior
- **📊 Error aggregation**: Comprehensive error reporting with detailed context
- **🧪 Production ready**: Thoroughly tested with race condition detection

## 🚀 Quick Start

```bash
go get github.com/krostar/service
```

### Basic Service Runner

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/krostar/service"
)

func main() {
  doSomethingFunc := func(ctx context.Context) error {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for {
      select {
      case <-ctx.Done():
        fmt.Println("Worker shutting down gracefully...")
        return nil
      case <-ticker.C:
        fmt.Println("Worker doing work...")
      }
    }
  }

  ctx := context.Background()
  worker1 := service.RunFunc(doSomethingFunc)
  worker2 := service.RunFunc(doSomethingFunc)

  if err := service.Run(ctx, worker1, worker2); err != nil {
    log.Fatalf("service failed: %v", err)
  }
}
```

### HTTP Server Example

```go
package main

import (
  "fmt"
  "log"
  "net/http"
  "syscall"

  "github.com/krostar/cli"
  "github.com/krostar/service"
  httpservice "github.com/krostar/service/net/http"
  netservice "github.com/krostar/service/net"
)

func main() {
  ctx, cancel := cli.NewContextCancelableBySignal(syscall.SIGINT, syscall.SIGTERM)
  defer cancel()

  apiHandler := func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, "ping") }

  mainServer := httpservice.ListenAndServe(
    http.HandlerFunc(apiHandler),
    netservice.ListenWithAddress("tcp", ":8080"),
  )

  metricsHandler := func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprintln(w, "42 requests") }

  metricsServer := httpservice.ListenAndServe(
    http.HandlerFunc(metricsHandler),
    netservice.ListenWithAddress("tcp", ":9090"),
  )

  if err := service.Run(ctx, mainServer, metricsServer); err != nil {
    log.Fatalf("services failed: %v", err)
  }
}
```

## 🏗️ Project layout

The service package is built around a few core concepts:

### Runner Interface

```go
type Runner interface {
  Run(ctx context.Context) error
}
```

Any type implementing this interface can be managed by the service runner. The `Run` method should:

- block until the context is canceled or an unrecoverable error occurs
- return `nil` or `context.Canceled` when stopped due to context cancellation
- return a specific error only for unexpected failures

### Service Coordination

The `service.Run()` function coordinates multiple runners:

1. **Parallel execution**: all runners start concurrently in separate goroutines
1. **Fail-fast**: when any runner stops, all others are signaled to stop via context cancellation
1. **Graceful shutdown**: context cancellation allows services to clean up properly

### Network Services

The `net` subpackage provides specialized utilities for network services:

- **HTTP servers**: Simplified HTTP server management with graceful shutdown
- **TLS support**: Easy TLS configuration for secure communications
- **Systemd integration**: Support for systemd socket activation
- **Keep-Alive configuration**: Fine-tuned TCP keep-alive settings

## Contributing

This project uses Nix for development environment management, and Just to simplify commands / parameters:

```bash
# Enter development shell
nix develop

# Run all checks
just ci

# Format code
just fmt
```

## License

This project is under the MIT licence, please see the LICENSE file.
