# builder image
FROM golang:1.19.1-alpine3.16 as builder
ADD ./ /build/
WORKDIR /build
RUN CGO_ENABLED=0 GOOS=linux go build -a -o xsyn-transactions ./cmd/server

FROM alpine:3.16
COPY --from=builder /build/xsyn-transactions .

# executable
ENTRYPOINT [ "./xsyn-transactions" ]
