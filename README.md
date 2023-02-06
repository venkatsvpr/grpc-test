# grpc-test
Benchmarks for various options

# Results with local benchmark

Captured the results on local benchmark on a container.

```
root@e15e7b03dcbe:/stuff/GitDev/grpc-test# go test -bench=. -benchtime=10s
goos: linux
goarch: amd64
pkg: grpctest
cpu: 11th Gen Intel(R) Core(TM) i7-1185G7 @ 3.00GHz
BenchmarkCompare/Func_call,proto-8                174891             74176 ns/op
BenchmarkCompare/Call_through_UDS,proto-8        4633063              2534 ns/op
BenchmarkCompare/Call_through_GRPC,tcp_socket,_proto-8             27212            661547 ns/op
BenchmarkCompare/Call_through_GRPC,uds,proto-8                     16140            677380 ns/op
PASS
ok      grpctest        68.956s
```
