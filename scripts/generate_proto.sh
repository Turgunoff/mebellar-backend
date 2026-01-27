#!/bin/bash

# Generate Go code from proto files
# Requires: protoc, protoc-gen-go, protoc-gen-go-grpc

set -e

PROTO_DIR="./proto"
OUT_DIR="./pkg/pb"

echo "🔧 Generating gRPC code from proto files..."

# Create output directory if it doesn't exist
mkdir -p "$OUT_DIR"

# Generate code for each proto file
for proto_file in "$PROTO_DIR"/*.proto; do
    if [ -f "$proto_file" ]; then
        echo "📦 Processing $(basename $proto_file)..."
        protoc \
            --go_out="$OUT_DIR" \
            --go_opt=paths=source_relative \
            --go-grpc_out="$OUT_DIR" \
            --go-grpc_opt=paths=source_relative \
            --proto_path="$PROTO_DIR" \
            "$proto_file"
    fi
done

echo "✅ Proto code generation complete!"
echo "📁 Generated files are in: $OUT_DIR"
