package main

import (
	"context"
	"log"
	"time"

	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/smithy-go/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

var awsRequestSeconds = prometheus.NewSummaryVec(prometheus.SummaryOpts{
	Name: "monitoring_cloudwatch_aws_request_seconds",
	Help: "Duration of requests we've made to the AWS service",
}, []string{"service", "call"})

var awsErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "monitoring_cloudwatch_aws_errors_total",
	Help: "Count of errors we've gotten from the AWS service",
}, []string{"service", "call"})

func init() {
	prometheus.MustRegister(awsRequestSeconds, awsErrorsTotal)
}

func initDependencies(ctx context.Context, config configuration) (*cloudwatch.Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(config.Region))
	if err != nil {
		return nil, err
	}

	// XXX recieve config as argument
	// awssession.ApplyUserAgent(sess, fmt.Sprintf("ZipRecruiter (monitoring/cloudwatch; %s; security@ziprecruiter.com)", Version))

	if config.Debug {
		cfg.APIOptions = append(cfg.APIOptions, func(stack *middleware.Stack) error {
			return stack.Initialize.Add(middleware.InitializeMiddlewareFunc("DebugLog", func(
				ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler,
			) (middleware.InitializeOutput, middleware.Metadata, error) {
				log.Printf("Request: %s/%s, Payload: %+v\n", awsmiddleware.GetServiceID(ctx), awsmiddleware.GetOperationName(ctx), in.Parameters)
				return next.HandleInitialize(ctx, in)
			}), middleware.Before)
		})
	}

	// Added after the retry middleware so that each attempt is observed, as
	// with the per-attempt handlers in v1 of the SDK.
	cfg.APIOptions = append(cfg.APIOptions, func(stack *middleware.Stack) error {
		return stack.Finalize.Add(middleware.FinalizeMiddlewareFunc("Counters", func(
			ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler,
		) (middleware.FinalizeOutput, middleware.Metadata, error) {
			service := awsmiddleware.GetServiceID(ctx)
			call := awsmiddleware.GetOperationName(ctx)
			start := time.Now()

			out, md, err := next.HandleFinalize(ctx, in)

			awsRequestSeconds.WithLabelValues(service, call).Observe(time.Since(start).Seconds())
			if err != nil {
				awsErrorsTotal.WithLabelValues(service, call).Inc()
			}

			return out, md, err
		}), middleware.After)
	})

	return cloudwatch.NewFromConfig(cfg), nil
}
