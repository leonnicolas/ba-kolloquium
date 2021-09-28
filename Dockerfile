FROM golang:alpine as build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o server --mod=vendor

FROM scratch
COPY --from=build /src/server .
ENTRYPOINT ["./server"]
