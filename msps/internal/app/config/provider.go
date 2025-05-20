package config

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	wire.FieldsOf(new(*Config),
		"IsDebug",
		"HttpPort",
		"SwagHost",
		"DatabaseDSN",
	),
	InitConfig,
	GlobalConfig,
)
