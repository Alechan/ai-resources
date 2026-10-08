package fail

type Code int

const (
	ErrUnknown Code = iota
	ErrConfig
	ErrAuth
	ErrNetwork
)

func (c Code) String() string {
	switch c {
	case ErrConfig:
		return "CONFIG_ERROR"
	case ErrAuth:
		return "AUTH_ERROR"
	case ErrNetwork:
		return "NETWORK_ERROR"
	default:
		return "UNKNOWN_ERROR"
	}
}
