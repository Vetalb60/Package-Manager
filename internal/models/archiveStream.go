package models

import "io"

type IArchiveStream interface {
	io.ReadSeeker
	io.Closer
	io.ReaderAt
}
