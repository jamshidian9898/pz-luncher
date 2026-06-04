# Production Readiness Checklist

**Platform**: PZ Launcher Execution Kernel v1.0  
**Status**: Pre-Production Validation  

## SLOs (Service Level Objectives)

| SLO | Target | Measurement | Status |
|-----|--------|-------------|--------|
| **Availability** | ≥ 99% | (Total - Fatal Failures) / Total | ⏳ Pending Campaign |
| **Success Rate** | ≥ 95% | Successful / Total Executions | ⏳ Pending Campaign |
| **Drift Rate** | < 10% | Drift Detections / Total Comparisons | ⏳ Pending Campaign |
| **P99 Latency** | < 60s | 99th percentile execution time | ⏳ Pending Campaign |

**Reliability Score**: 0-100 (25 points per SLO met)  
**Production Threshold**: ≥ 80/100  

---

## Validation Checklist

### Phase 1: Chaos Testing (Quick Validation)
- [ ] Run `go run apps/chaos-cli/main.go`
- [ ] All 5 preset scenarios pass (≥80%)
- [ ] Determinism report shows ≥80% replay accuracy
- [ ] No critical errors in chaos runs

**Evidence**: `cache/chaos-results/*.json`

---

### Phase 2: Shadow Validation (Real vs Simulation)
- [ ] Run `go run apps/validation-cli/main.go -mode=live`
- [ ] Run `go run apps/validation-cli/main.go -mode=chaos`
- [ ] Run `go run apps/validation-cli/main.go -mode=shadow`
- [ ] Drift rate < 10% confirmed
- [ ] No outcome mismatches (critical drift)

**Evidence**: `cache/validation/drift-report-*.json`

---

### Phase 3: Extended Campaign (Long-Run Validation)
- [ ] Run `go run apps/campaign-cli/main.go -runs=100`
- [ ] All SLOs met after 100 sessions
- [ ] Run `go run apps/campaign-cli/main.go -infinite` (30min minimum)
- [ ] No memory leaks (stable memory usage)
- [ ] No retry explosions (bounded attempts)
- [ ] Rate limiting effective (no API bans)

**Evidence**: `cache/campaign/campaign-metrics.json`, `campaign-final-report.json`

---

### Phase 4: Failure Distribution Analysis
- [ ] Retryable vs Fatal failure ratio documented
- [ ] SteamCMD fallback usage ratio measured
- [ ] Provider success rates documented
- [ ] Top failure types identified

**Evidence**: `cache/campaign/campaign-metrics.json` → `failureDistribution`

---

## Pre-Production Gates

### Gate 1: Code Stability
- [ ] Platform interfaces frozen (v1.0)
- [ ] No critical bugs open
- [ ] Build passes (`go build ./...`)
- [ ] No lint errors

### Gate 2: Documentation
- [ ] Guarantees documented
- [ ] Architecture locked
- [ ] Plugin boundary defined
- [ ] Breaking change policy established

### Gate 3: Validation
- [ ] Chaos tests pass
- [ ] Shadow validation confirms drift < 10%
- [ ] Extended campaign (100+ sessions) passes
- [ ] Reliability Score ≥ 80/100

### Gate 4: Operational
- [ ] Metrics export configured
- [ ] Alert thresholds defined
- [ ] Runbook for common failures
- [ ] Rollback procedure documented

---

## SLO Violation Response

### Availability < 99%
```
1. Check provider health (Steam API status)
2. Verify rate limiting not exceeded
3. Check for fatal error spike
4. Escalate if provider-side issue
```

### Success Rate < 95%
```
1. Analyze failure distribution
2. Check for new failure modes
3. Verify retry logic functioning
4. Consider temporary provider issues
```

### Drift Rate ≥ 10%
```
1. Identify drift type (outcome/timing/attempts)
2. Compare live vs chaos parameters
3. Update chaos model if needed
4. Investigate if real-world behavior changed
```

### P99 Latency ≥ 60s
```
1. Check network conditions
2. Verify rate limiting configured
3. Consider CDN/server issues
4. Monitor for sustained degradation
```

---

## Metrics Export

### Key Metrics to Monitor

```yaml
# Reliability Metrics
campaign_executions_total
campaign_success_rate
campaign_drift_rate
campaign_availability

# Performance Metrics
campaign_duration_seconds
{quantile="0.99"}  # P99 latency

# Provider Metrics
provider_usage_total{provider="Steam"}
provider_success_rate{provider="Steam"}
provider_fallback_count{from="SteamAPI",to="SteamCMD"}

# Error Metrics
failures_total{type="retryable|fatal"}
failures_by_provider
```

### Dashboard Queries

```promql
# Availability
1 - (failures_fatal_total / campaign_executions_total)

# Success Rate
campaign_success_rate

# Drift Rate
campaign_drift_rate

# P99 Latency
histogram_quantile(0.99, campaign_duration_seconds_bucket)
```

---

## Production Deployment

### Checklist
- [ ] All validation phases complete
- [ ] Reliability Score ≥ 80/100
- [ ] SLOs met for 24+ hours in staging
- [ ] Metrics flowing to monitoring
- [ ] Alerts configured
- [ ] Runbook reviewed
- [ ] Rollback tested

### Deployment Order
1. Deploy to staging
2. Run 24-hour campaign
3. Verify SLOs in staging
4. Deploy to production (canary)
5. Monitor for 1 hour
6. Full production rollout
7. Continuous monitoring

---

## Post-Production

### Continuous Validation
```bash
# Run continuous campaign in background
go run apps/campaign-cli/main.go -infinite -interval=5m

# Weekly SLO review
# Alert on SLO violation
# Monthly reliability report
```

### SLO Review Cadence
- **Daily**: Check metrics dashboard
- **Weekly**: SLO compliance review
- **Monthly**: Reliability report + drift analysis
- **Quarterly**: Platform review (v1.x → v2.0 consideration)

---

## Sign-Off

| Role | Name | Date | Signature |
|------|------|------|-----------|
| Engineering Lead | | | |
| SRE/Operations | | | |
| QA/Validation | | | |
| Product Owner | | | |

**Production Ready**: ⏳ Pending Validation

**Target Go-Live**: After 100-session campaign passes all SLOs
