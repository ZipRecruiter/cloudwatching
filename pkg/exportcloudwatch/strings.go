package exportcloudwatch

import (
	"regexp"
	"strings"
)

var re = regexp.MustCompile("[A-Z][a-z0-9_]+")

func cloudWatchToPrometheusNameV0(in string) string {
	found := re.FindAllString(in, -1)

	ret := strings.ToLower(found[0])
	for _, s := range found[1:] {
		ret += "_" + strings.ToLower(s)
	}

	return ret
}

var (
	nonAlphaCharsRE   = regexp.MustCompile("[^a-zA-Z0-9]")
	pascalCaseWordsRE = regexp.MustCompile("([A-Z0-9]*)([a-z]*)")
)

func cloudWatchToPrometheusNameV1(in string) string {
	words := make([]string, 0)

	// CloudWatch metrics follow a "SequenceOfPascalCaseWords" naming scheme, for the most part - so
	// first split up the input string by non-alphanumeric characters first…
	for _, chunk := range nonAlphaCharsRE.Split(in, -1) {
		// …and then detect Pascal-case-y words.
		for _, captures := range pascalCaseWordsRE.FindAllStringSubmatch(chunk, -1) {
			word := captures[0]
			upperCasePrefix := captures[1]
			lowerCaseSuffix := captures[2]

			// we need to match a sequence of uppercase/number followed by a sequence of lowercase -
			// either (but not both) may be absent, but handling the absence of both purely in regexp
			// is less clear than just tolerating it in the FindAllStringSubmatch results, detecting
			// that scenario, and moving on - both with the algorithm and our lives
			if len(word) == 0 {
				continue
			}

			// sometimes a word is an acronym, such as "TCP"…
			isAcronym := len(upperCasePrefix) >= 2

			// …which can be pluralized (eg. "CPUs", "LCUs")
			isPluralized := lowerCaseSuffix == "s"

			// …or concatenated with another Pascal-case word (eg. "ADAnomaly")
			hasAdjacentWord := lowerCaseSuffix != ""

			if isAcronym && !isPluralized && hasAdjacentWord {
				// if we see a non-plural acronym that's concatenated with another word,
				// we treat everything but the last uppercase character we found as part
				// of the acronym:
				words = append(words, upperCasePrefix[:len(upperCasePrefix)-1])

				// …and the last uppercase character as the first letter
				// of the next word:
				words = append(words, upperCasePrefix[len(upperCasePrefix)-1:]+lowerCaseSuffix)
			} else {
				// in all other situations, it's just a single word, so just use that
				words = append(words, word)
			}
		}
	}

	return strings.ToLower(strings.Join(words, "_"))
}
