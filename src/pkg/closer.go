package pkg

import "io"

func CloseQuietly(closer io.Closer) {
	_ = closer.Close()
}
