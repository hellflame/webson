# Webson Benchmark

## Steps

> Optional, build the code first

start the service

```bash
go run server.go 
```

start the benchmark client

```bash
go run client.go
```

## Benchmark params

By default, the *client* will do two kind of benchmarks.

1. Connectivity Benchmark (connect & close)
2. Endurance Benchmark (connect & send msg & close)

*Connectivity* test default runs a total `1000` connects and concurrently connect `100` lines.
*Endurance* test default runs a total `1000` connects and concurrently connect `100` lines, each connection send `10` messages to the server.

## System params

If the error says: `too many open files`

```bash
ulimit -n 1024  # or higher
```
