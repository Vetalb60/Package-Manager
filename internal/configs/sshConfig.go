package configs

import (
	"strings"

	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

type SSHConfig struct {
	SSHConfig_ `mapstructure:"ssh"`
}

type SSHConfig_ struct {
	Username        string   `mapstructure:"username"`
	Password        string   `mapstructure:"password"`
	Host            string   `mapstructure:"host"`
	Port            int      `mapstructure:"port"`
	Timeout         int64    `mapstructure:"timeout"`
	PrivateKeyFile  string   `mapstructure:"private-key-file"`
	SshKeyExchanges []string `mapstructure:"ssh-key-exchanges"`
	SshStoragePath  string   `mapstructure:"ssh-storage-path"`
}

func NewSSHConfig() *SSHConfig {
	return &SSHConfig{}
}

func (s *SSHConfig) LoadFromEnv() error {
	s.Password = viper.GetString("ssh.password")
	s.Host = viper.GetString("ssh.host")
	s.Port = viper.GetInt("ssh.port")
	s.Timeout = viper.GetInt64("ssh.timeout")
	s.Username = viper.GetString("ssh.username")
	s.PrivateKeyFile = viper.GetString("ssh.private.file")
	s.SshStoragePath = viper.GetString("ssh.storage.path")
	exchanges := strings.Split(viper.GetString("key.exchanges"), ";")
	if !(len(exchanges) == 1 && exchanges[0] == "") {
		for _, ex := range exchanges {
			s.SshKeyExchanges = append(s.SshKeyExchanges, ex)
		}
	}
	return nil
}

func (s *SSHConfig) Validate() error {
	if s.Username == "" {
		return tracerr.New("username is required")
	}
	if strings.ContainsAny(s.Host, "[!@#$^()=?/\\ ,:;]") {
		return tracerr.New("host contains invalid characters")
	}
	if !(s.Port > 1 && s.Port < 65536) {
		return tracerr.New("port must be in range from 1 to 65536")
	}
	if s.Timeout <= 0 {
		return tracerr.New("timeout must be greater than zero")
	}

	return nil
}
