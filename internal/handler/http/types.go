package http

import (
	"context"

	"github.com/epicsagas/korean-geocode/pkg/router"
)

// Router는 geocoding 라우터 인터페이스입니다
type Router interface {
	Geocode(ctx context.Context, query string) (*router.GeoRouteResult, error)
	GetProviderStatus() map[string]string
}
