package logger

import "os"

type fileWrite struct {
	fp *os.File
}

func newFileWrite(filename string) (*fileWrite, error) {
	fp, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &fileWrite{fp: fp}, nil
}

func (f *fileWrite) Write(p []byte) (n int, err error) {
	return f.fp.Write(p)
}

func (f *fileWrite) Close() {
	f.fp.Close()
}
