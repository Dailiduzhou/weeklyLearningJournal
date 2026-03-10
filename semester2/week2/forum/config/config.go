package config

type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	MongoDB MongoDBConfig `mapstructure:"mongodb"`
	JWT     JWTConfig     `mapstructure:"jwt"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type MongoDBConfig struct {
	URI      string `mapstructure:"uri"`
	Database string `mapstructure:"database"`
}

type JWTConfig struct {
	Secret    string `mapstructure:"secret"`
	ExpiresIn string `mapstructure:"expires_in"`
	Issuer    string `mapstructure:"issuer"`
}
