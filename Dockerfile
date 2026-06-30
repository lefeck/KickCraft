# Global ARGs must be declared before the first FROM to be usable in FROM instructions
ARG ROCKY_VERSION=9
ARG APP_VERSION=1.0
ARG RELEASE_TAG

# -----------------------
# Build Stage
# -----------------------
FROM golang:1.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kickcraft main.go

# -----------------------
# Runtime Stage (Rocky)
# -----------------------
FROM rockylinux/rockylinux:${ROCKY_VERSION}

# ARG must be re-declared after FROM to be available inside this stage
ARG ROCKY_VERSION=9
ARG APP_VERSION=1.0
ARG RELEASE_TAG

LABEL org.opencontainers.image.version=${APP_VERSION} \
      org.opencontainers.image.revision=${RELEASE_TAG} \
      org.opencontainers.image.base.name="rockylinux:${ROCKY_VERSION}"

ENV KICKCRAFT_VERSION=${APP_VERSION}
ENV KICKCRAFT_PORT=8080
ENV KICKCRAFT_MODE=release
ENV STATIC_DIR=/app/static
ENV TEMPLATE_DIR=/app/templates

# Install required packages
RUN dnf -y update && dnf -y install epel-release && dnf -y install --allowerasing \
    xorriso \
    p7zip \
    wget \
    curl \
    ca-certificates \
    && dnf clean all

WORKDIR /app
COPY --from=builder /app/kickcraft .
COPY --from=builder /app/static ${STATIC_DIR}
COPY --from=builder /app/templates ${TEMPLATE_DIR}

EXPOSE ${KICKCRAFT_PORT}

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://0.0.0.0:${KICKCRAFT_PORT}/health || exit 1

ENTRYPOINT ["/app/kickcraft"]
CMD ["--port", "8080", "--static-dir", "/app/static", "--templates-dir", "/app/templates"]
