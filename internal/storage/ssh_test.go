package storage

import (
	"PackageManager/internal/configs"
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

func TestNewSshClient(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(tracerr.Sprint(err))
	}
	viper.SetEnvPrefix("uploader")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err = godotenv.Load(filepath.Join(home, ".env")); err != nil {
		t.Fatal(tracerr.Sprint(err))
	}

	sshCfg := configs.NewSSHConfig()
	if err = sshCfg.LoadFromEnv(); err != nil {
		t.Fatal(tracerr.Sprint(err))
	}

	if err = sshCfg.Validate(); err != nil {
		t.Fatal(tracerr.Sprint(err))
	}

	ctx := context.WithValue(context.Background(), "ssh-config", sshCfg)

	t.Run("NewSshClient", func(t *testing.T) {
		sshClient, err := NewSshClient(ctx)
		if err != nil {
			t.Fatal(tracerr.Sprint(err))
		}
		defer sshClient.Close()
	})

	//	Need mock
	/*t.Run("test push file and fetch (settings from env)", func(t *testing.T) {
		sshClient, err := NewSshClient(ctx)
		if err != nil {
			t.Fatal(tracerr.Sprint(err))
		}
		defer sshClient.Close()

		testFileName := "test"
		testFileData := []byte("First Step")
		err = os.Chdir("../")
		if err != nil {
			t.Fatal(tracerr.Sprint(err))
		}

		testReader := bytes.NewBuffer(testFileData)
		testZipPath := fmt.Sprintf("%s.zip", testFileName)

		err = createArchiveWithTestFile(testZipPath, testFileData)
		if err != nil {
			t.Fatal(tracerr.Sprint(err))
		}
		defer func() {
			os.Remove(testZipPath)
			err = sshClient.Remove(testZipPath)
			if err != nil {
				t.Fatal(tracerr.Sprint(err))
			}
		}()

		testZip, err := os.Open(testZipPath)
		if err != nil {
			t.Fatal(tracerr.Sprint(err))
		}

		err = sshClient.Upload(testZip, testZipPath)
		if err != nil {
			t.Fatal(tracerr.Sprint(err))
		}
		testZip.Close()

		testFileData = []byte("Second Step")
		err = createArchiveWithTestFile(testZipPath, testFileData)
		if err != nil {
			t.Fatal(tracerr.Sprint(err))
		}
		testZip, err = os.Open(testZipPath)
		if err != nil {
			t.Fatal(tracerr.Sprint(err))
		}
		err = sshClient.Update(testReader, testFileName)
		if err != nil {
			t.Fatal(tracerr.Sprint(err))
		}
		testZip.Close()

	})*/
}

func createArchiveWithTestFile(path string, data []byte) error {
	testZip, err := os.Create(path)
	if err != nil {
		return tracerr.Wrap(err)
	}
	w := zip.NewWriter(testZip)
	entry, err := w.Create("test")
	if err != nil {
		return tracerr.Wrap(err)
	}
	_, err = entry.Write(data)
	if err != nil {
		return tracerr.Wrap(err)
	}
	w.Close()
	testZip.Close()
	return nil
}
