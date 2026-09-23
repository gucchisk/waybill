package domain

type Action int

const (
	ActionView Action = iota
	ActionDownload
)

func (action Action) ProgressLabel() string {
	switch action {
	case ActionView:
		return "Viewing"
	case ActionDownload:
		return "Downloading"
	}
	return ""
}

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
