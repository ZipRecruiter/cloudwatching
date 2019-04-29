package exportcloudwatch

import (
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/prometheus/client_golang/prometheus"
)

// ExportConfig describes which cloudwatch metrics we want to export.  Make sure
// you call Validate.
type ExportConfig struct {
	// Namespace and Name map directly to metrics in AWS Cloudwatch
	Namespace, Name string

	// Dimensions are the names of dimensions to pull the data of, and Statistics
	// are the Statistics to pull
	Dimensions, Statistics []string

	// Parallel array with Statistics to transform the values. Can be empty or
	// same length as Statistics.
	Transform []func(float64) float64

	// Both of these filter the metrics based on the values of the dimension
	DimensionsMatch, DimensionsNoMatch map[string]*regexp.Regexp

	// A slice of GaugeVecs.  Leave empty and they'll be generated and registered
	// for you.
	Collectors []*prometheus.GaugeVec
}

func (e *ExportConfig) isDynamodDBIndexMetric() bool {
	if e.Namespace != "AWS/DynamoDB" {
		return false
	}

	for _, d := range e.Dimensions {
		if d == "GlobalSecondaryIndexName" {
			return true
		}
	}

	return false
}

func (e *ExportConfig) String(i int) string {
	var base string
	if e.isDynamodDBIndexMetric() {
		base = e.Name + "Index" + e.Statistics[i]
	} else {
		base = e.Name + e.Statistics[i]
	}
	base = strings.ToLower(e.Namespace) + "_" + pascalToUnderScores(base)
	base = strings.ReplaceAll(base, "/", "_")

	return base
}

// Validate returns an error if the configuration is incorrect and registers
// each metric with the default prometheus registry.
func (e *ExportConfig) Validate() error {
	if len(e.Statistics) == 0 {
		return errors.New("At least one statistic is required")
	}

	// these to cheaply compare to other list at runtime
	sort.Strings(e.Dimensions)

	for k := range e.DimensionsMatch {
		// verify that we are matching against dimensions we are going to be
		// using
		var found bool
		for _, d := range e.Dimensions {
			if k == d {
				found = true
				break
			}
		}
		if !found {
			return errors.New("DimensionsMatch name not in Dimensions")
		}
	}

	for k := range e.DimensionsNoMatch {
		// verify that we are matching against dimensions we are going to be
		// using
		var found bool
		for _, d := range e.Dimensions {
			if k == d {
				found = true
				break
			}
		}
		if !found {
			return errors.New("DimensionsNoMatch name not in Dimensions")
		}
	}

	if len(e.Collectors) == 0 {
		e.Collectors = make([]*prometheus.GaugeVec, len(e.Statistics))
		aliasedDimensions := make([]string, len(e.Dimensions))
		for j, d := range e.Dimensions {
			aliasedDimensions[j] = pascalToUnderScores(d)
		}

		for j := range e.Statistics {
			e.Collectors[j] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: e.String(j),
				Help: "",
			}, aliasedDimensions)
			if err := prometheus.Register(e.Collectors[j]); err != nil {
				return errors.Wrap(err, "Namespace="+e.Namespace+" Name="+e.Name)
			}
		}
	}

	if len(e.Collectors) != len(e.Statistics) {
		return errors.Wrap(errors.New("Collectors length not equal to Statistics length"), "namespace="+e.Namespace+" name="+e.Name)
	}

	if len(e.Transform) != 0 && len(e.Transform) != len(e.Statistics) {
		return errors.Wrap(errors.New("Transform length not equal to Statistics length"), "namespace="+e.Namespace+" name="+e.Name)
	}

	return nil
}
