FROM golang:1.23.1-alpine3.20 AS epoxy-builder
WORKDIR /work
COPY . .

RUN go build -o /epoxyd cmd/epoxyd/main.go

FROM alpine:3.20.3
RUN apk add --no-cache ca-certificates
COPY --from=epoxy-builder /epoxyd /epoxyd
USER nobody
CMD [ "/epoxyd" ]
