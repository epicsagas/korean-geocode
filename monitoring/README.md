# Korean Geocode - Monitoring Setup

Prometheus + Grafana 기반 실시간 쿼터 모니터링 시스템

## 📊 Overview

이 모니터링 스택은 다음을 제공합니다:

- **실시간 메트릭 수집**: Prometheus가 `/metrics` 엔드포인트에서 10분마다 수집
- **시각화 대시보드**: Grafana에서 쿼터 사용률, 트렌드, 알림 확인
- **자동 알림**: 쿼터 사용률 80%, 95% 임계값 알림
- **영구 저장소**: Docker 볼륨으로 데이터 보존

## 🚀 Quick Start

### 1. API 서버 실행

```bash
# API 서버 실행 (메트릭 노출)
make run

# 메트릭 확인
curl http://localhost:8080/metrics
```

### 2. 모니터링 스택 실행

```bash
cd monitoring

# Prometheus + Grafana 실행
docker-compose up -d

# 로그 확인
docker-compose logs -f
```

### 3. 대시보드 접속

- **Prometheus UI**: http://localhost:9090
- **Grafana UI**: http://localhost:3000
  - Username: `admin`
  - Password: `admin`

## 📈 Exposed Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `geocode_quota_total` | Gauge | Provider별 총 일일 쿼터 |
| `geocode_usage_total` | Gauge | Provider별 현재 사용량 |
| `geocode_quota_remaining` | Gauge | Provider별 남은 쿼터 |
| `geocode_quota_utilization_percent` | Gauge | Provider별 사용률 (0-100%) |

각 메트릭은 `provider` 레이블로 구분됩니다: `google`, `kakao`, `vworld`, `naver`

### Example Metrics Output

```
# HELP geocode_quota_total Total daily quota for each provider
# TYPE geocode_quota_total gauge
geocode_quota_total{provider="kakao"} 300000
geocode_quota_total{provider="vworld"} 1000000

# HELP geocode_usage_total Current usage count for each provider
# TYPE geocode_usage_total gauge
geocode_usage_total{provider="kakao"} 15234
geocode_usage_total{provider="vworld"} 8765

# HELP geocode_quota_remaining Remaining quota for each provider
# TYPE geocode_quota_remaining gauge
geocode_quota_remaining{provider="kakao"} 284766
geocode_quota_remaining{provider="vworld"} 991235

# HELP geocode_quota_utilization_percent Quota utilization percentage
# TYPE geocode_quota_utilization_percent gauge
geocode_quota_utilization_percent{provider="kakao"} 5.08
geocode_quota_utilization_percent{provider="vworld"} 0.88
```

## 🔔 Alert Rules

### HighQuotaUsage (Warning)
- **조건**: 쿼터 사용률 > 80%
- **지속 시간**: 5분
- **심각도**: warning

### CriticalQuotaUsage (Critical)
- **조건**: 쿼터 사용률 > 95%
- **지속 시간**: 2분
- **심각도**: critical

### LowQuotaRemaining (Warning)
- **조건**: 남은 쿼터 < 1000
- **지속 시간**: 5분
- **심각도**: warning

## 📋 Grafana Dashboard

대시보드는 자동으로 프로비저닝됩니다:

### Dashboard Panels

1. **Quota Utilization Gauges** (상단)
   - Google, Kakao, vWorld, Naver 각각의 사용률 게이지
   - 색상 임계값: 녹색(0-70%), 노란색(70-80%), 주황(80-95%), 빨강(95-100%)

2. **Usage Trends** (중단 좌측)
   - 시간별 사용량 트렌드 그래프
   - Provider별 색상 구분

3. **Remaining Quota** (중단 우측)
   - 남은 쿼터 추세 그래프
   - 최소값/최신값 표시

4. **Provider Status Overview** (하단)
   - 모든 Provider 상태를 테이블로 표시
   - 총 쿼터, 현재 사용량, 남은 쿼터, 사용률 한눈에 확인

## 🔧 Configuration

### Prometheus 설정 변경

`prometheus.yml`:
```yaml
scrape_configs:
  - job_name: 'korean-geocode'
    static_configs:
      - targets: ['host.docker.internal:8080']
    scrape_interval: 10s  # 수집 간격 조정
```

### Alert 임계값 변경

`alert_rules.yml`:
```yaml
- alert: HighQuotaUsage
  expr: geocode_quota_utilization_percent > 80  # 임계값 조정
  for: 5m  # 지속 시간 조정
```

### Grafana 비밀번호 변경

`docker-compose.yml`:
```yaml
environment:
  - GF_SECURITY_ADMIN_USER=admin
  - GF_SECURITY_ADMIN_PASSWORD=your-secure-password
```

## 🛠️ Troubleshooting

### API 메트릭이 수집되지 않을 때

1. API 서버 실행 확인:
   ```bash
   curl http://localhost:8080/health
   ```

2. 메트릭 엔드포인트 확인:
   ```bash
   curl http://localhost:8080/metrics
   ```

3. Prometheus 타겟 상태 확인:
   - http://localhost:9090/targets

### Docker 호스트 연결 문제

**Linux 환경**에서 `host.docker.internal`이 작동하지 않으면:

```yaml
# prometheus.yml 수정
- targets: ['172.17.0.1:8080']  # Docker bridge IP 사용
```

또는:

```yaml
# docker-compose.yml에 network_mode 추가
services:
  prometheus:
    network_mode: "host"
```

### Grafana 대시보드가 표시되지 않을 때

1. Datasource 연결 확인:
   - Grafana → Configuration → Data Sources → Prometheus
   - "Save & Test" 클릭하여 연결 테스트

2. 대시보드 수동 import:
   - Grafana → Dashboards → Import
   - `dashboards/geocode-quota-dashboard.json` 업로드

## 📁 File Structure

```
monitoring/
├── README.md                          # 이 파일
├── docker-compose.yml                 # Prometheus + Grafana 오케스트레이션
├── prometheus.yml                     # Prometheus 설정
├── alert_rules.yml                    # 알림 규칙
├── datasources.yml                    # Grafana 데이터소스 자동 설정
└── dashboards/
    ├── dashboard-provisioning.yml     # 대시보드 자동 프로비저닝 설정
    └── geocode-quota-dashboard.json   # 쿼터 모니터링 대시보드
```

## 🔄 Maintenance

### 데이터 백업

```bash
# Docker 볼륨 백업
docker run --rm -v monitoring_prometheus-data:/data \
  -v $(pwd)/backup:/backup alpine \
  tar czf /backup/prometheus-backup.tar.gz -C /data .

docker run --rm -v monitoring_grafana-data:/data \
  -v $(pwd)/backup:/backup alpine \
  tar czf /backup/grafana-backup.tar.gz -C /data .
```

### 데이터 복원

```bash
# Docker 볼륨 복원
docker run --rm -v monitoring_prometheus-data:/data \
  -v $(pwd)/backup:/backup alpine \
  tar xzf /backup/prometheus-backup.tar.gz -C /data

docker run --rm -v monitoring_grafana-data:/data \
  -v $(pwd)/backup:/backup alpine \
  tar xzf /backup/grafana-backup.tar.gz -C /data
```

### 설정 리로드

```bash
# Prometheus 설정 리로드 (재시작 없이)
curl -X POST http://localhost:9090/-/reload

# Grafana 재시작
docker-compose restart grafana
```

## 🌐 Production Deployment

프로덕션 환경에서는 다음을 고려하세요:

1. **보안**:
   - Grafana 비밀번호 변경
   - 외부 접근 제한 (리버스 프록시, 방화벽)
   - TLS/HTTPS 설정

2. **영구 저장소**:
   - 외부 볼륨 마운트 (NFS, EBS 등)
   - 정기 백업 자동화

3. **알림 통합**:
   - Alertmanager 추가 (Slack, Email, PagerDuty 등)
   - `alert_rules.yml`에 알림 채널 설정

4. **고가용성**:
   - Prometheus 클러스터링
   - Grafana 클러스터링 (PostgreSQL 백엔드)

## 📚 Additional Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [PromQL Query Examples](https://prometheus.io/docs/prometheus/latest/querying/examples/)
- [Grafana Dashboard Best Practices](https://grafana.com/docs/grafana/latest/best-practices/)
