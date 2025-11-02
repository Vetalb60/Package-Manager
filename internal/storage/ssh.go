package storage

import (
	"RemoteUploader/internal/configs"
	"RemoteUploader/internal/utils"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mholt/archives"
	"github.com/pkg/sftp"
	"github.com/ztrue/tracerr"
	"golang.org/x/crypto/ssh"
)

type SshClient struct {
	sshConfig *configs.SSHConfig
	clientCfg *ssh.ClientConfig
	session   *sftp.Client
	conn      *ssh.Client
}

const _max_packet_size = 1 << 15
const _max_pub_key_size = 1 << 15

func NewSshClient(ctx context.Context) (*SshClient, error) {
	var err error

	sshConfig, ok := ctx.Value("ssh-config").(*configs.SSHConfig)
	if !ok {
		return nil, tracerr.New("SSH client config not found in context")
	}

	auth := ssh.Password(sshConfig.Password)
	if sshConfig.PrivateKeyFile != "" {

		stat, err := os.Stat(sshConfig.PrivateKeyFile)
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
		if stat.Size() > _max_pub_key_size {
			return nil, tracerr.New("private key file too large")
		}

		key, err := os.ReadFile(sshConfig.PrivateKeyFile)
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, tracerr.New(fmt.Sprintf("ssh parse private key: %s", err.Error()))
		}
		auth = ssh.PublicKeys(signer)
	}

	sshClient := &SshClient{
		sshConfig: sshConfig,
		clientCfg: &ssh.ClientConfig{
			User:            sshConfig.Username,
			Auth:            []ssh.AuthMethod{auth},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         time.Duration(sshConfig.Timeout) * time.Second,
			Config: ssh.Config{
				KeyExchanges: sshConfig.SshKeyExchanges,
			},
		},
	}
	sshClient.conn, err = sshClient.establishConnection()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	sshClient.session, err = sshClient.createSession(sshClient.conn)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	return sshClient, nil
}

func (s *SshClient) establishConnection() (*ssh.Client, error) {
	sshConn, err := ssh.Dial("tcp", net.JoinHostPort(s.sshConfig.Host, strconv.Itoa(s.sshConfig.Port)), s.clientCfg)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	return sshConn, nil
}

func (s *SshClient) createSession(conn *ssh.Client) (*sftp.Client, error) {
	// open an SFTP session over an existing ssh connection.
	sftp, err := sftp.NewClient(conn, sftp.MaxPacket(_max_packet_size))
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	return sftp, nil
}

func (s *SshClient) checkIfExist(filePath string) (bool, error) {
	var err error
	_, err = s.session.Stat(filePath)
	if err != nil {
		if err == os.ErrNotExist {
			return false, nil
		}
		return false, tracerr.Wrap(err)
	}

	return true, nil
}

func (s *SshClient) GetStoragePath() string {
	return s.sshConfig.SshStoragePath
}

func (s *SshClient) Upload(r io.ReadWriter, filepath_ string) error {
	defer func() {
		if err := recover(); err != nil {
			s.session.RemoveAll(filepath_)
			s.session.Close()
		}
	}()
	isExist, err := s.checkIfExist(filepath_)
	if err != nil {
		return tracerr.Wrap(err)
	}
	if isExist {
		return tracerr.New("file already exists")
	}

	return s.move(r, filepath_)
}

func (s *SshClient) Update(r io.ReadWriter, dst string) error {
	defer func() {
		if err := recover(); err != nil {
			s.session.RemoveAll(dst)
			s.session.Close()
		}
	}()
	return s.move(r, dst)
}

func (s *SshClient) move(r io.ReadWriter, filepath_ string) error {
	// Create the destination file
	err := s.session.MkdirAll(filepath.Dir(filepath_))
	if err != nil {
		return tracerr.Wrap(err)
	}
	dstFile, err := s.session.Create(filepath_)
	if err != nil {
		return tracerr.Wrap(err)
	}
	defer dstFile.Close()

	// write to file
	if _, err = io.CopyN(dstFile, r, _max_packet_size); err != nil {
		if err == io.EOF {
			return nil
		}
		return tracerr.Wrap(err)
	}
	return nil
}

func (s *SshClient) Remove(packetPath string, needVer string, op int) error {
	entries, err := s.session.ReadDir(packetPath)
	if err != nil {
		return tracerr.Wrap(err)
	}
	for _, entry := range entries {
		haveVer := strings.Replace(entry.Name(), filepath.Ext(entry.Name()), "", -1)
		ok, err := utils.CompareVersions(haveVer, needVer, op)
		if err != nil {
			return tracerr.Wrap(err)
		}
		if !ok {
			continue
		}
		log.Println(fmt.Sprintf("removing from %s package...", filepath.Base(packetPath)))
		err = s.session.Remove(filepath.Join(packetPath, entry.Name()))
		if err != nil {
			return tracerr.Wrap(err)
		}
	}
	return nil
}

func (s *SshClient) Download(dst, packetPath, needVer string, op int) error {
	entries, err := s.session.ReadDir(packetPath)
	if err != nil {
		return tracerr.Wrap(err)
	}
	for _, entry := range entries {
		haveVer := strings.Replace(entry.Name(), filepath.Ext(entry.Name()), "", -1)
		ok, err := utils.CompareVersions(haveVer, needVer, op)
		if err != nil {
			return tracerr.Wrap(err)
		}
		if !ok {
			continue
		}

		log.Println(fmt.Sprintf("fetching %s to %s...", filepath.Base(packetPath), dst))

		srcFile, err := s.session.OpenFile(filepath.Join(packetPath, entry.Name()), os.O_RDONLY)
		if err != nil {
			return tracerr.Wrap(err)
		}
		defer srcFile.Close()

		fs_, err := archives.FileSystem(context.Background(), entry.Name(), srcFile)
		if err != nil {
			return tracerr.Wrap(err)
		}

		err = fs.WalkDir(fs_, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return tracerr.Wrap(err)
			}
			if path == "." {
				return nil
			}
			filepath_ := filepath.Join(dst, path)
			if d.IsDir() {
				err = os.MkdirAll(filepath_, 0777)
				if err != nil {
					return tracerr.Wrap(err)
				}
				return nil
			}
			f, err := os.Create(filepath_)
			if err != nil {
				return tracerr.Wrap(err)
			}
			defer f.Close()

			archEntry, err := fs_.Open(path)
			if err != nil {
				return tracerr.Wrap(err)
			}
			defer archEntry.Close()

			_, err = io.Copy(f, archEntry)
			if err != nil {
				return tracerr.Wrap(err)
			}
			return nil
		})
		if err != nil {
			return tracerr.Wrap(err)
		}
	}
	return nil
}

func (s *SshClient) Close() error {
	s.session.Close()
	return s.conn.Close()
}
