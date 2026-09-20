# waybill

コンテナイメージの manifest をターミナル上で閲覧・辿り・ダウンロードできる CLI ツールです。

image index → manifest → config / layer / attestation(in-toto, cosign, sigstore bundle など)と、JSON 内の `digest` 参照を対話的に辿れます。

## 特長

- 整形・色分けされた JSON を行単位で表示
- カーソル位置の `mediaType` + `digest` を持つ object を反転表示
- 選択した object を新しい画面で表示(入れ子で辿れる)、またはカレントディレクトリへダウンロード
- 矢印キーと Emacs バインドの両方で操作可能
- Docker の設定(`~/.docker/config.json` 等)を [regclient](https://github.com/regclient/regclient) 経由で利用するため、認証・証明書設定はそのまま使える

## インストール

Go 1.27.1 以上が必要です。

```sh
go install github.com/gucchisk/waybill/cmd/waybill@latest
```

ソースからビルドする場合:

```sh
git clone https://github.com/gucchisk/waybill.git
cd waybill
go build -o waybill ./cmd/waybill
```

## 使い方

```sh
waybill <image-ref>
```

例:

```sh
waybill ghcr.io/regclient/regctl:latest
```

| オプション | 説明 |
| --- | --- |
| `-h`, `--help` | 使い方を表示 |

### 基本的な流れ

1. 起動するとイメージの manifest が JSON で表示されます。
2. カーソルを動かすと、`mediaType` と `digest` を持つ最内の object が反転表示されます。
3. `Enter` で操作(表示 / ダウンロード)を選ぶ popup が開きます。もう一度 `Enter` で実行します。
4. 「表示」を選ぶと取得した JSON が新しい画面で開きます。`Esc` / `q` で前の画面に戻ります。
5. 「ダウンロード」を選ぶとカレントディレクトリに保存され、保存先パスがステータス行に表示されます。

`digest` を持たない object(ルートの manifest など)は取得先を特定できないため、選択対象外です。

## キーバインド

| 操作 | キー |
| --- | --- |
| 上へ / 下へ | `↑` / `↓`、`Ctrl-p` / `Ctrl-n` |
| ページ上 / ページ下 | `PgUp` / `PgDn`、`Alt-v` / `Ctrl-v` |
| 先頭 / 末尾 | `Home` / `End`、`Alt-<` / `Alt->` |
| popup 表示 / 実行 | `Enter` |
| popup を閉じる / 前の画面へ戻る | `Esc`、`Ctrl-g`、`q` |
| 終了 | `Ctrl-c`(最初の画面では `Esc` / `q` でも終了) |

取得・ダウンロード中は `Ctrl-c` 以外のキー操作を受け付けません。

## ダウンロード

- 保存先: カレントディレクトリ
- ファイル名: digest の `:` を `-` に置換したものに拡張子を付与(例: `sha256-xxxx.tar.gz`)
    - layer: `.tar.gz` / `.tar` / `.tar.zst`
    - その他の `+json`: `.json`
    - 上記以外: `.bin`

## mediaType ごとの操作

| mediaType | 操作 |
| --- | --- |
| `application/vnd.oci.image.index.v1+json` | 表示、ダウンロード |
| `application/vnd.oci.image.manifest.v1+json` | 表示、ダウンロード |
| `application/vnd.oci.image.config.v1+json` | 表示、ダウンロード |
| `application/vnd.oci.image.layer.v1.tar+gzip` | ダウンロード |
| `application/vnd.oci.image.layer.v1.tar` | ダウンロード |
| `application/vnd.oci.image.layer.v1.tar+zstd` | ダウンロード |
| `application/vnd.oci.empty.v1+json` | 表示 |
| `application/vnd.docker.distribution.manifest.v2+json` | 表示、ダウンロード |
| `application/vnd.docker.distribution.manifest.list.v2+json` | 表示、ダウンロード |
| `application/vnd.docker.container.image.v1+json` | 表示、ダウンロード |
| `application/vnd.docker.image.rootfs.diff.tar.gzip` | ダウンロード |
| `application/vnd.in-toto+json` | 表示、ダウンロード |
| `application/vnd.dsse.envelope.v1+json` | 表示、ダウンロード |
| `application/vnd.dev.sigstore.bundle.v0.3+json` | 表示、ダウンロード |
| `application/vnd.dev.cosign.simplesigning.v1+json` | 表示、ダウンロード |
| 未知の `+json` で終わる mediaType | 表示、ダウンロード |
| 未知のその他の mediaType | ダウンロード |

## 制限事項

- 画面に表示できる JSON のサイズは最大 32MiB です。

## 開発

- Go 1.27.1
- [regclient](https://github.com/regclient/regclient)(レジストリアクセス)
- [tcell](https://github.com/gdamore/tcell)(TUI)
- クリーンアーキテクチャ(`internal/domain` / `usecase` / `adapter` / `infrastructure`)

```sh
go build ./...
go test ./...
```

仕様の詳細は [SPEC.md](SPEC.md) を参照してください。
