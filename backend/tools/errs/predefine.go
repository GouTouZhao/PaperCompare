package errs

const (
	ServerInternalError = 500
	ArgsError           = 1001
	TokenError          = 1401
	NoPermissionError   = 1403
	RecordNotFoundError = 1404
	DuplicateError      = 1409
	LLMError            = 1500
	RemoteNodeError     = 1502
)

var (
	ErrInternalServer = NewCodeError(ServerInternalError, "ServerInternalError")
	ErrArgs           = NewCodeError(ArgsError, "ArgError")
	ErrToken          = NewCodeError(TokenError, "TokenError")
	ErrNoPermission   = NewCodeError(NoPermissionError, "NoPermission")
	ErrRecordNotFound = NewCodeError(RecordNotFoundError, "RecordNotFound")
	ErrDuplicate      = NewCodeError(DuplicateError, "Duplicate")
	ErrLLM            = NewCodeError(LLMError, "LLMError")
	ErrRemoteNode     = NewCodeError(RemoteNodeError, "RemoteNodeError")
)
