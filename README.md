# waybill

A CLI tool for browsing, following and downloading container image manifests in the terminal.

You can interactively walk from an image index → manifest → config / layer / attestation (in-toto, cosign, sigstore bundle, etc.) by following the `digest` references inside the JSON.

## Features

- Shows pretty-printed, colorized JSON line by line
- Highlights (with a subtle background color) the innermost object under the cursor that has both `mediaType` and `digest`
- Opens the selected object on a new screen (nestable), or downloads it to the current directory
- Supports both arrow keys and Emacs key bindings
- Uses Docker's configuration (`~/.docker/config.json`, etc.) through [regclient](https://github.com/regclient/regclient), so existing authentication and certificate settings just work

## Installation

Go 1.27.1 or later is required.

```sh
go install github.com/gucchisk/waybill/cmd/waybill@latest
```

To build from source:

```sh
git clone https://github.com/gucchisk/waybill.git
cd waybill
go build -o waybill ./cmd/waybill
```

## Usage

```sh
waybill <image-ref>
```

Example:

```sh
waybill ghcr.io/regclient/regctl:latest
```

| Option | Description |
| --- | --- |
| `-h`, `--help` | Show usage |

If `<image-ref>` is missing or more than one is given, waybill prints the error message and the usage, then exits with status 1.

### Basic flow

1. On startup, the image manifest is shown as JSON.
2. As you move the cursor, the innermost object that has `mediaType` and `digest` is highlighted.
3. Press `Enter` to open a popup for choosing an action (View / Download). Press `Enter` again to run it.
4. Choosing "View" opens the fetched JSON on a new screen. Press `Esc` / `q` to go back.
5. Choosing "Download" saves the content to the current directory and shows the saved path in the status line.

Objects without a `digest` (such as the root manifest) cannot be selected because there is no way to tell where to fetch them from.

## Key bindings

| Action | Keys |
| --- | --- |
| Up / Down | `↑` / `↓`, `Ctrl-p` / `Ctrl-n` |
| Page up / Page down | `PgUp` / `PgDn`, `Alt-v` / `Ctrl-v` |
| Top / Bottom | `Home` / `End`, `Alt-<` / `Alt->` |
| Show popup / Run | `Enter` |
| Close popup / Go back | `Esc`, `Ctrl-g`, `q` |
| Quit | `Ctrl-c` (`Esc` / `q` also quit on the first screen) |

While fetching or downloading, all keys except `Ctrl-c` are ignored.

## Download

- Destination: the current directory
- File name: the digest with `:` replaced by `-`, plus an extension (e.g. `sha256-xxxx.tar.gz`)
    - Layers: `.tar.gz` / `.tar` / `.tar.zst`
    - Other `+json` types: `.json`
    - Everything else: `.bin`

## Actions per mediaType

| mediaType | Actions |
| --- | --- |
| `application/vnd.oci.image.index.v1+json` | View, Download |
| `application/vnd.oci.image.manifest.v1+json` | View, Download |
| `application/vnd.oci.image.config.v1+json` | View, Download |
| `application/vnd.oci.image.layer.v1.tar+gzip` | Download |
| `application/vnd.oci.image.layer.v1.tar` | Download |
| `application/vnd.oci.image.layer.v1.tar+zstd` | Download |
| `application/vnd.oci.empty.v1+json` | View |
| `application/vnd.docker.distribution.manifest.v2+json` | View, Download |
| `application/vnd.docker.distribution.manifest.list.v2+json` | View, Download |
| `application/vnd.docker.container.image.v1+json` | View, Download |
| `application/vnd.docker.image.rootfs.diff.tar.gzip` | Download |
| `application/vnd.in-toto+json` | View, Download |
| `application/vnd.dsse.envelope.v1+json` | View, Download |
| `application/vnd.dev.sigstore.bundle.v0.3+json` | View, Download |
| `application/vnd.dev.cosign.simplesigning.v1+json` | View, Download |
| Unknown mediaType ending with `+json` | View, Download |
| Any other unknown mediaType | Download |

## Limitations

- The maximum size of JSON that can be shown on screen is 32MiB.

## Development

- Go 1.27.1
- [cobra](https://github.com/spf13/cobra) (CLI commands, arguments and flags)
- [regclient](https://github.com/regclient/regclient) (registry access)
- [tcell](https://github.com/gdamore/tcell) (TUI)
- Clean architecture (`internal/domain` / `usecase` / `adapter` / `infrastructure`)

```sh
go build ./...
go test ./...
```

See [SPEC.md](SPEC.md) for the detailed specification (written in Japanese).
