# Contributing

Use small, focused changes. Backend code should prefer the Go standard library and keep domain logic in pure packages where possible. Frontend changes should preserve the operational app surface: dense, readable, keyboard-friendly, and free of marketing-only screens.

Before opening a PR, run:

```sh
go test ./...
cd webui && npm run lint
```

