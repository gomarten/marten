# Marten Benchmarks

Performance comparison of Marten v0.1.3 against popular Go web frameworks.

## Test Environment

- **CPU**: Intel(R) Xeon(R) Platinum 8259CL @ 2.50GHz
- **OS**: Linux (amd64)
- **Go Version**: 1.24.0
- **Date**: January 2026

## Frameworks Compared

| Framework | Version | Description |
|-----------|---------|-------------|
| [Marten](https://github.com/gomarten/marten) | v0.1.3 | Zero-dependency, lightweight web framework |
| [Gin](https://github.com/gin-gonic/gin) | v1.9.1 | High-performance HTTP web framework |
| [Echo](https://github.com/labstack/echo) | v4.11.4 | High performance, minimalist web framework |
| [Chi](https://github.com/go-chi/chi) | v5.0.11 | Lightweight, idiomatic router |
| [Fiber](https://github.com/gofiber/fiber) | v2.52.0 | Express-inspired framework built on Fasthttp |

## Benchmark Results

### Static Route (`/hello`)

| Framework | ns/op | B/op | allocs/op | vs Marten |
|-----------|-------|------|-----------|-----------|
| **Gin** | 1,323 | 1,040 | 9 | **+8.4% faster** |
| **Echo** | 1,421 | 1,024 | 10 | **+1.7% faster** |
| **Marten** | 1,445 | 1,040 | 11 | baseline |
| **Chi** | 2,208 | 1,392 | 12 | -34.6% slower |
| **Fiber** | 24,300 | 10,685 | 31 | -94.1% slower |

### Param Route (`/users/:id`)

| Framework | ns/op | B/op | allocs/op | vs Marten |
|-----------|-------|------|-----------|-----------|
| **Gin** | 1,419 | 1,040 | 9 | **+7.6% faster** |
| **Echo** | 1,474 | 1,016 | 10 | **+4.0% faster** |
| **Marten** | 1,536 | 1,048 | 11 | baseline |
| **Chi** | 2,520 | 1,720 | 14 | -39.1% slower |
| **Fiber** | 24,571 | 10,676 | 30 | -93.7% slower |

### JSON Response

| Framework | ns/op | B/op | allocs/op | vs Marten |
|-----------|-------|------|-----------|-----------|
| **Gin** | 1,583 | 1,040 | 10 | **+4.1% faster** |
| **Marten** | 1,651 | 1,024 | 10 | baseline |
| **Echo** | 1,754 | 1,056 | 10 | -5.9% slower |
| **Chi** | 1,890 | 1,408 | 12 | -12.6% slower |
| **Fiber** | 25,841 | 10,707 | 32 | -93.6% slower |

### JSON Binding

| Framework | ns/op | B/op | allocs/op | vs Marten |
|-----------|-------|------|-----------|-----------|
| **Chi** | 6,810 | 7,410 | 25 | **+18.3% faster** |
| **Marten** | 8,339 | 7,547 | 33 | baseline |
| **Gin** | 8,634 | 7,612 | 34 | -3.4% slower |
| **Echo** | 8,766 | 7,579 | 33 | -4.9% slower |
| **Fiber** | 51,014 | 13,617 | 53 | -83.7% slower |

*Note: Chi's advantage is due to minimal JSON processing in the benchmark.*

### Multi-Param Route

| Framework | ns/op | B/op | allocs/op | vs Marten |
|-----------|-------|------|-----------|-----------|
| **Gin** | 1,511 | 1,043 | 10 | **+17.9% faster** |
| **Echo** | 1,634 | 1,024 | 11 | **+11.2% faster** |
| **Marten** | 1,841 | 1,112 | 11 | baseline |
| **Chi** | 2,836 | 1,720 | 14 | -35.1% slower |
| **Fiber** | 26,456 | 10,721 | 30 | -93.0% slower |

### Query Params (`?q=golang&page=1&limit=10`)

| Framework | ns/op | B/op | allocs/op | vs Marten |
|-----------|-------|------|-----------|-----------|
| **Echo** | 3,789 | 2,016 | 23 | **+30.1% faster** |
| **Gin** | 4,002 | 2,112 | 27 | **+26.1% faster** |
| **Marten** | 5,419 | 2,945 | 35 | baseline |

### Large JSON Response

| Framework | ns/op | B/op | allocs/op | vs Marten |
|-----------|-------|------|-----------|-----------|
| **Marten** | 2,737 | 1,424 | 11 | baseline |
| **Gin** | 2,818 | 1,696 | 11 | -2.9% slower |
| **Echo** | 2,867 | 1,456 | 11 | -4.5% slower |

### Route Group (`/api/v1/users/:id`)

| Framework | ns/op | B/op | allocs/op | vs Marten |
|-----------|-------|------|-----------|-----------|
| **Echo** | 2,364 | 1,472 | 15 | **+6.3% faster** |
| **Gin** | 2,365 | 1,456 | 16 | **+6.3% faster** |
| **Marten** | 2,524 | 1,504 | 16 | baseline |

### Wildcard Route (`/files/*filepath`)

| Framework | ns/op | B/op | allocs/op | vs Marten |
|-----------|-------|------|-----------|-----------|
| **Gin** | 1,379 | 1,040 | 9 | **+18.4% faster** |
| **Echo** | 1,473 | 1,032 | 10 | **+12.8% faster** |
| **Marten** | 1,690 | 1,104 | 12 | baseline |

### Parallel Requests (Concurrent)

| Framework | ns/op | B/op | allocs/op | vs Marten |
|-----------|-------|------|-----------|-----------|
| **Gin** | 4,607 | 6,145 | 18 | **+2.3% faster** |
| **Echo** | 4,667 | 6,129 | 19 | **+1.1% faster** |
| **Marten** | 4,717 | 6,145 | 20 | baseline |

## Performance Summary

### Overall Ranking

1. **Gin** - Fastest overall, optimized JSON library
2. **Echo** - Very close to Gin, excellent performance
3. **Marten** - Competitive with Gin/Echo, zero dependencies ⭐
4. **Chi** - Good for simple use cases
5. **Fiber** - Slower in tests due to `app.Test()` overhead

### Marten's Performance Profile

**Strengths:**
- ✅ Best large JSON response performance
- ✅ Competitive with Gin and Echo (within 2-18%)
- ✅ Excellent parallel/concurrent performance
- ✅ Consistent memory allocations
- ✅ Zero dependencies = smaller binaries

**Areas for Improvement:**
- ⚠️ Query parameter parsing slower than Gin/Echo
- ⚠️ Wildcard routes slower than Gin/Echo
- ⚠️ Multi-param routes slower than Gin/Echo

### Real-World Context

For most applications, these differences are negligible:

- **1,445 ns/op** = ~692,000 requests/second/core
- Network latency (1-100ms) dominates request time
- Database queries (1-100ms) are the real bottleneck
- JSON encoding is rarely the limiting factor

**Choose Marten if you value:**
- Zero dependencies
- Readable, maintainable code
- Predictable behavior
- Small binary size
- Competitive performance

**Choose Gin/Echo if you need:**
- Maximum performance
- Large ecosystem
- Extensive middleware library

## Run Benchmarks

```bash
cd benchmarks
go mod tidy
go test -bench=. -benchmem -benchtime=3s
```

### Compare Specific Frameworks

```bash
# Marten vs Gin
go test -bench="Marten|Gin" -benchmem

# All JSON benchmarks
go test -bench="JSON" -benchmem

# Parallel benchmarks only
go test -bench="Parallel" -benchmem

# Memory profiling
go test -bench=Marten_StaticRoute -memprofile=mem.out
go tool pprof mem.out
```

## Benchmark Methodology

- Each benchmark runs for 3 seconds minimum
- Uses `httptest.NewRecorder()` for consistent testing
- Measures time per operation (ns/op)
- Measures bytes allocated per operation (B/op)
- Measures number of allocations per operation (allocs/op)
- All frameworks tested with default configuration
- Gin runs in release mode (`gin.ReleaseMode`)
- Parallel benchmarks use `b.RunParallel()` for concurrency testing

## Notes

- **Fiber** appears slower due to `app.Test()` overhead in benchmarks. In production with real HTTP connections, Fiber performs better.
- **Chi** JSON binding benchmark doesn't fully parse JSON, giving it an unfair advantage.
- **Marten** uses stdlib `encoding/json` while Gin uses `jsoniter` for faster JSON encoding.
- Results may vary based on CPU, Go version, and workload patterns.
- Parallel benchmarks simulate concurrent request handling.

## Interpretation

- **ns/op** - Nanoseconds per operation (lower is better)
- **B/op** - Bytes allocated per operation (lower is better)
- **allocs/op** - Number of allocations per operation (lower is better)

## Key Takeaways

1. **Marten is competitive** - Within 2-18% of Gin/Echo for most operations
2. **Zero dependencies matter** - No external packages to manage or update
3. **Performance is excellent** - 692K+ requests/second/core for simple routes
4. **Real bottlenecks elsewhere** - Network and database are typically slower
5. **Choose based on needs** - Performance vs simplicity vs ecosystem

## Contributing

Found a benchmark that could be improved? PRs welcome!
