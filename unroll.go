package main

import (
	"fmt"
)

func unrollMetrics(ms []metricStat) map[string]metricStat {
	ret := make(map[string]metricStat, len(ms))

	for i, m := range ms {
		ret[fmt.Sprintf("i%d", i)] = m
	}

	return ret
}
