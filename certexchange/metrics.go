package certexchange

import (
	"context"
	"errors"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var meter = otel.Meter("f3/certexchange")

var (
	attrStatus = attribute.Key("status")

	attrStatusSuccess = attribute.KeyValue{
		Key:   attrStatus,
		Value: attribute.StringValue("success"),
	}
	attrStatusError = attribute.KeyValue{
		Key:   attrStatus,
		Value: attribute.StringValue("error-other"),
	}
	attrStatusCanceled = attribute.KeyValue{
		Key:   attrStatus,
		Value: attribute.StringValue("error-canceled"),
	}
	attrStatusTimeout = attribute.KeyValue{
		Key:   attrStatus,
		Value: attribute.StringValue("error-timeout"),
	}
	attrStatusInternalError = attribute.KeyValue{
		Key:   attrStatus,
		Value: attribute.StringValue("error-internal"),
	}

	attrDialFailed = attribute.Key("dial-failed")

	attrWithPowerTable = attribute.Key("with-power-table")
)

func status(ctx context.Context, err error) attribute.KeyValue {
	if err == nil {
		return attrStatusSuccess
	}

	if os.IsTimeout(err) || errors.Is(err, os.ErrDeadlineExceeded) {
		return attrStatusTimeout
	}

	switch ctx.Err() {
	case nil:
		return attrStatusError
	case context.DeadlineExceeded:
		return attrStatusTimeout
	default:
		return attrStatusCanceled
	}

}

var metrics = struct {
	requestLatency     metric.Float64Histogram
	totalResponseTime  metric.Float64Histogram
	serveTime          metric.Float64Histogram
	certificatesServed metric.Int64Histogram
}{
	requestLatency: must(meter.Float64Histogram(
		"f3_certexchange_request_latency_s",
		metric.WithDescription("The outbound request latency."),
		metric.WithUnit("s"),
	)),
	totalResponseTime: must(meter.Float64Histogram(
		"f3_certexchange_total_response_time_s",
		metric.WithDescription("The total time for outbound requests."),
		metric.WithUnit("s"),
	)),
	serveTime: must(meter.Float64Histogram(
		"f3_certexchange_serve_time_s",
		metric.WithDescription("The time spent serving requests."),
		metric.WithUnit("s"),
	)),
	certificatesServed: must(meter.Int64Histogram(
		"f3_certexchange_certificates_served",
		metric.WithDescription("The number of certificates served (per request)."),
		metric.WithUnit("{certificate}"),
	)),
}
