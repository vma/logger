**Description**

`logger` is a http request logger for the [pressly/chi] (https://github.com/pressly/chi) go HTTP router.

It comes  with `CommonLogger` and `CombinedLogger` middlewares wich logs the requests
in Apache CommonLoger and CombinedLogger format respectively.

These middlewares are ported from [gorilla handlers] (https://github.com/gorilla/handlers)

**Installation**

Installation can be done as usual:

```
$ go get github.com/vma/logger
```

**Usage**

```go
package main

import (
    "net/http"

    "github.com/pressly/chi"
    "github.com/vma/logger"
)

func main() {
    r := chi.NewRouter()

    r.Use(logger.CombinedLogger(os.Stdout))
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!\n"))
    })
    http.ListenAndServe(":3000", r)
}
```

This code will produce a log line like this one:

`::1 - - [02/Jan/2017:20:07:27 +0100] "GET / HTTP/1.1" 200 13 0.012ms "" "curl/7.51.0"`

You wil notice the additional answer time field. The difference with the CommonLog format is
that the latter does not contain the two last fields (referrer and user agent)

The full explanation of the log formats can be found at http://stackoverflow.com/a/9234855

The `CombinedLogger` and `CommonLogger` functions accept an `io.Writer` parameter so, you can
easily write to a file instead of stdout.


