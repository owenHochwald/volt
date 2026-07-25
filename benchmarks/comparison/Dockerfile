# syntax=docker/dockerfile:1

ARG GO_VERSION=1.25.4
ARG PYTHON_VERSION=3.13.5

FROM golang:${GO_VERSION}-bookworm AS external-go-tools

RUN mkdir -p /out \
    && GOBIN=/out go install github.com/rakyll/hey@v0.1.5 \
    && GOBIN=/out go install go.k6.io/k6/v2@v2.0.0

FROM golang:${GO_VERSION}-bookworm AS volt-builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN mkdir -p /out \
    && CGO_ENABLED=0 go build -trimpath -o /out/volt ./cmd/volt \
    && CGO_ENABLED=0 go build -trimpath -o /out/benchmark-target ./benchmarks/target

FROM debian:bookworm-slim AS wrk-builder

ARG WRK_VERSION=4.2.0
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        build-essential \
        ca-certificates \
        git \
        libssl-dev \
        unzip \
    && rm -rf /var/lib/apt/lists/*
RUN git clone --branch "${WRK_VERSION}" --depth 1 https://github.com/wg/wrk.git /src/wrk \
    && make -C /src/wrk -j"$(nproc)"

FROM debian:bookworm-slim AS target

COPY --from=volt-builder /out/benchmark-target /usr/local/bin/benchmark-target
EXPOSE 8080
ENTRYPOINT ["benchmark-target"]

FROM python:${PYTHON_VERSION}-slim-bookworm AS runner

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates libssl3 \
    && rm -rf /var/lib/apt/lists/*
COPY --from=volt-builder /out/volt /usr/local/bin/volt
COPY --from=external-go-tools /out/hey /out/k6 /usr/local/bin/
COPY --from=wrk-builder /src/wrk/wrk /usr/local/bin/wrk
COPY benchmarks/comparison/compare.py benchmarks/comparison/k6.js /opt/benchmark/
ENTRYPOINT ["python3", "/opt/benchmark/compare.py"]
