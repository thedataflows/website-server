## build go binary
FROM golang:1.27-alpine AS gobuilder
ARG VERSION=dev
WORKDIR /app
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download && \
  go build -trimpath -ldflags "-s -w -X=main.version=${VERSION}" -o ./ws

## build final image
FROM alpine:3.24
WORKDIR /app
COPY --from=gobuilder /app/ws .
COPY config.yaml .
RUN apk add --no-cache gcompat
EXPOSE 3000
USER nobody
CMD ["./ws"]
