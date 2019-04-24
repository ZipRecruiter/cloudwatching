package main

import (
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/prometheus/client_golang/prometheus"
)

type exportConfig struct {
	Namespace, Name string

	Dimensions, Statistics []string

	DimensionsMatch, DimensionsNoMatch map[string]string

	dimensionsMatch, dimensionsNoMatch map[string]*regexp.Regexp

	// each collector maps to the statistic in the same location
	collectors []*prometheus.GaugeVec
}

type configuration struct {
	Region string
	Debug  bool

	ExportConfigs []exportConfig
}

func (e exportConfig) isDynamodDBIndexMetric() bool {
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

func (e exportConfig) String(i int) string {
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

// XXX metrics for AWS API
func (c *configuration) Validate() error {
	// using i because we are mutating the values
	for i := range c.ExportConfigs {
		if len(c.ExportConfigs[i].Statistics) == 0 {
			return errors.New("At least one statistic is required")
		}

		// these to cheaply compare to other list at runtime
		sort.Strings(c.ExportConfigs[i].Dimensions)

		c.ExportConfigs[i].dimensionsMatch = make(map[string]*regexp.Regexp, len(c.ExportConfigs[i].DimensionsMatch))
		c.ExportConfigs[i].dimensionsNoMatch = make(map[string]*regexp.Regexp, len(c.ExportConfigs[i].DimensionsNoMatch))

		for k, v := range c.ExportConfigs[i].DimensionsMatch {
			// verify that we are matching against dimensions we are going to be
			// using
			var found bool
			for _, d := range c.ExportConfigs[i].Dimensions {
				if k == d {
					found = true
					break
				}
			}
			if !found {
				return errors.New("DimensionsMatch name not in Dimensions")
			}

			re, err := regexp.Compile(v)
			if err != nil {
				return err
			}

			c.ExportConfigs[i].dimensionsMatch[k] = re
		}

		for k, v := range c.ExportConfigs[i].DimensionsNoMatch {
			// verify that we are matching against dimensions we are going to be
			// using
			var found bool
			for _, d := range c.ExportConfigs[i].Dimensions {
				if k == d {
					found = true
					break
				}
			}
			if !found {
				return errors.New("DimensionsNoMatch name not in Dimensions")
			}

			re, err := regexp.Compile(v)
			if err != nil {
				return err
			}

			c.ExportConfigs[i].dimensionsNoMatch[k] = re
		}

		c.ExportConfigs[i].collectors = make([]*prometheus.GaugeVec, len(c.ExportConfigs[i].Statistics))
		aliasedDimensions := make([]string, len(c.ExportConfigs[i].Dimensions))
		for j, d := range c.ExportConfigs[i].Dimensions {
			aliasedDimensions[j] = pascalToUnderScores(d)
		}

		for j := range c.ExportConfigs[i].Statistics {
			c.ExportConfigs[i].collectors[j] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: c.ExportConfigs[i].String(j),
				Help: "",
			}, aliasedDimensions)
			if err := prometheus.Register(c.ExportConfigs[i].collectors[j]); err != nil {
				return errors.Wrap(err, "Namespace="+c.ExportConfigs[i].Namespace+" Name="+c.ExportConfigs[i].Name)
			}
		}
	}

	return nil
}

func defaultConfig() configuration {
	var c configuration
	c.Region = "us-east-1"
	c.ExportConfigs = []exportConfig{
		{
			Namespace:  "AWS/SQS",
			Name:       "ApproximateAgeOfOldestMessage",
			Dimensions: []string{"QueueName"},
			Statistics: []string{"Maximum"},

			DimensionsNoMatch: map[string]string{"QueueName": "(STAGING|STG|stg|dev|DEV|botoutil)"},
		},
		{
			Namespace:  "AWS/SQS",
			Name:       "NumberOfMessagesDeleted",
			Dimensions: []string{"QueueName"},
			Statistics: []string{"Sum"},

			DimensionsNoMatch: map[string]string{"QueueName": "(STAGING|STG|stg|dev|DEV|botoutil)"},
		},
	}

	if err := c.Validate(); err != nil {
		panic(err.Error())
	}

	return c
}
