package clickhouse

import (
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

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

func EnsureClosed(chRows driver.Rows) {
	if err := chRows.Close(); err != nil {
		fmt.Println("Failed to close rows: ", err.Error())
	}
}
