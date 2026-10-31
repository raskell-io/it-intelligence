# itx: one static binary, plus the configuration and the Kestrel seed it reads.
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# The Anthropic provider is left out by default: compiling its SDK needs more memory than the 2 GB a
# default Docker VM has, and the demo runs on recorded judgements. Build with --build-arg GO_TAGS= to include it.
ARG GO_TAGS=noanthropic
RUN CGO_ENABLED=0 go build -tags "$GO_TAGS" -trimpath -ldflags="-s -w" -o /out/itx ./cmd/itx

FROM alpine:3.22
RUN adduser -D -u 10001 itx
WORKDIR /app
COPY --from=build /out/itx /usr/local/bin/itx
COPY config ./config
COPY seed ./seed
RUN mkdir -p var && chown itx var
USER itx
ENV ITX_ROOT=/app ITX_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["itx"]
CMD ["serve"]
