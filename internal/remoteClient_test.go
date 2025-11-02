package internal

import (
	"RemoteUploader/internal/models"
	"RemoteUploader/internal/utils"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mholt/archives"
	"github.com/ztrue/tracerr"
)

func TestRemoteClient_Create(t *testing.T) {
	var tests = []struct {
		name_       string
		inputPack   models.Pack
		inputUnpack models.Unpack
		wantFiles   []string
	}{
		{
			name_: "test file in files",
			inputPack: models.Pack{
				Packets: []models.Packets{
					{
						Name: "packet-1",
						Ver:  "2.0",
						Targets: []models.Targets{
							{
								Path:    "test/*/*",
								Exclude: "*.ext",
							},
						},
					},
					{
						Name: "packet-1",
						Ver:  "1.0",
						Targets: []models.Targets{
							{
								Path:    "test/*",
								Exclude: "*.noext",
							},
						},
					},
				},
			},
			inputUnpack: models.Unpack{
				Packages: []models.Packages{
					{
						Name: "packet-1",
						Ver:  "<2.0",
					},
				},
			},
			wantFiles: []string{
				"test/file1",
				"test/file2",
				"test/file3.ext",
			},
		},
		{
			name_: "test files",
			inputPack: models.Pack{
				Packets: []models.Packets{{
					Name: "packet-1",
					Ver:  "1.0",
					Targets: []models.Targets{{
						Path:    "test/*",
						Exclude: "*.ext",
					},
					},
				}},
			},
			inputUnpack: models.Unpack{
				Packages: []models.Packages{
					{
						Name: "packet-1",
						Ver:  "=1.0",
					},
				},
			},
		},
	}

	os.Chdir("../")

	remoteFsPath := "remote"
	outputPath := "output"

	defer os.RemoveAll(remoteFsPath)
	defer os.RemoveAll(outputPath)

	for _, tt := range tests {
		os.Mkdir(remoteFsPath, fs.ModePerm)
		os.Mkdir(outputPath, fs.ModePerm)
		client, err := NewRemoteClient(context.Background(), &uploaderMock{})
		if err != nil {
			t.Fatal(tracerr.Sprint(err))
		}

		err = client.create(models.Create(tt.inputPack), remoteStorageMockFunc_create)
		if err != nil {
			t.Fatal(tracerr.Sprint(err))
		}
		for _, p := range tt.inputPack.Packets {
			_, err := os.Stat(fmt.Sprintf("%s/%s/%s.zip", remoteFsPath, p.Name, p.Ver))
			if err != nil {
				t.Fatal(err)
			}
		}

		err = client.fetch(models.Read(tt.inputUnpack), outputPath, remoteStorageMockFunc_fetch)
		if err != nil {
			t.Fatal(tracerr.Sprint(err))
		}
		for _, f := range tt.wantFiles {
			_, err := os.Stat(fmt.Sprintf("%s/%s", outputPath, f))
			if err != nil {
				t.Fatal(err)
			}
		}

		err = client.delete(models.Delete(tt.inputUnpack), remoteStorageMockFunc_remove)
		if err != nil {
			t.Fatal(tracerr.Sprint(err))
		}
		os.RemoveAll(outputPath)
		os.Mkdir(outputPath, fs.ModePerm)
		err = client.fetch(models.Read(tt.inputUnpack), outputPath, remoteStorageMockFunc_fetch)
		if err != nil {
			t.Fatal(tracerr.Sprint(err))
		}
		for _, f := range tt.wantFiles {
			_, err := os.Stat(fmt.Sprintf("%s/%s", outputPath, f))
			if err == nil {
				t.Fatal(err)
			}
		}

		os.RemoveAll(remoteFsPath)
		os.RemoveAll(outputPath)
	}
}

func getFiles(wantFiles []models.Targets) ([]string, error) {
	ret := make([]string, 0)

	for _, p := range wantFiles {
		g, err := filepath.Glob(p.Path)
		if err != nil {
			return nil, err
		}
		for _, f := range g {
			m, err := filepath.Match(p.Exclude, filepath.Base(f))
			if err != nil {
				return nil, err
			}
			fi, err := os.Stat(f)
			if err != nil {
				return nil, err
			}
			if !m && !fi.IsDir() {
				ret = append(ret, f)
			}
		}
	}

	return ret, nil
}

type uploaderMock struct{}

func (u uploaderMock) Remove(filePath string, needVer string, op int) error {
	//	mock
	return nil
}

func (u uploaderMock) Download(dst, filePath, needVer string, op int) error {
	//	mock
	return nil
}

func (u uploaderMock) GetStoragePath() string {
	//	mock
	return "remote"
}

func (u uploaderMock) Upload(r io.ReadWriter, dst string) error {
	//	mock
	return nil
}

func (u uploaderMock) Update(r io.ReadWriter, dst string) error {
	//	mock
	return nil
}

func (u uploaderMock) Close() error {
	//	mock
	return nil
}

func remoteStorageMockFunc_create(r io.ReadWriter, remotePath string) error {
	_, err := os.Stat(remotePath)
	if err == nil {
		return errors.New("file already exists")
	}

	os.MkdirAll(filepath.Dir(remotePath), fs.ModePerm)
	f, err := os.OpenFile(remotePath, os.O_CREATE|os.O_WRONLY, fs.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	if err != nil {
		return err
	}
	return nil
}

func remoteStorageMockFunc_update(r io.ReadWriter, dst string) error {
	f, err := os.OpenFile(dst, os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	if err != nil {
		return err
	}
	return nil
}

func remoteStorageMockFunc_fetch(dst, packagePath string, needVer string, op int) error {
	entries, err := os.ReadDir(packagePath)
	if err != nil {
		return tracerr.Wrap(err)
	}
	for _, entry := range entries {
		haveVer := strings.Replace(entry.Name(), filepath.Ext(entry.Name()), "", -1)
		ok, err := utils.CompareVersions(haveVer, needVer, op)
		if err != nil {
			return nil
		}
		if !ok {
			return nil
		}
		archPath := filepath.Join(packagePath, entry.Name())
		srcFile, err := os.OpenFile(archPath, os.O_RDONLY, 0755)
		if err != nil {
			return tracerr.Wrap(err)
		}
		defer srcFile.Close()

		fs_, err := archives.FileSystem(context.Background(), filepath.Base(archPath), srcFile)
		if err != nil {
			return tracerr.Wrap(err)
		}

		err = fs.WalkDir(fs_, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return tracerr.Wrap(err)
			}
			if d.IsDir() {
				err = os.MkdirAll(filepath.Join(dst, path), fs.ModePerm)
				if err != nil {
					return tracerr.Wrap(err)
				}
				return nil
			}

			f, err := os.Create(fmt.Sprintf("%s/%s", dst, path))
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

func remoteStorageMockFunc_remove(packagePath string, needVer string, op int) error {
	entries, err := os.ReadDir(packagePath)
	if err != nil {
		return tracerr.Wrap(err)
	}
	for _, entry := range entries {
		haveVer := strings.Replace(entry.Name(), filepath.Ext(entry.Name()), "", -1)
		ok, err := utils.CompareVersions(haveVer, needVer, op)
		if err != nil {
			return nil
		}
		if !ok {
			return nil
		}

		err = os.RemoveAll(filepath.Join(packagePath, entry.Name()))
		if err != nil {
			return err
		}
	}

	return nil
}
