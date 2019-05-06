package tools

import (
	"io"
)

type UnblockedWriter chan []byte

func Unblock(wc io.WriteCloser) io.WriteCloser {

	uw := make(UnblockedWriter)
	go func() {
		for b := range uw {
			wc.Write(b)
			//TODO catch and report errors
		}
		wc.Close()
	}()
	return uw
}

func (uw UnblockedWriter) Write(p []byte) (n int, err error) {

	uw <- p
	return len(p), nil
}

func (uw UnblockedWriter) Close() error {

	close(uw)
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type multicloser struct {
	closer []io.Closer
}

func (closer *multicloser) Close() (err error) {
	for _, c := range closer.closer {
		if err == nil {
			c.Close()
		} else {
			err = c.Close()
		}
	}
	return
}

func MultiCloser(closer ...io.Closer) io.Closer {
	return &multicloser{closer: closer}
}
