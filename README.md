# service

[![License](https://img.shields.io/badge/license-MIT-blue)](https://choosealicense.com/licenses/mit/)
![go.mod Go version](https://img.shields.io/github/go-mod/go-version/krostar/service?label=go)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/krostar/service)
[![Latest tag](https://img.shields.io/github/v/tag/krostar/service)](https://github.com/krostar/service/tags)
[![Go Report](https://goreportcard.com/badge/github.com/krostar/service)](https://goreportcard.com/report/github.com/krostar/service)

Golang library that handle the complexity of running long-living goroutines (like a http server).

```go
func main() {
    ctx := context.Background()

    listener1, err := net.Listen("tcp", ":8080")
    if err != nil {
        panic(fmt.Errorf("unable to create listener: %v", err))
    }

    server1 := netservice.Serve(
        &http.Server{Handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
            rw.WriteHeader(http.StatusTeapot)
        })},
        listener1,
        netservice.ServerWithServeErrorTransformer(func(err error) error {
            if errors.Is(err, http.ErrServerClosed) {
                return nil
            }
            return err
        }),
    )

    listener2, err := netservice.NewListener(ctx, ":9090")
    if err != nil {
        panic(fmt.Errorf("unable to create listener: %v", err))
    }

    server2 := netservice.Serve(
        &http.Server{Handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
            rw.WriteHeader(http.StatusOK)
        })},
        listener2,
        netservice.ServerWithServeErrorTransformer(func(err error) error {
            if errors.Is(err, http.ErrServerClosed) {
                return nil
            }
            return err
        }),
    )

    if err := service.Run(ctx, server1, server2); err != nil {
        panic(fmt.Errorf("unable to run services: %v", err))
    }
}
```

## License

This project is under the MIT licence, please see the LICENSE file.
