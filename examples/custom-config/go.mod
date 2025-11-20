module example-custom-config

go 1.23

require (
	github.com/gin-gonic/gin v1.11.0
	github.com/epicsagas/korean-geocode v0.0.0
)

replace github.com/epicsagas/korean-geocode => ../..
