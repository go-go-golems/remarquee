package devicecapture

import "io"

type readAtCloser interface {
	io.ReaderAt
	io.Closer
}
