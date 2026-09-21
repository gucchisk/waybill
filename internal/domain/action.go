package domain

// Action is an operation that can be performed on an object that has a mediaType.
type Action int

const (
	ActionView Action = iota
	ActionDownload
)

func (action Action) Label() string {
	switch action {
	case ActionView:
		return "View"
	case ActionDownload:
		return "Download"
	}
	return ""
}

// ActionsFor returns the actions available for each mediaType.
// An unknown mediaType gets view and download if it ends with +json, and download only otherwise.
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
