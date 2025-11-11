package clickhouse

func BuildPlaceholders(count int) string {
	if count == 0 {
		return ""
	}
	placeholders := "?"
	for i := 1; i < count; i++ {
		placeholders += ",?"
	}
	return placeholders
}
