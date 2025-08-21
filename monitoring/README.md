# ISX Pulse - Scratch Card License Monitoring

This directory contains comprehensive monitoring and observability configurations for the ISX Pulse scratch card license system.

## Overview

The monitoring system provides:
- **Real-time metrics** collection and visualization
- **Proactive alerting** for system issues
- **Comprehensive logging** with structured data
- **Performance tracking** and optimization insights
- **Security monitoring** and audit trails

## Directory Structure

```
monitoring/
├── README.md                    # This file
├── prometheus.yml               # Prometheus configuration
├── scratch-card-dashboard.json  # Grafana dashboard config
├── logging-standards.md         # Logging requirements and patterns
└── alerts/
    └── scratch-card.yml         # Prometheus alerting rules
```

## Components

### 1. Metrics Collection (Prometheus)

**Configuration**: `prometheus.yml`

**Key Metrics**:
- `scratch_card_activations_total` - Total card activation attempts by type
- `scratch_card_success_rate` - Success rate histogram by card type
- `apps_script_response_time_seconds` - Apps Script API response times
- `fingerprint_generation_duration_seconds` - Device fingerprint timings
- `batch_processing_duration_seconds` - Batch operation performance
- `active_pending_cards` - Current pending card count

**Scrape Jobs**:
- Main service metrics (15s interval)
- Health checks (30s interval)
- Scratch card specific metrics (5s interval)
- System resources via node-exporter

### 2. Visualization (Grafana)

**Dashboard**: `scratch-card-dashboard.json`

**Panels**:
- Activation rate overview by card type
- Success rate gauges
- Apps Script performance graphs
- Device fingerprint metrics
- Rate limiting and security events
- Batch processing throughput
- Error distribution pie chart
- System health status

**Import Instructions**:
1. Open Grafana UI
2. Go to Dashboards > Import
3. Upload `scratch-card-dashboard.json`
4. Configure Prometheus data source
5. Save and view dashboard

### 3. Alerting (Prometheus Alerts)

**Configuration**: `alerts/scratch-card.yml`

**Critical Alerts**:
- High failure rates (>10% warning, >25% critical)
- Apps Script connectivity loss
- Service downtime
- Critical pending card count (>5000)

**Warning Alerts**:
- Slow response times (>5s Apps Script, >3s validation)
- Rate limiting spikes
- High fingerprint mismatches
- Cache efficiency below 70%
- Security event spikes

### 4. Structured Logging

**Standards**: `logging-standards.md`

**Requirements**:
- JSON format for production
- Trace ID in all log entries
- No sensitive data logging
- Separate audit stream for security events
- Performance timing data inclusion

## Quick Start

### 1. Start Monitoring Stack

```bash
# Start Prometheus
docker run -d \
  --name prometheus \
  -p 9090:9090 \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus

# Start Grafana
docker run -d \
  --name grafana \
  -p 3000:3000 \
  grafana/grafana

# Start Alertmanager (optional)
docker run -d \
  --name alertmanager \
  -p 9093:9093 \
  prom/alertmanager
```

### 2. Configure ISX Service

Ensure your ISX License service exposes metrics at `/metrics` endpoint:

```go
// In your main.go or server setup
import "github.com/prometheus/client_golang/prometheus/promhttp"

// Add metrics endpoint
r.Handle("/metrics", promhttp.Handler())
```

### 3. Import Grafana Dashboard

1. Access Grafana at http://localhost:3000
2. Login (admin/admin by default)
3. Import `scratch-card-dashboard.json`
4. Set Prometheus data source to http://localhost:9090

### 4. Verify Metrics

Check that metrics are being collected:

```bash
# Check Prometheus targets
curl http://localhost:9090/api/v1/targets

# Check scratch card metrics
curl http://localhost:8080/metrics | grep scratch_card
```

## Key Performance Indicators (KPIs)

### 1. Availability Metrics
- **Service Uptime**: Target >99.9%
- **Apps Script Connectivity**: Target >99.5%
- **Google Sheets Connectivity**: Target >99%

### 2. Performance Metrics
- **Card Activation Time**: Target <2s (95th percentile)
- **Apps Script Response**: Target <3s (95th percentile)
- **Batch Processing**: Target <30s for 100 cards
- **Cache Hit Rate**: Target >80%

### 3. Quality Metrics
- **Activation Success Rate**: Target >95%
- **Fingerprint Match Rate**: Target >98%
- **Error Rate**: Target <1%

### 4. Security Metrics
- **Invalid Key Attempts**: Monitor for spikes
- **Rate Limiting Events**: Track pressure
- **Security Events**: Investigate all events

## Alerting Thresholds

### Critical (Immediate Response Required)
- Service down for >1 minute
- Apps Script connectivity lost for >1 minute
- Failure rate >25% for >2 minutes
- Pending cards >5000 for >2 minutes

### Warning (Monitor and Investigate)
- Failure rate >10% for >5 minutes
- Response time >5s for >5 minutes
- Rate limiting >0.05/sec for >2 minutes
- Cache hit rate <70% for >10 minutes

### Info (Track Trends)
- Performance degradation
- Unusual usage patterns
- Security events

## Troubleshooting

### Common Issues

1. **No Metrics Appearing**
   - Check service is running on port 8080
   - Verify `/metrics` endpoint is accessible
   - Check Prometheus targets status

2. **Dashboard Not Loading**
   - Verify Grafana data source configuration
   - Check Prometheus connectivity
   - Ensure dashboard JSON is valid

3. **Alerts Not Firing**
   - Verify Prometheus rules are loaded
   - Check alert rule syntax
   - Ensure Alertmanager is configured

### Debug Commands

```bash
# Check service metrics endpoint
curl http://localhost:8080/metrics

# Check Prometheus configuration
curl http://localhost:9090/api/v1/status/config

# Check alert rules
curl http://localhost:9090/api/v1/rules

# Check active alerts
curl http://localhost:9090/api/v1/alerts
```

## Advanced Configuration

### 1. Custom Metrics

Add custom business metrics:

```go
// In your application code
var customMetric = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "scratch_card_business_metric",
        Help: "Custom business logic counter",
    },
    []string{"card_type", "region"},
)

// Register the metric
prometheus.MustRegister(customMetric)

// Use the metric
customMetric.WithLabelValues("PREMIUM", "US").Inc()
```

### 2. Long-term Storage

Configure remote write for long-term storage:

```yaml
# In prometheus.yml
remote_write:
  - url: "https://your-tsdb-endpoint/api/v1/write"
    queue_config:
      max_samples_per_send: 10000
```

### 3. Multi-environment Setup

Use different configurations for dev/staging/prod:

```bash
# Development
prometheus --config.file=prometheus-dev.yml

# Staging
prometheus --config.file=prometheus-staging.yml

# Production
prometheus --config.file=prometheus-prod.yml
```

## Maintenance

### Regular Tasks

1. **Weekly**: Review alert noise and adjust thresholds
2. **Monthly**: Analyze performance trends and capacity
3. **Quarterly**: Update dashboards and metrics
4. **Annually**: Review retention policies and storage

### Performance Optimization

- Monitor Prometheus storage usage
- Adjust scrape intervals based on needs
- Use recording rules for expensive queries
- Implement metric filtering for high cardinality

## Security Considerations

- **Access Control**: Secure Prometheus and Grafana endpoints
- **Data Privacy**: Ensure no PII in metrics labels
- **Network Security**: Use TLS for metric collection
- **Audit Logging**: Monitor access to monitoring systems

## Support

For issues with monitoring setup:
1. Check logs in the service and monitoring components
2. Review configuration files for syntax errors
3. Verify network connectivity between components
4. Consult ISX Pulse documentation and support channels