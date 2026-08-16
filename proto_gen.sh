#!/bin/bash
set -e

script_dir="$(cd "$(dirname "$0")" && pwd)"
tool_dir="${TMPDIR:-/tmp}/taoxi-protoc-tools"
mkdir -p "$tool_dir"

go build -o "$tool_dir/protoc-gen-go" google.golang.org/protobuf/cmd/protoc-gen-go
go -C "$script_dir/../wali" build -o "$tool_dir/protoc-gen-go-grpc-locator" ./cmd/protoc-gen-go-grpc-locator

PATH="$tool_dir:$PATH" protoc \
  --proto_path="$script_dir/proto" \
  --go_out="$script_dir" \
  --go_opt=module=github.com/2comjie/taoxi-server \
  --go-grpc-locator_out="$script_dir" \
  --go-grpc-locator_opt=module=github.com/2comjie/taoxi-server \
  "$script_dir/proto/test/test.proto" \
  "$script_dir/proto/shared/actor_type.proto" \
  "$script_dir/proto/player/player_client.proto"
