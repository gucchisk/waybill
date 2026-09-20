package domain

// Action は mediaType を持つオブジェクトに対して実行できる操作。
type Action int

const (
	ActionView Action = iota
	ActionDownload
)

func (action Action) Label() string {
	switch action {
	case ActionView:
		return "表示"
	case ActionDownload:
		return "ダウンロード"
	}
	return ""
}

// ActionsFor は mediaType ごとに選択可能な操作を返す。
// 未知の mediaType は +json で終わるなら表示とダウンロード、それ以外はダウンロードのみ。
func ActionsFor(mediaType MediaType) []Action {
	switch mediaType {
	case MediaTypeOCIEmpty:
		return []Action{ActionView}
	case MediaTypeOCILayerTar, MediaTypeOCILayerTarGzip, MediaTypeOCILayerTarZstd, MediaTypeDockerLayerGzip:
		return []Action{ActionDownload}
	}
	if mediaType.isJSON() {
		return []Action{ActionView, ActionDownload}
	}
	return []Action{ActionDownload}
}
