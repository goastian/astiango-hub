FROM golang:1.23 AS build

# Context path is the root of the repository

WORKDIR /go/src/app

# Copy the current directory (backend) first
COPY ./backend ./backend/

# Copy required modules from relative paths
COPY ./core ./core/
COPY ./grpc ./grpc/
COPY ./trace ./trace/
COPY ./vcs ./vcs/

ENV GO111MODULE on

# Build a product-named binary from the backend entrypoint.
RUN cd backend && go mod tidy && go build -o /go/bin/astiango-hub-server .

FROM build AS test
RUN cd backend && go test ../core/middlewares ../core/utils ../core/models/models && \
    go test ../core/task/handler -run TestRunnerRedactsInjectedSecretValues

FROM alpine:3.14

# Copy the built binary
COPY --from=build /go/bin/astiango-hub-server /go/bin/astiango-hub-server
