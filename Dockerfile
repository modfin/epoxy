FROM golang:1.22.4-alpine3.20 as epoxy-builder
WORKDIR /work
COPY . .

RUN go build -o /epoxyd cmd/epoxyd/main.go

FROM alpine:3.20.0
RUN apk add --no-cache ca-certificates
COPY --from=epoxy-builder /epoxyd /epoxyd
USER nobody
CMD /epoxyd
