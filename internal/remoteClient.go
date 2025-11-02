package internal

import (
	"RemoteUploader/internal/models"
	"RemoteUploader/internal/utils"
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/ztrue/tracerr"
)

type IRemoteClient interface {
	Upload(r io.ReadWriter, dst string) error
	Update(r io.ReadWriter, dst string) error
	Remove(filePath string, needVer string, op int) error
	Download(dst, filePath, needVer string, op int) error
	GetStoragePath() string
	Close() error
}

type RemoteClient struct {
	client IRemoteClient
}

func NewRemoteClient(ctx context.Context, up IRemoteClient) (*RemoteClient, error) {

	if ctx == nil {
		return nil, errors.New("context is nil")
	}

	rClient := &RemoteClient{
		client: up,
	}

	return rClient, nil
}

func (u *RemoteClient) Create(pack models.Create) error {
	return u.create(pack, u.client.Upload, "create")
}

func (u *RemoteClient) Update(pack models.Update) error {
	return u.create(models.Create(pack), u.client.Update, "update")
}

func (u *RemoteClient) Download(unpack models.Read, output string) error {
	return u.fetch(unpack, output, u.client.Download)
}

func (u *RemoteClient) Remove(unpack models.Delete) error {
	return u.delete(unpack, u.client.Remove)
}

// Create method for create and update files on remote storage
// pack - data from packet.json file
// f - function (Create/Update) of storage client (sshClient)
func (u *RemoteClient) create(pack models.Create, f func(r io.ReadWriter, dst string) error, action string) error {
	for _, p := range pack.Packets {
		localZipPath := fmt.Sprintf("%s_%s.zip", p.Name, p.Ver)
		zipFile, err := os.Create(localZipPath)
		if err != nil {
			return tracerr.Wrap(err)
		}
		filesCount := 0
		zipWriter := zip.NewWriter(zipFile)
		defer func() {
			if r := recover(); r != nil {
				zipWriter.Close()
				zipFile.Close()
				os.Remove(localZipPath)
			}
		}()
		for _, t := range p.Targets {
			matches, err := filepath.Glob(t.Path)
			if err != nil {
				return tracerr.Wrap(err)
			}
			for _, match := range matches {
				exclude, err := filepath.Match(t.Exclude, filepath.Base(match))
				if err != nil {
					return tracerr.Wrap(err)
				}
				stat, err := os.Stat(match)
				if err != nil {
					return tracerr.Wrap(err)
				}
				if !exclude && !stat.IsDir() {
					if err := u.packArchive(zipWriter, match); err != nil {
						return tracerr.Wrap(err)
					}
					filesCount++
				}
			}
		}
		zipWriter.Close()
		zipFile.Close()
		err = u.actionFile(localZipPath, fmt.Sprintf("%s/%s/%s.zip", u.client.GetStoragePath(), p.Name, p.Ver), os.O_RDONLY, f)
		if err != nil {
			return tracerr.Wrap(err)
		}
		os.Remove(localZipPath)
		log.Printf("package: %s@%s with %d files was %s", p.Name, p.Ver, filesCount, action)
	}
	return nil
}

func (u *RemoteClient) fetch(unpack models.Read, output string, f func(dst, archPath, needVer string, op int) error) error {
	for _, p := range unpack.Packages {
		op, needVer := utils.ParseVersion(p.Ver)
		err := f(output, fmt.Sprintf("%s/%s", u.client.GetStoragePath(), p.Name), needVer, op)
		if err != nil {
			return tracerr.Wrap(err)
		}
	}
	return nil
}

func (u *RemoteClient) delete(unpack models.Delete, f func(rm string, needVer string, op int) error) error {
	for _, p := range unpack.Packages {
		op, needVer := utils.ParseVersion(p.Ver)
		err := f(fmt.Sprintf("%s/%s", u.client.GetStoragePath(), p.Name), needVer, op)
		if err != nil {
			return tracerr.Wrap(err)
		}
	}
	return nil
}

func (u *RemoteClient) actionFile(localPath, remoteStoragePath string, openFlags int, f func(f io.ReadWriter, dst string) error) error {
	fetched, err := os.OpenFile(localPath, openFlags, 0666)
	if err != nil {
		return tracerr.Wrap(err)
	}
	defer func() {
		if r := recover(); r != nil {
			fetched.Close()
		}
	}()
	err = f(fetched, remoteStoragePath)
	if err != nil {
		return tracerr.Wrap(err)
	}
	if err = fetched.Close(); err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

func (u *RemoteClient) packArchive(zipWriter *zip.Writer, filepath_ string) error {
	entry, err := zipWriter.Create(filepath_)
	if err != nil {
		return tracerr.Wrap(err)
	}
	matchFile, err := os.Open(filepath_)
	if err != nil {
		return tracerr.Wrap(err)
	}
	defer func() {
		if r := recover(); r != nil {
			matchFile.Close()
		}
	}()
	_, err = io.Copy(entry, matchFile)
	if err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}
