package domain

import (
	"regexp"
	"strings"
)

// AddressType은 주소 유형을 나타냅니다
type AddressType string

const (
	AddressTypeRoad   AddressType = "ROAD"   // 도로명주소
	AddressTypeParcel AddressType = "PARCEL" // 지번주소
)

var (
	// 괄호 안 동/읍/면/리 정보 제거 패턴
	// 예: "남부순환로347길 52 (서초동)" → "남부순환로347길 52"
	dongParenthesesPattern = regexp.MustCompile(`\s*\([^)]*[동읍면리]\)`)

	// 도로명주소 패턴: 대로, 로, 길로 끝나는 경우
	// 예: "테헤란로", "남부순환로347길", "디지털로30길", "APEC로"
	roadNamePattern = regexp.MustCompile(`[가-힣A-Za-z\d]+(?:대로|로|길)`)

	// 지번주소 패턴: 동/읍/면/리 + 번지
	// 예: "서초동 1234-5", "삼평동 681"
	parcelPattern = regexp.MustCompile(`[가-힣]+(?:동|읍|면|리)\s+\d+(?:-\d+)?`)
)

// CleanAddress는 주소를 정제합니다
// 1. 괄호 안의 동/읍/면/리 정보 제거: (서초동), (삼평동) 등
// 2. 연속된 공백을 단일 공백으로 변환
// 3. 앞뒤 공백 제거
func CleanAddress(address string) string {
	// 1. 괄호 안 동/읍/면/리 제거
	cleaned := dongParenthesesPattern.ReplaceAllString(address, "")

	// 2. 연속된 공백을 단일 공백으로
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")

	// 3. 앞뒤 공백 제거
	cleaned = strings.TrimSpace(cleaned)

	return cleaned
}

// DetectAddressType은 주소 유형을 판별합니다
//
// 판별 기준 (신뢰도 순):
// 1. 도로명 키워드 ("대로", "로", "길") 포함 → ROAD
// 2. 지번 패턴 (동/읍/면/리 + 번지) → PARCEL
// 3. 기본값 → ROAD (최신 주소 체계이므로)
//
// 참고:
// - 도로명주소: 2014년부터 전면 시행된 새로운 주소 체계
// - 도로명은 너비에 따라 분류: 대로(8차로↑), 로(2-7차로), 길(기타)
// - 건물번호는 20m 간격으로 부여 (좌측 홀수, 우측 짝수)
// - "APEC로"는 한국 내 유일한 영문 도로명
func DetectAddressType(address string) AddressType {
	// 우선순위 1: 도로명 패턴 검사
	if roadNamePattern.MatchString(address) {
		return AddressTypeRoad
	}

	// 우선순위 2: 지번 패턴 검사
	if parcelPattern.MatchString(address) {
		return AddressTypeParcel
	}

	// 기본값: 도로명주소 (최신 주소 체계)
	return AddressTypeRoad
}

// ParseAddress는 주소를 정제하고 유형을 판별합니다
func ParseAddress(address string) (cleaned string, addrType AddressType) {
	cleaned = CleanAddress(address)
	addrType = DetectAddressType(cleaned)
	return
}
