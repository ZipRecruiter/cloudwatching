## monitoring--cloudwatch

This is a reimplementation of the official
[cloudwatch-exporter](https://github.com/prometheus/cloudwatch_exporter).  The
main motivation is to use GetMetricData instead of GetMetricStatistics, which
can lead to a reduction in AWS API calls by a factor of about 100.

Hope this helps!

fREW Schmidt

---

## Flow of the code:

 1. Each ExportConfig defines a pattern of metrics to export.

 2. metricsToRead uses the config to ListMetrics and create a metricStat for
    each metric.

 3. unrollMetrics expands metricStats into a map that can be used to directly
    update metrics.

 4. readMetrics uses GetMetricData in bundles of 100 at a time to load all of
    the cloudwatch metrics into the relevant prometheus metrics.
