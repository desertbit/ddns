From golang:alpine

WORKDIR /app
COPY . .

RUN go get -d -v ./...
RUN CGO_ENABLED=0 \
    go build -v \
        ./cmd/ddns

FROM alpine
RUN apk add --no-cache ca-certificates

COPY --from=0 /app/ddns /ddns
ENTRYPOINT ["/ddns"]
CMD ["server"]

EXPOSE 53/udp
EXPOSE 53/tcp
EXPOSE 80/tcp