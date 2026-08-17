# gRPC Setup and Compilation Guide

This guide explains how to set up and compile Protocol Buffer (protobuf) files for gRPC services in Go.

## Prerequisites

Before you can compile the protobuf files, you need to install several dependencies:

### 1. Protocol Buffers Compiler

#### For Ubuntu/Debian:
```bash
sudo apt install protobuf-compiler
```

#### For macOS:
```bash
brew install protobuf
```

#### For other systems:
Download the appropriate release from the [official protobuf releases page](https://github.com/protocolbuffers/protobuf/releases)

### 2. Go Protobuf Plugins

Install the necessary Go plugins for protobuf compilation:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Make sure `$GOPATH/bin` is in your PATH:
```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Project Structure

The expected project structure is:
```
.
├── bin/
│   └── compile.sh
├── proto/
│   └── *.proto
└── generated Go files
```

## Compilation Script

The `compile.sh` script in the `bin` directory handles the protobuf compilation:

```bash
#!/bin/bash
if [ -L $0 ]
then
    BASE_DIR=`dirname $(readlink $0)`
else
    BASE_DIR=`dirname $0`
fi
base_path=$(cd $BASE_DIR/..; pwd)
cd $base_path && \
  protoc -I ./proto \
  --go_out=. \
  --go-grpc_out=. \
  ./proto/**/*.proto
```

### Making the Script Executable

Give execute permissions to the compilation script:
```bash
chmod +x ./bin/compile.sh
```

## Usage

To compile your protobuf files, run:
```bash
./bin/compile.sh
```

Or if the script isn't executable:
```bash
bash ./bin/compile.sh
```

## Generated Files

The compilation will generate several types of files:
- `*.pb.go`: Contains the Go structs generated from your protocol buffer message definitions
- `*_grpc.pb.go`: Contains the gRPC service definitions and client/server code

## Troubleshooting

### Common Issues:

1. `protoc: command not found`
    - Solution: Install the protobuf compiler as described in the Prerequisites section

2. `permission denied`
    - Solution: Make the script executable using `chmod +x ./bin/compile.sh`

3. `*.proto: No such file or directory`
    - Solution: Ensure your .proto files are in the correct directory structure under ./proto

4. `--go_out: protoc-gen-go: Plugin failed with status code 1`
    - Solution: Make sure the Go protobuf plugins are properly installed and in your PATH

## Maintaining Generated Code

- Don't modify the generated `.pb.go` and `_grpc.pb.go` files directly
- Always make changes to the `.proto` files and recompile
- Consider adding generated files to your version control system as they are needed for compilation

## Additional Resources

- [Protocol Buffers Documentation](https://developers.google.com/protocol-buffers)
- [gRPC-Go Documentation](https://grpc.io/docs/languages/go/)
- [Go Plugins for Protocol Buffers](https://github.com/golang/protobuf)