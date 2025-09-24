package common_utils

const string_replacement_prefix = "STRING_REPLACEMENT"

func StrPTR(src string) *string {
	ret := new(string)
	*ret = src
	return ret
}
