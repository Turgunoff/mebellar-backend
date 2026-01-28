#!/bin/bash

# Generate Go code from proto files
# Requires: protoc, protoc-gen-go, protoc-gen-go-grpc

set -e

PROTO_DIR="./proto"
OUT_DIR="./pkg/pb"

# Create output directory if it doesn't exist
mkdir -p "$OUT_DIR"

# Generate code for all proto files at once
echo "📦 Processing all proto files..."
protoc \
    --go_out="$OUT_DIR" \
    --go_opt=paths=source_relative \
    --go-grpc_out="$OUT_DIR" \
    --go-grpc_opt=paths=source_relative \
    --proto_path="$PROTO_DIR" \
    "$PROTO_DIR"/*.proto

echo "✅ Proto code generation complete!"
echo "📁 Generated files are in: $OUT_DIR"
