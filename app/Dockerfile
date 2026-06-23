# ---- Stage 1: build ----
FROM golang:1.26 AS builder
WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 go build -o /server .

# ---- Stage 2: runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /server /server
USER nonroot:nonroot         
EXPOSE 8080
ENTRYPOINT ["/server"]