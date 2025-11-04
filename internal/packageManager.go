package internal

import (
	"PackageManager/internal/models"
	"PackageManager/internal/utils"
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archives"
	"github.com/ztrue/tracerr"
)

type IPackageManager interface {
	Upload(streamFrom io.ReadWriter, versionStatement string) error
	Update(streamFrom io.ReadWriter, versionStatement string) error
	Remove(versionStatement string) error
	Download(versionStatement string) (models.IArchiveStream, error)
	GetVersions(packageName string) ([]os.FileInfo, error)
	Close() error
}

type PackageManager struct {
	client IPackageManager
}

func NewRemoteClient(ctx context.Context, up IPackageManager) (*PackageManager, error) {

	if ctx == nil {
		return nil, errors.New("context is nil")
	}

	rClient := &PackageManager{
		client: up,
	}

	return rClient, nil
}

func (u *PackageManager) Create(pack models.Create) error {
	return u.create(pack, u.client.Upload, "create")
}

func (u *PackageManager) Update(pack models.Update) error {
	return u.create(models.Create(pack), u.client.Update, "update")
}

func (u *PackageManager) Download(unpack models.Read, output string) error {
	return u.fetch(unpack, output, u.client.Download)
}

func (u *PackageManager) Remove(unpack models.Delete) error {
	return u.delete(unpack, u.client.Remove)
}

// Create method for create and update files on remote storage
// pack - data from packet.json file
// f - function (Create/Update) of storage client (sshClient)
func (u *PackageManager) create(pack models.Create, f func(r io.ReadWriter, dst string) error, action string) error {
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
					if err := u.createArchiveEntry(zipWriter, match); err != nil {
						return tracerr.Wrap(err)
					}
					filesCount++
				}
			}
		}
		zipWriter.Close()
		zipFile.Close()
		err = u.actionZipArchive(localZipPath, fmt.Sprintf("%s/%s", p.Name, p.Ver), os.O_RDONLY, f)
		if err != nil {
			return tracerr.Wrap(err)
		}
		os.Remove(localZipPath)
		log.Printf("package: %s@%s with %d files was %s", p.Name, p.Ver, filesCount, action)
	}
	return nil
}

func (u *PackageManager) fetch(unpack models.Read, output string, f func(versionStatement string) (models.IArchiveStream, error)) error {
	for _, p := range unpack.Packages {
		op, needVer := utils.ParseVersion(p.Ver)
		versions, err := u.client.GetVersions(p.Name)
		if err != nil {
			return tracerr.Wrap(err)
		}
		for _, version := range versions {
			haveVer := strings.Replace(version.Name(), filepath.Ext(version.Name()), "", -1)
			ok, err := utils.CompareVersions(haveVer, needVer, op)
			if err != nil {
				return tracerr.Wrap(err)
			}
			if !ok {
				continue
			}

			log.Println(fmt.Sprintf("fetching %s to %s...", filepath.Base(p.Name), output))

			versionStatement := fmt.Sprintf("%s/%s", p.Name, version.Name())

			packageStream, err := f(versionStatement)
			if err != nil {
				return tracerr.Wrap(err)
			}

			err = u.handleArchive(output, packageStream)
			if err != nil {
				return tracerr.Wrap(err)
			}
		}
	}
	return nil
}

func (u *PackageManager) delete(unpack models.Delete, f func(rm string) error) error {
	for _, p := range unpack.Packages {
		op, needVer := utils.ParseVersion(p.Ver)
		versions, err := u.client.GetVersions(p.Name)
		if err != nil {
			return tracerr.Wrap(err)
		}
		for _, version := range versions {
			haveVer := strings.Replace(version.Name(), filepath.Ext(version.Name()), "", -1)
			ok, err := utils.CompareVersions(haveVer, needVer, op)
			if err != nil {
				return tracerr.Wrap(err)
			}
			if !ok {
				continue
			}

			log.Println(fmt.Sprintf("removing from %s package...", p.Name))

			err = f(filepath.Join(p.Name, version.Name()))
			if err != nil {
				return tracerr.Wrap(err)
			}
		}

	}
	return nil
}

func (u *PackageManager) actionZipArchive(localPath, versionStatement string,
	openFlags int, f func(f io.ReadWriter, dst string) error) error {
	archiveFileStream, err := os.OpenFile(localPath, openFlags, 0666)
	if err != nil {
		return tracerr.Wrap(err)
	}
	defer func() {
		if r := recover(); r != nil {
			archiveFileStream.Close()
		}
	}()
	err = f(archiveFileStream, versionStatement)
	if err != nil {
		return tracerr.Wrap(err)
	}
	if err = archiveFileStream.Close(); err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

func (u *PackageManager) createArchiveEntry(zipWriter *zip.Writer, filepath_ string) error {
	entry, err := zipWriter.Create(filepath_)
	if err != nil {
		return tracerr.Wrap(err)
	}
	localFile, err := os.Open(filepath_)
	if err != nil {
		return tracerr.Wrap(err)
	}
	defer func() {
		if r := recover(); r != nil {
			localFile.Close()
		}
	}()
	_, err = io.Copy(entry, localFile)
	if err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

func (u *PackageManager) handleArchive(output string, packageStream models.IArchiveStream) error {
	fs_, err := archives.FileSystem(context.Background(), "", packageStream)
	if err != nil {
		return tracerr.Wrap(err)
	}
	defer packageStream.Close()

	err = fs.WalkDir(fs_, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return tracerr.Wrap(err)
		}
		if path == "." {
			return nil
		}
		filepath_ := filepath.Join(output, path)
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
	return nil
}
