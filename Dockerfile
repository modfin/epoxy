FROM golang:1.24.3-alpine3.21 AS epoxy-builder
WORKDIR /work
COPY . .

RUN go build -o /epoxyd cmd/epoxyd/main.go

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=epoxy-builder /epoxyd /epoxyd
USER nobody
CMD [ "/epoxyd" ]
