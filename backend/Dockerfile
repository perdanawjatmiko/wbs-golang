# syntax=docker/dockerfile:1
ARG GO_IMAGE=golang:1.26.11-bookworm
FROM ${GO_IMAGE} AS build
WORKDIR /src
ENV GOTOOLCHAIN=local CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN mkdir -p /out /runtime/var/private/attachments && chmod 0700 /runtime/var/private/attachments \
 && chown -R 10001:10001 /runtime/var \
 && for cmd in api worker migrate seed check; do go build -trimpath -ldflags='-s -w' -o /out/$cmd ./cmd/$cmd; done

FROM scratch AS runtime
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /usr/local/go/lib/time/zoneinfo.zip /usr/local/go/lib/time/zoneinfo.zip
COPY --from=build /out/ /app/
COPY --from=build --chown=10001:10001 /runtime/var/ /var/
USER 10001:10001
ENV HTTP_HOST=0.0.0.0 HTTP_PORT=8080 UPLOAD_DIR=/var/private/attachments
EXPOSE 8080
STOPSIGNAL SIGTERM
HEALTHCHECK --interval=15s --timeout=3s --start-period=15s --retries=3 CMD ["/app/check", "live"]
CMD ["/app/api"]
