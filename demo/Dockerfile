FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /tfui ./cmd/tfui

FROM alpine:3.20
COPY --from=build /tfui /usr/local/bin/tfui
COPY demo/fixtures/ /fixtures/
ENTRYPOINT ["tfui"]
CMD ["--plan", "/fixtures/plan-large.json", "--state", "/fixtures/state-large.json"]
