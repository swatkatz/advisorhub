package graph

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

// est is the Eastern Time location used for all DateTime marshaling.
var est *time.Location

func init() {
	var err error
	est, err = time.LoadLocation("America/New_York")
	if err != nil {
		panic(fmt.Sprintf("loading America/New_York timezone: %v", err))
	}
}

// DateTime is a custom scalar that marshals as RFC3339 with EST offset.
type DateTime = time.Time

// Date is a custom scalar that marshals as "YYYY-MM-DD".
type Date = time.Time

// MarshalDateTime marshals a time.Time as an RFC3339 string with EST offset.
func MarshalDateTime(t time.Time) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Quote(t.In(est).Format(time.RFC3339)))
	})
}

// UnmarshalDateTime unmarshals an RFC3339 string into a time.Time.
func UnmarshalDateTime(v any) (time.Time, error) {
	s, ok := v.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("DateTime must be a string, got %T", v)
	}
	return time.Parse(time.RFC3339, s)
}

// MarshalDate marshals a time.Time as a "YYYY-MM-DD" date string.
func MarshalDate(t time.Time) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Quote(t.Format("2006-01-02")))
	})
}

// UnmarshalDate unmarshals a "YYYY-MM-DD" string into a time.Time.
func UnmarshalDate(v any) (time.Time, error) {
	s, ok := v.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("Date must be a string, got %T", v)
	}
	return time.Parse("2006-01-02", s)
}
