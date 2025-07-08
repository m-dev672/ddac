FROM golang:1.24.4-bullseye AS build

WORKDIR /src
COPY src/ .

RUN go mod tidy
RUN go build --ldflags "-extldflags \"-static\"" -o /airport

FROM gcr.io/distroless/base-debian11

WORKDIR /
COPY --from=build /airport ./airport
COPY src/DispatcherABI.json ./

CMD ["/airport"]