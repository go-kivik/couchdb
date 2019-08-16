package testy

import (
	"regexp"
)

// Replacement is a replacement rule to be used on input before diffing.
type Replacement struct {
	Regexp      *regexp.Regexp
	Replacement string
}

func replace(t string, replacements ...Replacement) string {
	for _, re := range replacements {
		t = re.Regexp.ReplaceAllString(t, re.Replacement)
	}
	return t
}
