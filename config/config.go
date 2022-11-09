package config

import (
	"fmt"
	"math"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/SSSOC-CAN/laniakea/utils"
	bg "github.com/SSSOCPaulCote/blunderguard"
)

const (
	ErrFileNotFound = bg.Error("file not found")
)

type Info struct {
	URL         string `toml:"url"`         // the advertised URL for websocket connection Ex: wss://nostr.example.com
	Name        string `toml:"name"`        // name for your relay
	Description string `toml:"description"` // description for clients
	Pubkey      string `toml:"pubkey"`      // Relay admin contact pubkey
	Contact     string `toml:"contact"`     // Relay admin contact URI
}

type Logging struct {
	LogLevel      string `toml:"log_level"`         // (INFO|DEBUG|TRACE|ERROR)
	ConsoleOutput bool   `toml:"console_output"`    // Print logs to console
	LogFileDir    string `toml:"log_file_location"` // location of the log files. Default is AppData/Tandem | Application Support/Tandem or ~/.tandem
}

type Data struct {
	DataDir string `toml:"data_directory"` // Directory for the database. Defaults to AppData | Application Support | ~/.tandem
}

type Network struct {
	BindAddress  string `toml:"bind_address"`  // Bind to this network address
	Port         uint16 `toml:"port"`          // Port to listen on
	PingInterval uint16 `toml:"ping_interval"` // WebSockets ping interval in seconds, default: 300
}

type Limits struct {
	MsgPerSec        uint32 `toml:"message_per_second"`      // Limit of events that can be created per second
	MaxEventSize     uint32 `toml:"maximum_event_size"`      // Maximum size in bytes, an event can be. Max=2^32 - 1
	MaxWSMsgSize     uint32 `toml:"maximum_ws_message_size"` // Maximum size in bytes a websocket message can be. Max=2^32 - 1
	RejectFutureSecs uint64 `toml:"reject_future_seconds"`   // Reject events that have timestamps greater than this many seconds in the future.  Recommended to reject anything greater than 30 minutes from the current time, but the default is to allow any date.
}

type Authorization struct {
	PubkeyWhitelist []string `toml:"whitelist"` // List of authorized pubkeys for event publishing. If not set, all pubkeys are accepted
}

type Config struct {
	Relay    Info          // relevant relay information
	Logging  Logging       // relevant logging settings
	Database Data          // relevant data settings
	Network  Network       // relevant network settings
	Limits   Limits        // relevant limit settings
	Auth     Authorization // relevant auth settings
}

var (
	config_file_name                  = "tandem.toml"
	default_bind_address              = "0.0.0.0"
	default_port               uint16 = 5150
	default_ping_interval      uint16 = 300
	default_data_dir                  = utils.AppDataDir("tandem", false)
	default_log_level                 = "ERROR"
	default_log_dir                   = default_data_dir
	default_console_out               = true
	default_msg_per_sec        uint32 = 50000
	default_max_event_size     uint32 = math.MaxUint32
	default_max_ws_msg_size    uint32 = math.MaxUint32
	default_reject_future_secs uint64 = 1800
	default_config                    = func() Config {
		return Config{
			Logging:  Logging{LogLevel: default_log_level, LogFileDir: default_log_dir, ConsoleOutput: default_console_out},
			Database: Data{DataDir: default_data_dir},
			Network:  Network{BindAddress: default_bind_address, Port: default_port, PingInterval: default_ping_interval},
			Limits:   Limits{MsgPerSec: default_msg_per_sec, MaxEventSize: default_max_event_size, MaxWSMsgSize: default_max_ws_msg_size, RejectFutureSecs: default_reject_future_secs},
		}
	}
)

// Initialize the config either from a configuration file or using default values
func InitConfig(cfgDir string) (*Config, error) {
	cfgFile := path.Join(cfgDir, config_file_name)
	config := default_config()
	if utils.FileExists(cfgFile) {
		_, err := toml.DecodeFile(cfgFile, &config)
		if err != nil {
			return nil, err
		}
	} else {
		fmt.Println("Using default configuration...")
	}
	return &config, nil
}
