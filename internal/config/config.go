package config

type Config struct {
	Port string
}

func LoadConfig() (*Config, error) {
	// Carregar a configuração aqui (por exemplo, de um arquivo, variáveis de ambiente, etc.)
	return &Config{
		Port: "8080",
	}, nil
}
