FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/coordinator ./cmd/coordinator
RUN mkdir /data

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/coordinator /coordinator
COPY --from=build /src/migrations /migrations
COPY --from=build /data /data
ENV MIGRATIONS_DIR=/migrations
ENV DB_PATH=/data/coordinator.db
EXPOSE 8080
ENTRYPOINT ["/coordinator"]
