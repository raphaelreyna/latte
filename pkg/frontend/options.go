package frontend

import templatingengine "github.com/raphaelreyna/latte/pkg/template/templating-engine"

type OnMissingKey string

func (k OnMissingKey) Valid() bool {
	return k == OnMissingKey_Err ||
		k == OnMissingKey_Zero ||
		k == OnMissingKey_Nothing
}

const (
	OnMissingKey_Err     OnMissingKey = "error"
	OnMissingKey_Zero    OnMissingKey = "zero"
	OnMissingKey_Nothing OnMissingKey = "nothing"
)

var MkMap = map[OnMissingKey]templatingengine.MissingKeyHandler{
	OnMissingKey_Err:     templatingengine.MissingKeyHandler_Error,
	OnMissingKey_Zero:    templatingengine.MissingKeyHandler_ZeroValue,
	OnMissingKey_Nothing: templatingengine.MissingKeyHandler_Nothing,
}
