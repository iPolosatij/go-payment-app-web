package config

type Config struct {
	ServerPort   string
	APIGate2All  string
	APIRobokassa string
}

func Load() *Config {
	return &Config{
		ServerPort:   ":3000",
		APIGate2All:  "https://api.gate2all.com.br/v1",
		APIRobokassa: "https://auth.robokassa.ru/Merchant/Index.aspx",
	}
}
