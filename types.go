package main

type FileObject struct {
	Source   int64 // used by sync
	Name     string
	Size     int64
	Checksum string
}
