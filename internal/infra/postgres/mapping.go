package postgres

func NullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func NullableStr[T ~string](s *T) *string {
	if s == nil {
		return nil
	}
	return NullIfEmpty(string(*s))
}
