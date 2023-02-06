# grpc-test
Comparing the performance of 

# Results with local benchmark

Captured the results on local benchmark on a container.

```
goos: linux
goarch: amd64
pkg: grpctest
cpu: Intel(R) Xeon(R) Platinum 8272CL CPU @ 2.60GHz
BenchmarkCompare/Function_call,_proto__                    19078             62848 ns/op
BenchmarkCompare/Call_through_UDS,proto                   287431              4093 ns/op
BenchmarkCompare/Call_through_GRPC,tcp,proto                5125            218449 ns/op
BenchmarkCompare/Call_through_GRPC,uds,proto                5938            198954 ns/op
PASS
ok      grpctest        5.421s
```
