package models

import "io"

type write struct {
	Reader io.Reader
	Path   string
}

type Create Pack

type Update Create

type Delete Unpack

type Read Unpack
