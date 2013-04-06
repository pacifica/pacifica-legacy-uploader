package pipepair

import (
	"io"
)

type PipePair struct {
	In io.ReadCloser
	Out io.WriteCloser
}

func (self PipePair) Read(p []byte) (n int, err error) {
	return self.In.Read(p)
}

func (self PipePair) Write(p []byte) (n int, err error) {
	return self.Out.Write(p)
}

func (self PipePair) Close() error {
	err := self.In.Close()
	if err != nil {
		return err
	}
	return self.Out.Close()
}
