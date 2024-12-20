// Package conf implements the config parser for required data in prepper. Data
// is formatted as a JSON configuration document and is loaded once at startup.
package conf

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/go-playground/validator"
)

const DefaultHelpText = "For queries or assistance, please do not hesitate to contact your system administrator"

// Config represents the config file loaded from somewhere on disk at startup.
// It is de-serialized from JSON by encoding/json.
type Config struct {
	*validator.Validate

	ListenAddr     string   `validate:"ip_addr|hostname" json:"address"`
	ListenPort     uint16   `json:"port"`
	TrustedProxies []string `validate:"dive,ip_addr" json:"proxies"`

	HelpText string `json:"help_text"`

	DebugMode bool `json:"debug"`

	Database Database     `json:"database"`
	ISAMS    *ISAMSConfig `json:"isams"`

	TimetableLayout *TimetableLayout `json:"timetable_layout"`
}

// NewConfig parses a JSON config file from the file at path.
func NewConfig(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}

	c := Config{Validate: validator.New()}
	if err := json.Unmarshal([]byte(b), &c); err != nil {
		return c, fmt.Errorf("parse config: %w", err)
	}

	// Allow for nil period to be used.
	if c.TimetableLayout == nil {
		c.TimetableLayout = &TimetableLayout{nil}
	}

	if err := c.Struct(c); err != nil {
		return c, fmt.Errorf("validate config: %w", err)
	}

	return c, nil
}

func (c Config) FullAddr() string {
	return fmt.Sprintf("%s:%d", c.ListenAddr, c.ListenPort)
}

// HasISAMS returns true if ISAMS is configured in the config file, enabling
// ISAMS features.
func (c Config) HasISAMS() bool {
	return c.ISAMS != nil
}

// Database is a sub object contained within config which contains database
// credentials and other important configuration values.
type Database struct {
	Hostname string `validate:"ip_addr|hostname" json:"hostname"`
	Database string `json:"database"`
	Port     uint16 `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// FullAddr returns the full formatted address which the GORM constructor will
// accept.
func (d Database) FullAddr() string {
	return fmt.Sprint(d.Hostname, ":", d.Port)
}

type ISAMSConfig struct {
	Domain string `validate:"hostname" json:"domain"`
	APIKey string `json:"api_key"`
}
