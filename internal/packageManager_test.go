package internal

import (
	"PackageManager/internal/models"
	"PackageManager/internal/utils"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ztrue/tracerr"
)

var remoteFsPath = "remote"
var outputPath = "output"

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

	defer os.RemoveAll(remoteFsPath)
	defer os.RemoveAll(outputPath)

	for _, tt := range tests {
		os.Mkdir(remoteFsPath, fs.ModePerm)
		os.Mkdir(outputPath, fs.ModePerm)
		mock := &uploaderMock{}
		client, err := NewRemoteClient(context.Background(), mock)
		if err != nil {
			t.Fatal(tracerr.Sprint(err))
		}

		err = client.create(models.Create(tt.inputPack), remoteStorageMockFunc_create, "create")
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

		for _, p := range tt.inputUnpack.Packages {
			op, needVer := utils.ParseVersion(p.Ver)
			versions, err := mock.GetVersions(p.Name)
			if err != nil {
				t.Fatal(tracerr.Sprint(err))
			}
			for _, version := range versions {
				haveVer := strings.Replace(version.Name(), filepath.Ext(version.Name()), "", -1)
				ok, err := utils.CompareVersions(haveVer, needVer, op)
				if err != nil {
					t.Fatal(tracerr.Sprint(err))
				}
				if ok {
					t.Fatal(tracerr.Sprint(fmt.Errorf("version %s not match", version.Name())))
				}
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

func (u uploaderMock) Remove(versionStatement string) error {
	//	mock
	return nil
}

func (u uploaderMock) Download(versionStatement string) (models.IArchiveStream, error) {
	//	mock
	return nil, nil
}

func (u uploaderMock) GetVersions(versionStatement string) ([]os.FileInfo, error) {
	packagePath := fmt.Sprintf("%s/%s", remoteFsPath, versionStatement)
	entries, err := os.ReadDir(packagePath)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	ret := make([]os.FileInfo, 0)
	for _, entry := range entries {
		fi, err := entry.Info()
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
		ret = append(ret, fi)
	}

	return ret, nil
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

func remoteStorageMockFunc_create(streamFrom io.ReadWriter, versionStatement string) error {
	versionStatement = fmt.Sprintf("%s/%s.zip", remoteFsPath, versionStatement)
	_, err := os.Stat(versionStatement)
	if err == nil {
		return errors.New("file already exists")
	}

	os.MkdirAll(filepath.Dir(versionStatement), fs.ModePerm)
	f, err := os.OpenFile(versionStatement, os.O_CREATE|os.O_WRONLY, fs.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, streamFrom)
	if err != nil {
		return err
	}
	return nil
}

func remoteStorageMockFunc_update(r io.ReadWriter, versionStatement string) error {
	versionStatement = fmt.Sprintf("%s/%s.zip", remoteFsPath, versionStatement)
	os.MkdirAll(filepath.Dir(versionStatement), fs.ModePerm)
	f, err := os.OpenFile(versionStatement, os.O_TRUNC|os.O_WRONLY, 0755)
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

func remoteStorageMockFunc_fetch(versionStatement string) (models.IArchiveStream, error) {
	srcFile, err := os.OpenFile(fmt.Sprintf("%s/%s", remoteFsPath, versionStatement), os.O_RDONLY, 0755)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	return srcFile, nil
}

func remoteStorageMockFunc_remove(versionStatement string) error {
	err := os.RemoveAll(fmt.Sprintf("%s/%s", remoteFsPath, versionStatement))
	if err != nil {
		return err
	}

	return nil
}
