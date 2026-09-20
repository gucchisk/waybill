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

## development

- go v1.27.1
- [regclient](https://github.com/regclient/regclient)
- [tcell](https://github.com/gdamore/tcell)
- クリーンアーキテクチャ
