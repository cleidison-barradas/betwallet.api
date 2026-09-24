package postgres

func NullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
