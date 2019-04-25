## monitoring--cloudwatch

This is a reimplementation of the official
[cloudwatch-exporter](https://github.com/prometheus/cloudwatch_exporter).  The
main motivation is to use GetMetricData instead of GetMetricStatistics, which
can lead to a reduction in AWS API calls by a factor of about 100.

## Quickstart

```
go get -u github.com/ZipRecruiter/monitoring--cloudwatch
```

Copy paste the following config to `~/mc.json`:

```
{
  "exportconfigs": [
    {
      "dimensions": [
        "QueueName"
      ],
      "dimensionsMatch": {
        "QueueName": "(?i:prod)"
      },
      "dimensionsNoMatch": {
        "QueueName": "(?i:dev|staging)"
      },
      "name": "ApproximateAgeOfOldestMessage",
      "namespace": "AWS/SQS",
      "statistics": [
        "Maximum"
      ]
    },
    {
      "dimensions": [
        "QueueName"
      ],
      "dimensionsMatch": {
        "QueueName": "(?i:prod)"
      },
      "dimensionsNoMatch": {
        "QueueName": "(?i:dev|staging)"
      },
      "name": "NumberOfMessagesReceived",
      "namespace": "AWS/SQS",
      "statistics": [
        "Maximum"
      ]
    },
    {
      "dimensions": [
        "QueueName"
      ],
      "dimensionsMatch": {
        "QueueName": "(?i:prod)"
      },
      "dimensionsNoMatch": {
        "QueueName": "(?i:dev|staging)"
      },
      "name": "NumberOfMessagesDeleted",
      "namespace": "AWS/SQS",
      "statistics": [
        "Maximum"
      ]
    }
  ],
  "region": "us-east-1",
  "debug": false
}
```

And run:

```bash
MC_CONFIG=~/mc.json monitoring--cloudwatch
```

You should be able to see the metrics at `locahost:8080`.

## Description

This tool surfaces AWS CloudWatch metrics as prometheus metrics.  It gets the
metric data at scrape time, rather than pulling and caching periodically.
Currently the naming of the metrics is not configurable, but that should be easy
to fix if it would help anyone.

An important detail to be aware of is that CloudWatch metrics are generally
unlike prometheus metrics (dimensions are less like labels and more like oddly
named directories.)  For the most part this is not a huge problem, but for
DynamoDB Indexes we have to append `_index` to the metric to prevent collisions
if you are scraping table level and index level metrics.

## Advanced Customization

The majority of the code for `monitoring--cloudwatch` is in [a
package](https://godoc.org/github.com/ZipRecruiter/monitoring--cloudwatch/pkg/exportcloudwatch)
so that less common requirements can be supported by a separate main package.

You should be able to trivially swap in other configuration styles (like YAML,
if that's what you prefer,) have prometheus listen at a different location.

If you do create an alternate version please make sure that you are using go
modules or some other pinning strategy.  I have ideas to improve this package
that will require breaking changes at some point; if you don't pin, you'll need
to fix your code when that happens.

I suggest that you look over [how we create the `*cloudwatch.CloudWatch`
client](https://github.com/ZipRecruiter/monitoring--cloudwatch/blob/master/deps.go)
and copy some of the patterns, since surfacing how the exporter is interacting
with the AWS API can be tricky but is worth the effort.

---

Hope this helps!

fREW Schmidt
