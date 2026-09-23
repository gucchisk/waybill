# waybill command spec

- waybillはCLIです
- コンテナイメージのmanifestを表示します(json)
- mediaTypeがあるオブジェクトを選択した場合、挙動が選択できます(mediaTypeにより挙動が変わります。詳細は「mediaTypeごとの操作」を参照)
- json表示は人間に見やすい形で
- 選択しているobjectは反転
- カーソルは矢印またはEmacsバインドで移動
- mediaTypeがあるオブジェクトを選択している場合は、Enterで選択肢をPopup表示
    - 選択肢の選択はカーソル移動と同じキーバインド
    - 洗濯後にEnterで実行

## 起動方法

- `waybill <image-ref>`(例: `waybill ghcr.io/regclient/regctl:latest`)
- 引数が`<image-ref>`ひとつ以外の場合はエラーメッセージと使い方(usage)を表示して異常終了する(終了コード1)
- `-h` / `--help`で使い方を表示する
- `-v` / `--version`でバージョンを表示する(ビルド時に`-ldflags "-X main.version=<version>"`で埋め込む。未指定の場合は`dev`)
- `--theme auto|dark|light`で背景の明暗に応じたハイライト色を選択する(既定値は`auto`)
- 認証・証明書はDockerの設定(`~/.docker/config.json`等)をregclient経由で利用する

## 画面

- 画面上のメッセージ・ラベル(操作名は`View` / `Download`、ヘルプ、ステータス行など)およびCLIのエラーメッセージは英語で表示する
- 整形したJSONを行単位で表示する(キー・文字列・数値などは色分け)
- カーソル行は行全体を前景色と背景色を反転して表示する(行選択表示。`>`などのマーカーは表示しない。行内の色分けは行わず、単色にする。暗い背景では端末のデフォルトの前景色・背景色を入れ替え、明るい背景では反転すると真っ黒で見づらいため、Bright Black(8)の背景にWhite(15)の文字で表示する)
- カーソル行が属する、`mediaType`と`digest`を持つ最内のobjectを薄い背景色でハイライト表示する(カーソル行以外の行に適用し、カーソル行は反転表示が優先される)
    - 薄い背景色は端末の背景色の明暗で切り替える。`--theme`が`auto`(既定値)の場合、起動時に端末へ背景色を問い合わせ(OSC 11。termenvを利用)、暗い背景ならANSIのBright Black(8)、明るい背景ならWhite(7)を使う
    - 端末が問い合わせに応答しない場合(tmuxなど環境によって応答しないことがある)は暗い背景として扱う(Bright Black)。その場合は`--theme dark`/`--theme light`で明示的に指定できる
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

mediaTypeによる挙動は以下のテーブルに集約する。

| mediaType | 操作 |
| --- | --- |
| application/vnd.oci.image.index.v1+json | 表示、ダウンロード |
| application/vnd.oci.image.manifest.v1+json | 表示、ダウンロード |
| application/vnd.oci.image.layer.v1.tar+gzip | ダウンロード |
| application/vnd.in-toto+json | 表示、ダウンロード |
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

## 配布・リリース

- Homebrewのtap(`gucchisk/homebrew-tap`)でbottleとして配布する(`brew install gucchisk/tap/waybill`)
- `v*`のタグをpushすると、GitHub Actions(`.github/workflows/release.yml`)が以下を行う
    1. `go test`を実行
    2. `git archive`でソースtarball(`waybill-<version>.tar.gz`)を作成
    3. waybillのGitHub Releaseを作成し、ソースtarballをアセットとしてアップロード(リリースノートは自動生成)
    4. tapの`Formula/waybill.rb`のurl / sha256をアップロードしたソースtarballに更新するPRを作成
- FormulaのurlにはGitHubが自動生成するアーカイブ(`archive/refs/tags/*.tar.gz`)を使わない(再圧縮でsha256が変わることがあるため)
- tap側ではPRに対して`brew test-bot`がmacOS各版・Linuxのbottleをビルドし、`pr-pull`ラベルを付けると`brew pr-pull`がbottleをtapのGitHub Releaseにアップロードし、Formulaに`bottle do`ブロックを追加する
- tapへのPR作成には、tapリポジトリへのContents / Pull requestsの書き込み権限を持つPATをsecret `HOMEBREW_TAP_TOKEN`に登録しておく

## development

- go v1.27.1
- [cobra](https://github.com/spf13/cobra)(CLIのコマンド・引数・フラグ処理)
- [regclient](https://github.com/regclient/regclient)
- [tcell](https://github.com/gdamore/tcell)
- クリーンアーキテクチャ
