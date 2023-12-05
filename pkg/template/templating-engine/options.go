package templatingengine

type MissingKeyHandler string

const (
	// MK_Error will cause an error if details is missing a key used in the template
	MissingKeyHandler_Error MissingKeyHandler = "error"
	// MK_Zero will cause values whose keys are missing from details to be replace with a zero value.
	MissingKeyHandler_ZeroValue MissingKeyHandler = "zero"
	// MK_Nothing will cause missing keys to be ignored.
	MissingKeyHandler_Nothing MissingKeyHandler = "nothing"
)

func (mkh MissingKeyHandler) Valid() bool {
	return mkh == MissingKeyHandler_Error || mkh == MissingKeyHandler_ZeroValue || mkh == MissingKeyHandler_Nothing
}

func (mkh MissingKeyHandler) Val() string {
	switch mkh {
	case "":
		fallthrough
	case MissingKeyHandler_Nothing:
		return "default"
	default:
		return string(mkh)
	}
}
