module wind_portable

go 1.25.0

toolchain go1.25.6

require (
	github.com/huanfeng/wind_input v0.0.0
	github.com/rodrigocfd/windigo v0.2.5
	golang.org/x/sys v0.29.0
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/huanfeng/wind_input => ../wind_input
