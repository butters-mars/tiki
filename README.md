# tit.io framework 
## Introduction

  A simple microservice framework based on grpc-go, which provides following features:

  1. Define services with protobuf3 and generate grpc server/client/RESTful gateway.
  2. Service discovery & registration with consul, and client-side load balancer.
  3. Distributed tracing with jeager.
  4. Monitoring by exposing metrics to promethues.
  5. Rate-limiting.
  6. Circuit-breaker.
  7. Authorization & Authentication.
  8. Logging with uniqe id per request.
  9. Validation support.

## Get started

### Validation support
Validation support through https://github.com/envoyproxy/protoc-gen-validate

  1. Install
  ```
  go get -d github.com/envoyproxy/protoc-gen-validate
  make build
  ```

