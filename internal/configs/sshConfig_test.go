package configs

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/ztrue/tracerr"
)

func TestNewSSHConfig(t *testing.T) {
	t.Run("Validate configs: correct one", func(t *testing.T) {
		sshCfg := NewSSHConfig()
		sshCfg.Username = "username"
		sshCfg.Password = "password"
		sshCfg.Host = "host"
		sshCfg.Port = 80
		sshCfg.Timeout = 5

		if err := sshCfg.Validate(); err != nil {
			t.Fatal(tracerr.Sprint(err))
		}
	})

	t.Run("loadFromEnv test", func(t *testing.T) {
		sshCfg := NewSSHConfig()
		sshCfg.Username = "username"
		sshCfg.Password = "password"
		sshCfg.Host = "host"
		sshCfg.Port = 80
		sshCfg.Timeout = 5
		err := os.Setenv("UPLOADER_SSH_USERNAME", "username")
		err = os.Setenv("UPLOADER_SSH_PASSWORD", "password")
		err = os.Setenv("UPLOADER_SSH_HOST", "host")
		err = os.Setenv("UPLOADER_SSH_PORT", "80")
		err = os.Setenv("UPLOADER_SSH_TIMEOUT", "5")
		if err != nil {
			t.Fatal(tracerr.Sprint(err))
		}

		sshCfg_2 := NewSSHConfig()
		viper.SetEnvPrefix("uploader")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		viper.AutomaticEnv() // read in environment variables that match
		if err := sshCfg_2.LoadFromEnv(); err != nil {
			t.Fatal(tracerr.Sprint(err))
		}

		if err := sshCfg_2.Validate(); err != nil {
			t.Fatal(tracerr.Sprint(err))
		}

		if !assert.Equal(t, sshCfg, sshCfg_2) {
			t.Fatal("sshCfg_2 should be equal to sshCfg")
		}
	})
}
