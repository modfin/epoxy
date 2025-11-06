ARG ALPINE_VERSION=3.22.1
ARG GO_VERSION=1.25.4-alpine3.22
ARG GO_BUILD_ARGS='-trimpath -tags=timetzdata'
ARG GO_LDFLAGS="-s -w"

FROM golang:${GO_VERSION} AS epoxy-builder
ARG GO_BUILD_ARGS
ARG GO_LDFLAGS
WORKDIR /src
RUN --mount=type=bind,target=. go build -v -ldflags="${GO_LDFLAGS}" ${GO_BUILD_ARGS} -o /epoxyd ./cmd/epoxyd

FROM alpine:${ALPINE_VERSION} AS epoxy
ENV TZ=UTC
RUN apk update && apk upgrade
RUN apk add --no-cache ca-certificates tzdata
COPY --from=epoxy-builder /epoxyd /epoxyd
USER nobody
CMD [ "/epoxyd" ]

FROM scratch AS epoxy-slim
ENV TZ=UTC
COPY --from=epoxy /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=epoxy /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=epoxy-builder /epoxyd /epoxyd
USER nobody:nobody
CMD [ "/epoxyd" ]
