package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
)

// copyAndHash streams src to dst, calls progress(bytesWritten) periodically,
// and returns the total bytes written and hex-encoded SHA256 of the content.
func copyAndHash(src io.Reader, dst io.Writer, progress func(int64)) (int64, string, error) {
	h := sha256.New()
	buf := make([]byte, 32*1024)
	var total int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return total, "", werr
			}
			h.Write(buf[:n])
			total += int64(n)
			if progress != nil {
				progress(total)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return total, "", err
		}
	}
	return total, hex.EncodeToString(h.Sum(nil)), nil
}

// jsonMarshalIndent is a thin wrapper around json.MarshalIndent.
func jsonMarshalIndent(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
