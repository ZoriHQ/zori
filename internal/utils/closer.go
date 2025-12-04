package utils

import (
	"fmt"
	"net/http"
)

func EnsureBodyClosed(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		if err := resp.Body.Close(); err != nil {
			fmt.Println("Error closing body: ", err.Error())
		}
	}
}
