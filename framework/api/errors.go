package api

func NotFound(message string) APIError {
	return APIError{Code: "not_found", Message: message}
}

func Unauthorized(message string) APIError {
	return APIError{Code: "unauthorized", Message: message}
}

func Validation(field, message string) APIError {
	return APIError{Field: field, Code: "validation_error", Message: message}
}
