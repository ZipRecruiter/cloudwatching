package exportcloudwatch

import (
	"sort"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

func includeMetric(e ExportConfig, m *cloudwatch.Metric) bool {
	if len(m.Dimensions) != len(e.Dimensions) {
		return false
	}

	got := make([]string, 0, len(e.Dimensions))

	for _, d := range m.Dimensions {
		if re := e.dimensionsNoMatch[*d.Name]; re != nil && re.MatchString(*d.Value) {
			return false
		}
		if re := e.dimensionsMatch[*d.Name]; re != nil && !re.MatchString(*d.Value) {
			return false
		}
		got = append(got, *d.Name)
	}

	expect := e.Dimensions

	// expect is already sorted when validating the config
	sort.Strings(got)

	for i, e := range expect {
		if e != got[i] {
			return false
		}
	}

	return true
}
