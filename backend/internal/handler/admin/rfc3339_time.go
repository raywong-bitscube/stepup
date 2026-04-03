package admin

import (
	"encoding/json"
	"time"
)

// RFC3339Time encodes time.Time as UTC RFC3339Nano in JSON.
type RFC3339Time time.Time

func (t RFC3339Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).UTC().Format(time.RFC3339Nano))
}
