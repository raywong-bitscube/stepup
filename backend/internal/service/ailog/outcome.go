package ailog

import "fmt"

// FormatOutcome combines business result_status with HTTP status for display when both matter.
func FormatOutcome(resultStatus string, httpStatus *int) string {
	if httpStatus == nil || *httpStatus == 0 {
		return resultStatus
	}
	if resultStatus == "success" && *httpStatus == 200 {
		return "success"
	}
	return fmt.Sprintf("%s · HTTP %d", resultStatus, *httpStatus)
}
