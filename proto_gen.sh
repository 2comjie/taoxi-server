#!/bin/bash
protoc \
  --proto_path=proto \
  --go_out=. \
  --go_opt=module=github.com/2comjie/taoxi-server \
  --go-grpc-locator_out=. \
  --go-grpc-locator_opt=module=github.com/2comjie/taoxi-server \
  proto/test/test.proto
