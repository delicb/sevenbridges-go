package sevenbridges

import (
	"fmt"
	"strconv"
	"time"
)

// Timestamp is serialization helper that know how to serialize to and from
// timestamp format for JSON serialization. It is only alias for time.Time
// that implements Marshaler and Unmarshaler interfaces so it is always safe
// to convert it to time.Time if time arithmetics is needed.
type Timestamp struct {
	time.Time
}

// MarshalJSON marshals timestamp to its timestamp value and returns it as
// byte array.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	ts := t.Unix()
	stamp := fmt.Sprint(ts)
	return []byte(stamp), nil
}

// UnmarshalJSON unmarshals provided byte array, treats it as a string
// that contains timestamp and is Unix timestamp of a specific time and
// populates t with that value.
func (t *Timestamp) UnmarshalJSON(b []byte) error {
	str := string(b)
	// if we got null, set time to nil without error, it is valid situation
	if str == "null" {
		t = nil
		return nil
	}
	i, err := strconv.ParseInt(str, 10, 64)
	if err == nil {
		t.Time = time.Unix(i, 0)
	} else {
		t.Time, err = time.Parse(`"`+time.RFC3339+`"`, str)
	}
	return err
}
