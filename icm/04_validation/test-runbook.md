# Validation runbook

For local end-to-end testing of the publisher (LiveKit dev server, allowlist
updates via stdin/SIGUSR1, subscriber enforcement), see
`local-testing-guide.md` and the helper scripts in `scripts/`.

```shell
gofmt -l .
go build ./...   # requires CGO + libgstreamer1.0-dev, libgstreamer-plugins-base1.0-dev
go vet ./...     # same CGO requirement
```

If CGO/GStreamer headers are unavailable in the local environment, record that
in the change validation evidence and rely on `gofmt` + targeted review.
