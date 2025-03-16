package util

import (
	"time"

	"github.com/spf13/viper"
)

// Config is the application configuration
type Config struct {
	DBSource               string        `mapstructure:"DB_SOURCE"`
	AllowedOrigins         []string      `mapstructure:"ALLOWED_ORIGINS"`
	HTTPServerAddress      string        `mapstructure:"HTTP_SERVER_ADDRESS"`
	TokenSymmetricKey      string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration    time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration   time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	RedisURL               string        `mapstructure:"REDIS_URL"`
	MigrationURL           string        `mapstructure:"MIGRATION_URL"`
	EmailSenderName        string        `mapstructure:"EMAIL_SENDER_NAME"`
	EmailSenderAddress     string        `mapstructure:"EMAIL_SENDER_ADDRESS"`
	EmailSenderPassword    string        `mapstructure:"EMAIL_SENDER_PASSWORD"`
	APILimitBound          int64         `mapstructure:"API_LIMIT_BOUND"`
	APILimitDuration       time.Duration `mapstructure:"API_LIMIT_DURATION"`
	GoogleClientID         string        `mapstructure:"GOOGLE_OAUTH_CLIENT_ID"`
	GoogleClientSecret     string        `mapstructure:"GOOGLE_OAUTH_CLIENT_SECRET"`
	GoogleOAuthRedirectUrl string        `mapstructure:"GOOGLE_OAUTH_REDIRECT_URL"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.SetTypeByDefaultValue(true)

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
