package exportcloudwatch

import (
	"regexp"
	"strings"
)

var re = regexp.MustCompile("[A-Z][a-z0-9_]+")

func pascalToUnderScores(in string) string {
	found := re.FindAllString(in, -1)

	ret := strings.ToLower(found[0])
	for _, s := range found[1:] {
		ret += "_" + strings.ToLower(s)
	}

	return ret
}
