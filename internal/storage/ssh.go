package storage

import (
	"PackageManager/internal/configs"
	"PackageManager/internal/models"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

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

func (s *SshClient) Upload(streamFrom io.ReadWriter, versionStatement string) error {
	versionStatement = s.setExt(s.setPrefix(versionStatement))

	isExist, err := s.checkPacketIfExist(versionStatement)
	if err != nil {
		return tracerr.Wrap(err)
	}
	if isExist {
		return tracerr.New("file already exists")
	}

	streamTo, err := s.createPacketStream(versionStatement)
	if err != nil {
		return tracerr.Wrap(err)
	}

	return s.copyPacket(streamFrom, streamTo)
}

func (s *SshClient) Update(streamFrom io.ReadWriter, versionStatement string) error {
	versionStatement = s.setExt(s.setPrefix(versionStatement))

	streamTo, err := s.createPacketStream(versionStatement)
	if err != nil {
		return tracerr.Wrap(err)
	}

	return s.copyPacket(streamFrom, streamTo)
}

func (s *SshClient) Remove(versionStatement string) error {
	versionStatement = s.setPrefix(versionStatement)
	err := s.session.Remove(versionStatement)
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}

func (s *SshClient) Download(versionStatement string) (models.IArchiveStream, error) {
	versionStatement = s.setPrefix(versionStatement)
	srcFile, err := s.session.OpenFile(versionStatement, os.O_RDONLY)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	return srcFile, nil
}

func (s *SshClient) GetVersions(packageName string) ([]os.FileInfo, error) {
	entries, err := s.session.ReadDir(s.setPrefix(packageName))
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	return entries, nil
}

func (s *SshClient) Close() error {
	s.session.Close()
	return s.conn.Close()
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

func (s *SshClient) checkPacketIfExist(filePath string) (bool, error) {
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

func (s *SshClient) getStoragePath() string {
	return s.sshConfig.SshStoragePath
}

func (s *SshClient) copyPacket(streamFrom io.ReadWriter, streamTo *sftp.File) error {
	// write to file
	defer streamTo.Close()
	if _, err := io.CopyN(streamTo, streamFrom, _max_packet_size); err != nil {
		if err == io.EOF {
			return nil
		}
		return tracerr.Wrap(err)
	}
	return nil
}

func (s *SshClient) createPacketStream(fullPath string) (*sftp.File, error) {
	// Create the destination file
	err := s.session.MkdirAll(filepath.Dir(fullPath))
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	streamTo, err := s.session.Create(fullPath)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	return streamTo, nil
}

func (s *SshClient) setPrefix(input string) string {
	return fmt.Sprintf("%s/%s", s.getStoragePath(), input)
}

func (s *SshClient) setExt(input string) string {
	return fmt.Sprintf("%s.%s", input, "zip")
}
