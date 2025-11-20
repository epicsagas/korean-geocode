---

# **GeoCoding 하이브리드 API 서버 기술 백서**

**(High-Availability Hybrid Geocoding System Technical White Paper)**

버전: 1.2.0
작성 기준: Go 1.23+ 기반
대상 API: Google Maps Platform, Kakao Local API, Naver Maps Geocoding API, V-World(국토교통부)

---

## **1\. 개요 (Executive Summary)**

본 문서는 위치 기반 서비스의 핵심인 '주소-좌표 변환(Geocoding)' 기능을 안정적이고 효율적으로 제공하기 위한 하이브리드 시스템 아키텍처를 정의한다.
Google Maps, Kakao Local, Naver Maps, vWorld API를 단일 엔드포인트로 통합하고, Go 언어의 동시성(Concurrency) 패턴과 장애 격리(Circuit Breaker) 기술을 적용하여 가용성 99.99%, 응답 속도 최적화, 비용 절감을 동시에 달성하는 것을 목표로 한다.

## **2\. 도입 배경 및 필요성**

### **2.1. 단일 벤더의 한계**

* **Google Maps:** 글로벌 데이터가 우수하고 신뢰성이 높으나, 비용 부담(월 $200 초과 시 유료)이 크며 국내 신규 도로명 주소 업데이트가 다소 늦음.
* **Kakao Local:** 국내 주소 정밀도가 높고 무료 할당량(일 30만 건)이 넉넉하나, 글로벌 주소 처리가 불가능하며 간헐적 장애 발생 가능성.
* **Naver Maps:** 국내 주소 정밀도가 우수하고 무료 할당량(일 10만 건), 영문 주소 지원(`language=eng`)이 가능하나 글로벌 커버리지는 제한적.
* **vWorld:** 무료 공공 데이터이나, 트래픽 제한이 있고 응답 속도가 상대적으로 느림.

### **2.2. 하이브리드 전략의 이점**

* **비용 최적화:** 무료 쿼터가 있는 로컬 API(vWorld/Kakao/Naver)를 우선 사용하고, 실패 시 유료 API(Google)로 전환(Failover).
* **고가용성(HA):** 특정 벤더 장애 시 자동으로 타 벤더로 우회하여 서비스 중단 방지.
* **데이터 커버리지 확대:** 국내 지번/도로명 주소와 해외 영문 주소를 모두 완벽하게 지원.

## **3\. 시스템 아키텍처**

### **3.1. 논리적 구성도**

시스템은 **Router(지능형 분배) → Controller(제어) → Provider(실제 요청)** 계층으로 구성된다.

Plaintext

\[Client Request\]  
      │  
      ▼  
\[Interface Layer\] (Validations & Normalization)  
      │  
      ▼  
\[Smart Router\] ─── (분기 판단: 한글/영문/패턴)
      │
      ├─ Case A (국내): vWorld(Primary) → Kakao(Secondary) → Naver(Tertiary) → Google(Fallback)
      └─ Case B (해외): Google(Primary) → Naver(language=eng) → Kakao(Fallback)
      │
      ▼
\[Circuit Breaker & Rate Limiter\] (장애 감지 및 쿼터 제한, SQLite/Redis)
      │
      ▼
\[Provider Pool\] (Go Routines with Context Timeout)
 ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
 │vWorld Adapter│  │ Kakao Adapter│  │ Naver Adapter│  │Google Adapter│
 └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘
        │                 │                 │                 │
        ▼                 ▼                 ▼                 ▼
   (External API)    (External API)    (External API)    (External API)

## **4\. 핵심 구현 전략 (Go Implementation)**

### **4.1. 인터페이스 기반의 유연한 설계**

특정 벤더에 종속되지 않도록 Geocoder 인터페이스를 정의하여 확장성을 보장한다.

Go

// Geocoder Interface: 모든 공급자는 이 인터페이스를 구현해야 함  
type Geocoder interface {  
    Name() string  
    Geocode(ctx context.Context, query string) (\*GeoResult, error)  
}

// 공통 결과 구조체 (좌표계는 WGS84로 통일)  
type GeoResult struct {  
    Provider  string  \`json:"provider"\`  
    Latitude  float64 \`json:"lat"\`  
    Longitude float64 \`json:"lng"\`  
    Address   string  \`json:"address"\`  
}

### **4.2. Context를 활용한 타임아웃 제어**

외부 API의 응답 지연이 전체 시스템의 병목이 되지 않도록 context.WithTimeout을 필수적으로 적용한다.

Go

func (g \*GoogleProvider) Geocode(ctx context.Context, query string) (\*GeoResult, error) {  
    // 요청 생성 시 부모 Context(Timeout 포함)를 주입  
    req, err := http.NewRequestWithContext(ctx, "GET", g.buildURL(query), nil)  
    if err \!= nil {  
        return nil, err  
    }  
      
    client := \&http.Client{}  
    resp, err := client.Do(req) // Context 만료 시 자동 취소됨  
    if err \!= nil {  
        return nil, err // Timeout or Network Error  
    }  
    defer resp.Body.Close()  
      
    // ... 결과 파싱 로직 ...  
}

### **4.3. Circuit Breaker (장애 격리)**

특정 API(예: vWorld)에서 연속적인 타임아웃이나 5xx 에러 발생 시, 즉시 해당 공급자를 일시 차단(Open State)하여 불필요한 대기를 제거한다.

* **라이브러리:** sony/gobreaker 등 활용.  
* **설정:** 최근 10회 요청 중 5회 이상 실패 시 60초간 요청 차단.

### **4.4. 좌표계(CRS) 정규화**

* **문제:** vWorld 등 일부 공공 API는 요청 파라미터 누락 시 GRS80이나 Bessel 좌표계를 반환할 수 있음.  
* **해결:** 모든 API 요청 시 EPSG:4326 (WGS84) 파라미터를 명시적으로 전달하고, 응답 수신 후 위경도 값이 유효 범위(Lat: \-90\~90, Lng: \-180\~180) 내인지 검증하는 미들웨어 적용.

## **5\. 운영 및 규정 준수 (Compliance & Operations)**

### **5.1. 쿼터 매니지먼트 (Quota Management)**

비용 통제를 위해 Redis 등을 활용한 실시간 사용량 카운팅 시스템을 도입한다.

* **Kakao:** 일일 30만 건 도달 시 → 자동으로 Google/vWorld로 라우팅 변경.  
* **Google:** 월 무료 크레딧($200) 소진 경고 알림 연동.

### **5.2. 데이터 저장 금지 (Terms of Service 준수)**

지도 API 사용 약관상 **Geocoding 결과의 영구 저장(DB 적재)은 엄격히 금지**된다.

* **정책:** 결과를 Database에 저장하지 않으며, 실시간 요청-응답 구조만 유지한다.  
* **최적화:** 성능 향상이 필요한 경우, HTTP 표준 헤더(Cache-Control)를 준수하는 범위 내에서 인메모리(In-Memory) 단기 캐싱만 제한적으로 허용한다.

## **6\. API 명세 및 배포 전략**

### **6.1. Request / Response 예시**

* **Endpoint:** GET /v1/geocode?q={address}  
* **Response (JSON):**  
  JSON  
  {  
    "data": {  
      "provider": "kakao",  
      "lat": 37.498095,  
      "lng": 127.027610,  
      "address": "서울 강남구 강남대로 396",  
      "crs": "WGS84"  
    },  
    "meta": {  
      "fallback\_history": \["google\_skipped\_cost", "kakao\_success"\]  
    }  
  }

### **6.2. 배포 및 보안**

* **API Key 관리:** 코드 하드코딩 금지. AWS Parameter Store 또는 Kubernetes Secrets를 통해 런타임에 환경변수(ENV)로 주입.  
* **IP 화이트리스트:** Google/Kakao 콘솔에서 서버의 Outbound IP만 허용하도록 설정.

## **7\. 결론**

본 시스템은 Go 언어의 강력한 동시성 모델을 기반으로 복수의 지도 API를 단일 인터페이스로 추상화했다. 이를 통해 \*\*'비용은 최소화(Kakao/vWorld 우선)'\*\*하면서도 \*\*'서비스 안정성은 극대화(Google Fallback)'\*\*하는 목표를 달성한다. 향후 네이버 지도 API나 OpenStreetMap 등을 추가하더라도 기존 비즈니스 로직의 변경 없이 Geocoder 인터페이스 구현체만 추가하면 되는 유연한 구조를 갖추었다.

---

