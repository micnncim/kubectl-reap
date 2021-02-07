FROM golang:1.15-alpine AS builder

WORKDIR /app
COPY . /app

RUN go build -o /bin/kubectl-reap ./cmd/kubectl-reap

FROM gcr.io/distroless/base-debian10

COPY --from=builder /bin/kubectl-reap /usr/bin
CMD ["kubectl-reap"]
