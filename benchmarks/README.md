# Marten Benchmarks

Performance comparison of Marten against popular Go web frameworks.

## Frameworks Compared

| Framework | Version | Description |
|-----------|---------|-------------|
| [Marten](https://github.com/gomarten/marten) | v0.1.1 | Zero-dependency, lightweight web framework |
| [Gin](https://github.com/gin-gonic/gin) | v1.9.1 | High-performance HTTP web framework |
| [Echo](https://github.com/labstack/echo) | v4.11.4 | High performance, minimalist web framework |
| [Chi](https://github.com/go-chi/chi) | v5.0.12 | Lightweight, idiomatic router |
| [Fiber](https://github.com/gofiber/fiber) | v2.52.0 | Express-inspired framework built on Fasthttp |

## Benchmark Categories

1. **Static Route** - Simple GET request to `/hello`
2. **Param Route** - Route with single parameter `/users/:id`
3. **Multi-Param** - Route with multiple parameters `/users/:userId/posts/:postId/comments/:commentId`
4. **JSON Response** - Serialize struct to JSON
5. **JSON Binding** - Parse JSON request body

## Run Benchmarks

```bash
cd benchmarks
go mod tidy
go test -bench=. -benchmem -benchtime=3s
```

## Sample Results

Results from Intel Xeon Platinum 8259CL @ 2.50GHz, Go 1.24, Linux:

```
goos: linux
goarch: amd64
cpu: Intel(R) Xeon(R) Platinum 8259CL CPU @ 2.50GHz

Static Route (/hello):
BenchmarkMarten_StaticRoute-2      2490843    1464 ns/op    1040 B/op    11 allocs/op
BenchmarkGin_StaticRoute-2         2709901    1336 ns/op    1040 B/op     9 allocs/op
BenchmarkEcho_StaticRoute-2        2552127    1436 ns/op    1024 B/op    10 allocs/op
BenchmarkChi_StaticRoute-2         1637276    2202 ns/op    1360 B/op    12 allocs/op
BenchmarkFiber_StaticRoute-2        139614   24506 ns/op   10311 B/op    30 allocs/op

Param Route (/users/:id):
BenchmarkMarten_ParamRoute-2       2334639    1564 ns/op    1048 B/op    11 allocs/op
BenchmarkGin_ParamRoute-2          2600355    1418 ns/op    1040 B/op     9 allocs/op
BenchmarkEcho_ParamRoute-2         2449431    1472 ns/op    1016 B/op    10 allocs/op
BenchmarkChi_ParamRoute-2          1405360    2559 ns/op    1688 B/op    14 allocs/op
BenchmarkFiber_ParamRoute-2         142867   25582 ns/op   10300 B/op    29 allocs/op

JSON Response:
BenchmarkMarten_JSON-2             2047221    1755 ns/op    1048 B/op    11 allocs/op
BenchmarkGin_JSON-2                2167873    2050 ns/op    1064 B/op    11 allocs/op
BenchmarkEcho_JSON-2               1901590    1835 ns/op    1080 B/op    11 allocs/op
BenchmarkChi_JSON-2                1920778    1868 ns/op    1376 B/op    12 allocs/op
BenchmarkFiber_JSON-2               136221   26325 ns/op   10356 B/op    32 allocs/op

JSON Binding (POST with body parsing):
BenchmarkMarten_JSONBind-2          419899    8438 ns/op    7194 B/op    32 allocs/op
BenchmarkGin_JSONBind-2             392713    8950 ns/op    7260 B/op    33 allocs/op
BenchmarkEcho_JSONBind-2            392806    8960 ns/op    7227 B/op    32 allocs/op
BenchmarkChi_JSONBind-2             540093    6460 ns/op    7025 B/op    24 allocs/op
BenchmarkFiber_JSONBind-2            70224   47575 ns/op   13251 B/op    53 allocs/op

Multi-Param Route (/users/:userId/posts/:postId/comments/:commentId):
BenchmarkMarten_MultiParam-2       1897114    1837 ns/op    1112 B/op    11 allocs/op
BenchmarkGin_MultiParam-2          2379756    1524 ns/op    1043 B/op    10 allocs/op
BenchmarkEcho_MultiParam-2         2186328    1655 ns/op    1024 B/op    11 allocs/op
BenchmarkChi_MultiParam-2          1276588    2869 ns/op    1688 B/op    14 allocs/op
BenchmarkFiber_MultiParam-2         132202   28778 ns/op   10343 B/op    29 allocs/op
```

*Note: Fiber uses `app.Test()` which has overhead compared to direct `ServeHTTP` calls.*

## Interpretation

- **ns/op** - Nanoseconds per operation (lower is better)
- **B/op** - Bytes allocated per operation (lower is better)
- **allocs/op** - Number of allocations per operation (lower is better)

## Run Your Own Benchmarks

Your results will vary based on:
- CPU architecture (x86 vs ARM)
- Go version
- Operating system
- System load

Always run benchmarks on your target hardware for accurate comparisons.

## Contributing

Found a benchmark that could be improved? PRs welcome!
