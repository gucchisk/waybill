# Development

- Go 1.27.1
- [cobra](https://github.com/spf13/cobra) (CLI commands, arguments and flags)
- [regclient](https://github.com/regclient/regclient) (registry access)
- [tcell](https://github.com/gdamore/tcell) (TUI)
- Clean architecture (`internal/domain` / `usecase` / `adapter` / `infrastructure`)

```sh
go build ./...
go test ./...
```

## Release

Push a `v*` tag to release:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The tag must match `^v[0-9A-Za-z][0-9A-Za-z._-]*$` (e.g. `v1.2.3`, `v1.0.0-rc.1`), because it becomes part of the release asset URL; otherwise the workflow fails. The `release` workflow then runs the tests, creates the GitHub Release with a source tarball (`waybill-<version>.tar.gz`, made with `git archive`) as an asset, and opens a pull request on [gucchisk/homebrew-tap](https://github.com/gucchisk/homebrew-tap) that points `Formula/waybill.rb` at that tarball. The formula does not use GitHub's generated `archive/refs/tags/*.tar.gz`, whose checksum can change when GitHub recompresses it. On the tap, `brew test-bot` builds bottles for macOS and Linux; once it succeeds on every OS, `brew pr-pull` runs automatically, uploads the bottles to the tap's GitHub Release, and pushes the `bottle do` block to the tap's `main` (the pull request is closed, not merged). Only pull requests from `waybill-*` branches of the tap are published automatically; for any other pull request, add the `pr-pull` label.

The workflow needs a `HOMEBREW_TAP_TOKEN` secret: a personal access token with Contents and Pull requests write access to the tap repository.
