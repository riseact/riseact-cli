package stringutils

import (
	"regexp"
	"strings"
)

func Slugify(input string) string {
	s := strings.ToLower(input)

	reg, _ := regexp.Compile("[^a-zA-Z0-9 ]+")
	s = reg.ReplaceAllString(s, "")

	s = strings.Replace(s, " ", "-", -1)

	return s
}
