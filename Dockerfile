FROM golang:1.20.1-alpine3.17 as epoxy-builder
WORKDIR /work
COPY . .

RUN go build -o /epoxyd cmd/epoxyd/main.go

FROM alpine:3.17
RUN apk add --no-cache ca-certificates
COPY --from=epoxy-builder /epoxyd /epoxyd
USER nobody
CMD /epoxyd
