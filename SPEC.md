# waybill command spec

- waybillはCLIです
- コンテナイメージのmanifestを表示します(json)
- mediaTypeがあるオブジェクトを選択した場合、挙動が選択できます(mediatypeにより挙動が変わります)
    - application/vnd.oci.image.index.v1+json
        - 表示
        - ダウンロード
    - application/vnd.oci.image.manifest.v1+json
        - 表示
        - ダウンロード
    - application/vnd.oci.image.layer.v1.tar+gzip
        - ダウンロード
    - application/vnd.in-toto+json
        - 表示
        - ダウンロード
- json表示は人間に見やすい形で
- 選択しているobjectは反転
- カーソルは矢印またはEmacsバインドで移動
- mediaTypeがあるオブジェクトを選択している場合は、Enterで選択肢をPopup表示
    - 選択肢の選択はカーソル移動と同じキーバインド
    - 洗濯後にEnterで実行

## 起動方法

- `waybill <image-ref>`(例: `waybill ghcr.io/regclient/regctl:latest`)
- 認証・証明書はDockerの設定(`~/.docker/config.json`等)をregclient経由で利用する

## 画面

- 整形したJSONを行単位で表示する(キー・文字列・数値などは色分け)
- カーソル行が属する、`mediaType`と`digest`を持つ最内のobjectを反転表示する
    - `digest`を持たないobject(ルートのmanifest等)は取得先を特定できないため選択対象外
- 表示を選ぶと、取得したJSONを新しい画面で表示する(入れ子で辿れる)
- 表示できるサイズの上限は32MiB
- 取得・ダウンロード中はCtrl-C以外のキー操作を受け付けない

## キーバインド

| 操作 | キー |
| --- | --- |
| 上へ / 下へ | ↑ / ↓、Ctrl-p / Ctrl-n |
| ページ上 / ページ下 | PgUp / PgDn、Alt-v / Ctrl-v |
| 先頭 / 末尾 | Home / End、Alt-< / Alt-> |
| Popup表示 / 実行 | Enter |
| Popupを閉じる / 前の画面へ戻る | Esc、Ctrl-g、q |
| 終了 | Ctrl-c(最初の画面でEsc / qでも終了) |

## ダウンロード

- 保存先はカレントディレクトリ
- ファイル名はdigest由来(`:`を`-`に置換)+ 拡張子(例: `sha256-xxxx.tar.gz`)
    - `.tar.gz` / `.tar` / `.tar.zst`: 各layer、その他の`+json`は`.json`、上記以外は`.bin`
- 完了時は保存先パスをステータス行に表示する

## mediaTypeごとの操作

上記のほか、以下のmediaTypeにも対応する。

| mediaType | 操作 |
| --- | --- |
| application/vnd.oci.image.config.v1+json | 表示、ダウンロード |
| application/vnd.docker.distribution.manifest.v2+json | 表示、ダウンロード |
| application/vnd.docker.distribution.manifest.list.v2+json | 表示、ダウンロード |
| application/vnd.docker.container.image.v1+json | 表示、ダウンロード |
| application/vnd.dsse.envelope.v1+json | 表示、ダウンロード |
| application/vnd.dev.sigstore.bundle.v0.3+json | 表示、ダウンロード |
| application/vnd.dev.cosign.simplesigning.v1+json | 表示、ダウンロード |
| application/vnd.oci.empty.v1+json | 表示 |
| application/vnd.oci.image.layer.v1.tar | ダウンロード |
| application/vnd.oci.image.layer.v1.tar+zstd | ダウンロード |
| application/vnd.docker.image.rootfs.diff.tar.gzip | ダウンロード |
| 未知の`+json`で終わるmediaType | 表示、ダウンロード |
| 未知のその他のmediaType | ダウンロード |

## development

- go v1.27.1
- [regclient](https://github.com/regclient/regclient)
- [tcell](https://github.com/gdamore/tcell)
- クリーンアーキテクチャ
