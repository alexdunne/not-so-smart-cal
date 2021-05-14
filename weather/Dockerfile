FROM golang:1.16-alpine as builder
# `skaffold debug` sets SKAFFOLD_GO_GCFLAGS to disable compiler optimizations
ARG SKAFFOLD_GO_GCFLAGS
# Determine which binary to build
ARG APP
ENV GO111MODULE=on
WORKDIR /code/
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /app "./cmd/${APP}"

FROM alpine:3.10
ARG APP
# Define GOTRACEBACK to mark this container as using the Go language runtime
# for `skaffold debug` (https://skaffold.dev/docs/workflows/debug/).
ENV GOTRACEBACK=single
EXPOSE 8080
COPY --from=builder /app .
CMD ["./app"]