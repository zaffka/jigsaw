package di

import (
	"time"

	"github.com/spf13/viper"
)

// ViperConfigReader adapts viper to ConfigReader interface.
type ViperConfigReader struct{}

func NewViperConfigReader() *ViperConfigReader {
	return &ViperConfigReader{}
}

func (v *ViperConfigReader) GetString(key string) string {
	return viper.GetString(key)
}

func (v *ViperConfigReader) GetDuration(key string) time.Duration {
	return viper.GetDuration(key)
}

func (v *ViperConfigReader) GetBool(key string) bool {
	return viper.GetBool(key)
}
