module gateway-knx

go 1.17

require (
	github.com/project-eria/eria-core v0.2.1
	github.com/rs/zerolog v1.26.1
	github.com/vapourismo/knx-go v0.0.0-20200220204125-dd963bbc67db
)

require (
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/project-eria/go-wot v0.1.4 // indirect
	golang.org/x/text v0.3.6 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)

replace github.com/project-eria/go-wot => ../go-wot

replace github.com/project-eria/eria-core => ../eria-core
