package common_utils

import "time"

const string_replacement_prefix = "STRING_REPLACEMENT"

func StrPTR(src string) *string {
	ret := new(string)
	*ret = src
	return ret
}

// "2026-01-06T00:00:00Z"
func StringToDate(dateString string) (*time.Time, error) {
	parsedTime, err := time.Parse(time.RFC3339, dateString)
	if err != nil {
		// This will happen if the string is not in the expected format
		return nil, err
	}

	return &parsedTime, nil
}
