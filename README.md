# desyncutil

[![Go Reference](https://pkg.go.dev/badge/github.com/andrewheberle/desyncutil.svg)](https://pkg.go.dev/github.com/andrewheberle/desyncutil)
[![Go Report Card](https://goreportcard.com/badge/github.com/andrewheberle/desyncutil)](https://goreportcard.com/report/github.com/andrewheberle/desyncutil)

Package `desyncutil` provides utilities that complement the
[desync](https://github.com/folbricht/desync) library without modifying it.

## Requirements

Go 1.24.0 or later (matching the minimum version required by desync).

## Installation

```sh
go get github.com/andrewheberle/desyncutil
```

## Contents

### RateLimitedStore

`RateLimitedStore` wraps any `desync.Store` and limits the bandwidth consumed
by `GetChunk` calls to a configurable bytes-per-second target. The limit is
shared across all concurrent callers, making it correct when desync’s parallel
download flag (`-n`) is in use.

Rate limiting is applied after each chunk is fetched using a token bucket
(provided by [`golang.org/x/time/rate`](https://pkg.go.dev/golang.org/x/time/rate)).
Because the `desync.Store` interface does not expose the compressed wire size
of a chunk, the token charge is based on the uncompressed chunk size. When the
inner store returns compressed chunks this causes the limiter to be more
conservative than the raw bandwidth alone would require, but for the purpose of
capping bandwidth the difference is acceptable.

#### Usage

```go
import (
    "github.com/andrewheberle/desyncutil"
    "github.com/folbricht/desync"
)

// Wrap an existing store, limiting throughput to 10 MB/s.
inner, err := desync.NewRemoteHTTPStore(location, opt)
if err != nil {
    log.Fatal(err)
}

store := desyncutil.NewRateLimitedStore(inner, 10*1024*1024)

// Use store anywhere a desync.Store is accepted.
```

## License

This project is licensed under the MIT License.
