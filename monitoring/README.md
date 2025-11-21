# Korean Geocode - Monitoring Setup

실시간 쿼터 모니터링을 위한 두 가지 스택 제공: **Prometheus + Grafana** 또는 **ELK Stack**

## 📊 Overview

선택 가능한 모니터링 스택:

### Option 1: Prometheus + Grafana (권장)
- **실시간 메트릭 수집**: Prometheus가 `/metrics` 엔드포인트에서 10분마다 수집
- **시각화 대시보드**: Grafana에서 쿼터 사용률, 트렌드, 알림 확인
- **자동 알림**: 쿼터 사용률 80%, 95% 임계값 알림
- **경량 & 빠름**: 메트릭 전문 시계열 데이터베이스

### Option 2: ELK Stack (Elasticsearch + Logstash + Kibana)
- **로그 & 메트릭 통합**: Logstash가 메트릭을 10분마다 수집하여 Elasticsearch에 저장
- **강력한 검색**: Elasticsearch 기반 전문 검색 및 집계
- **유연한 시각화**: Kibana 대시보드로 커스텀 시각화
- **장기 보관**: 인덱스 기반 데이터 보존 및 관리

### Option 3: 둘 다 사용
- 두 스택을 동시에 실행하여 각각의 장점 활용
- Prometheus: 실시간 메트릭 & 알림
- ELK: 장기 보관 & 검색

## 🚀 Quick Start

### 1. API 서버 실행

```bash
# API 서버 실행 (메트릭 노출)
make run

# 메트릭 확인
curl http://localhost:8080/metrics
```

### 2. 모니터링 스택 선택 및 실행

#### Option A: Prometheus + Grafana만 실행

```bash
cd monitoring

# Prometheus + Grafana 실행
docker-compose -f docker-compose.all.yml --profile prometheus up -d

# 로그 확인
docker-compose -f docker-compose.all.yml logs -f prometheus grafana
```

**접속 정보:**
- Prometheus UI: http://localhost:9090
- Grafana UI: http://localhost:3000 (admin/admin)

#### Option B: ELK Stack만 실행

```bash
cd monitoring

# ELK Stack 실행
docker-compose -f docker-compose.all.yml --profile elk up -d

# 로그 확인
docker-compose -f docker-compose.all.yml logs -f elasticsearch logstash kibana
```

**접속 정보:**
- Elasticsearch: http://localhost:9200
- Kibana UI: http://localhost:5601

#### Option C: 두 스택 모두 실행

```bash
cd monitoring

# 모든 스택 실행
docker-compose -f docker-compose.all.yml --profile all up -d

# 또는
docker-compose -f docker-compose.all.yml up -d

# 로그 확인
docker-compose -f docker-compose.all.yml logs -f
```

**접속 정보:**
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin)
- Elasticsearch: http://localhost:9200
- Kibana: http://localhost:5601

### 3. 개별 스택 실행 (선택사항)

각 스택을 별도로 실행하려면:

```bash
cd monitoring

# Prometheus + Grafana만
cd prometheus
docker-compose up -d

# 또는 ELK Stack만
cd elk
docker-compose up -d
```

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

## 📊 Prometheus + Grafana Stack

### Alert Rules

#### HighQuotaUsage (Warning)
- **조건**: 쿼터 사용률 > 80%
- **지속 시간**: 5분
- **심각도**: warning

#### CriticalQuotaUsage (Critical)
- **조건**: 쿼터 사용률 > 95%
- **지속 시간**: 2분
- **심각도**: critical

#### LowQuotaRemaining (Warning)
- **조건**: 남은 쿼터 < 1000
- **지속 시간**: 5분
- **심각도**: warning

### Grafana Dashboard

대시보드는 자동으로 프로비저닝됩니다:

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

## 🔍 ELK Stack

### Logstash 설정

Logstash는 두 가지 방식으로 데이터를 수집합니다:

1. **HTTP Poller**: 10분마다 `/metrics` 엔드포인트를 폴링하여 Prometheus 메트릭 수집
2. **HTTP Input**: API에서 직접 JSON 로그를 5044 포트로 전송 (향후 확장)

### Elasticsearch 인덱스

- `geocode-metrics-YYYY.MM.DD`: 메트릭 데이터 (일별 인덱스)
- `geocode-logs-YYYY.MM.DD`: 로그 데이터 (향후 확장)

### Kibana 대시보드 설정

#### 자동 방식 (권장)

Kibana 대시보드를 자동으로 import:

```bash
# Kibana가 실행 중인지 확인
curl http://localhost:5601/api/status

# 대시보드 import (Kibana 시작 후 1-2분 대기)
curl -X POST "http://localhost:5601/api/saved_objects/_import?overwrite=true" \
  -H "kbn-xsrf: true" \
  --form file=@elk/kibana/dashboard.ndjson
```

대시보드 접속: http://localhost:5601/app/dashboards

#### 수동 방식

1. Kibana 접속: http://localhost:5601
2. Management → Stack Management → Index Patterns
3. Create index pattern: `geocode-metrics-*`
4. Time field: `@timestamp`
5. Management → Stack Management → Saved Objects
6. Import → `elk/kibana/dashboard.ndjson` 업로드
7. Dashboard 탭에서 "Korean Geocode - Quota Monitoring" 확인

#### 대시보드 패널

1. **Quota Total Gauge**: Provider별 총 쿼터 (게이지 차트)
2. **Usage Trends**: 시간별 사용량 추이 (라인 차트)
3. **Remaining Quota**: 남은 쿼터 추이 (영역 차트)
4. **Quota Utilization**: Provider별 사용률 (가로 막대 차트)
5. **Provider Status Table**: 전체 메트릭 테이블

## 🔧 Configuration

### Prometheus 스크랩 간격 변경

`prometheus/prometheus.yml`:
```yaml
global:
  scrape_interval: 10m  # 10분마다 수집
  evaluation_interval: 10m  # 10분마다 규칙 평가
```

### Logstash 수집 간격 변경

`elk/logstash/pipeline/metrics.conf`:
```
http_poller {
  schedule => { every => "10m" }  # 10분마다 폴링
}
```

### Alert 임계값 변경

`prometheus/alert_rules.yml`:
```yaml
- alert: HighQuotaUsage
  expr: geocode_quota_utilization_percent > 80  # 임계값 조정
  for: 5m  # 지속 시간 조정
```

### Grafana 비밀번호 변경

`docker-compose.all.yml`:
```yaml
grafana:
  environment:
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

4. Elasticsearch 데이터 확인:
   ```bash
   curl http://localhost:9200/geocode-metrics-*/_search?pretty
   ```

### Docker 호스트 연결 문제

**Linux 환경**에서 `host.docker.internal`이 작동하지 않으면:

```yaml
# prometheus/prometheus.yml 또는 elk/logstash/pipeline/metrics.conf 수정
- targets: ['172.17.0.1:8080']  # Docker bridge IP 사용
```

또는:

```yaml
# docker-compose에 network_mode 추가
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
   - `prometheus/dashboards/geocode-quota-dashboard.json` 업로드

### ELK Stack 헬스 체크

```bash
# Elasticsearch 상태 확인
curl http://localhost:9200/_cluster/health?pretty

# Logstash 상태 확인
curl http://localhost:9600/_node/stats?pretty

# Kibana 상태 확인
curl http://localhost:5601/api/status
```

## 📁 File Structure

```
monitoring/
├── README.md                          # 이 파일
├── docker-compose.all.yml             # 통합 실행 (profiles로 선택)
│
├── prometheus/                        # Prometheus + Grafana Stack
│   ├── docker-compose.yml
│   ├── prometheus.yml                 # Prometheus 설정
│   ├── alert_rules.yml                # 알림 규칙
│   ├── datasources.yml                # Grafana 데이터소스 자동 설정
│   └── dashboards/
│       ├── dashboard-provisioning.yml
│       └── geocode-quota-dashboard.json
│
└── elk/                               # ELK Stack
    ├── docker-compose.yml
    ├── kibana/
    │   └── dashboard.ndjson           # Kibana 대시보드 (import용)
    └── logstash/
        ├── config/
        │   └── logstash.yml           # Logstash 기본 설정
        └── pipeline/
            └── metrics.conf           # 메트릭 수집 파이프라인
```

## 🔄 Maintenance

### 데이터 백업

```bash
# Prometheus 백업
docker run --rm -v monitoring_prometheus-data:/data \
  -v $(pwd)/backup:/backup alpine \
  tar czf /backup/prometheus-backup.tar.gz -C /data .

# Grafana 백업
docker run --rm -v monitoring_grafana-data:/data \
  -v $(pwd)/backup:/backup alpine \
  tar czf /backup/grafana-backup.tar.gz -C /data .

# Elasticsearch 백업
docker run --rm -v monitoring_elasticsearch-data:/data \
  -v $(pwd)/backup:/backup alpine \
  tar czf /backup/elasticsearch-backup.tar.gz -C /data .
```

### 데이터 복원

```bash
# Prometheus 복원
docker run --rm -v monitoring_prometheus-data:/data \
  -v $(pwd)/backup:/backup alpine \
  tar xzf /backup/prometheus-backup.tar.gz -C /data

# Grafana 복원
docker run --rm -v monitoring_grafana-data:/data \
  -v $(pwd)/backup:/backup alpine \
  tar xzf /backup/grafana-backup.tar.gz -C /data

# Elasticsearch 복원
docker run --rm -v monitoring_elasticsearch-data:/data \
  -v $(pwd)/backup:/backup alpine \
  tar xzf /backup/elasticsearch-backup.tar.gz -C /data
```

### 설정 리로드

```bash
# Prometheus 설정 리로드 (재시작 없이)
curl -X POST http://localhost:9090/-/reload

# Grafana 재시작
docker-compose -f docker-compose.all.yml restart grafana

# Logstash 재시작
docker-compose -f docker-compose.all.yml restart logstash
```

### 스택 중지 및 제거

```bash
# 특정 프로파일만 중지
docker-compose -f docker-compose.all.yml --profile prometheus down
docker-compose -f docker-compose.all.yml --profile elk down

# 모든 스택 중지 (데이터 보존)
docker-compose -f docker-compose.all.yml down

# 모든 스택 중지 및 데이터 삭제
docker-compose -f docker-compose.all.yml down -v
```

## 🌐 Production Deployment

프로덕션 환경에서는 다음을 고려하세요:

1. **보안**:
   - Grafana/Kibana 비밀번호 변경
   - 외부 접근 제한 (리버스 프록시, 방화벽)
   - TLS/HTTPS 설정
   - Elasticsearch 보안 활성화 (xpack.security.enabled=true)

2. **영구 저장소**:
   - 외부 볼륨 마운트 (NFS, EBS 등)
   - 정기 백업 자동화
   - Elasticsearch 스냅샷 저장소 설정

3. **알림 통합**:
   - Alertmanager 추가 (Slack, Email, PagerDuty 등)
   - Elasticsearch Watcher 또는 Kibana Alerting 설정

4. **고가용성**:
   - Prometheus 클러스터링
   - Grafana 클러스터링 (PostgreSQL 백엔드)
   - Elasticsearch 클러스터 구성 (3+ 노드)

5. **리소스 튜닝**:
   - Elasticsearch JVM heap 크기 조정 (ES_JAVA_OPTS)
   - Logstash worker 개수 조정
   - Prometheus retention 설정

## 💡 Stack 선택 가이드

### Prometheus + Grafana를 선택하는 경우:
- ✅ 실시간 메트릭 모니터링이 주 목적
- ✅ 빠른 쿼리와 알림이 중요
- ✅ 리소스 사용량을 최소화하고 싶을 때
- ✅ 메트릭 기반 시계열 분석

### ELK Stack을 선택하는 경우:
- ✅ 로그와 메트릭을 통합 관리하고 싶을 때
- ✅ 강력한 검색 및 집계 기능이 필요
- ✅ 장기 데이터 보관 및 분석
- ✅ 이미 ELK 인프라가 구축되어 있을 때

### 두 스택 모두 사용하는 경우:
- ✅ Prometheus: 실시간 알림 및 빠른 대시보드
- ✅ ELK: 장기 보관 및 상세 분석
- ⚠️ 리소스 사용량이 증가함 (최소 2GB 메모리 권장)

## 📚 Additional Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Elasticsearch Documentation](https://www.elastic.co/guide/en/elasticsearch/reference/current/index.html)
- [Logstash Documentation](https://www.elastic.co/guide/en/logstash/current/index.html)
- [Kibana Documentation](https://www.elastic.co/guide/en/kibana/current/index.html)
- [PromQL Query Examples](https://prometheus.io/docs/prometheus/latest/querying/examples/)
- [Grafana Dashboard Best Practices](https://grafana.com/docs/grafana/latest/best-practices/)
