package nullable

func StringOr(value *string, def string) string {
	if value == nil {
		return def
	}
	return *value
}
