package profile

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"pzlauncher/libs/contracts"
)

type ProfileBuilder struct {
	BasePath string // e.g. ./profiles
}

func NewProfileBuilder(base string) *ProfileBuilder { return &ProfileBuilder{BasePath: base} }

// fileSHA256 computes sha256 and size for a file at path.
func fileSHA256(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// Prepare creates profile folders and materializes package blobs from a cache
// directory into a deterministic layout under profiles/<server>/mods/<pkgID>/.
// cacheDir must point to the content-addressable cache (e.g. cache/sha256).
func (b *ProfileBuilder) Prepare(serverID string, manifestID string, packages []contracts.ResolvedPackage, cacheDir string) (string, error) {
	profilePath := filepath.Join(b.BasePath, serverID)
	modsPath := filepath.Join(profilePath, "mods")
	if err := os.MkdirAll(modsPath, 0o755); err != nil {
		return "", fmt.Errorf("mkdir mods: %w", err)
	}

	for _, p := range packages {
		pkgDir := filepath.Join(modsPath, p.ID)
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			return "", fmt.Errorf("mkdir pkg dir: %w", err)
		}
		dest := filepath.Join(pkgDir, p.ID+".pkg")

		cacheBlob := filepath.Join(cacheDir, p.SHA256)
		if _, err := os.Stat(cacheBlob); err != nil {
			// cache missing — create a lightweight placeholder if not present
			if _, err := os.Stat(dest); err == nil {
				continue
			}
			f, err := os.Create(dest)
			if err != nil {
				return "", fmt.Errorf("create placeholder: %w", err)
			}
			_, _ = f.WriteString("placeholder:" + p.ID + "\n")
			f.Close()
			continue
		}

		// integrity check
		sha, size, err := fileSHA256(cacheBlob)
		if err != nil {
			return "", fmt.Errorf("compute sha for %s: %w", cacheBlob, err)
		}
		if sha != p.SHA256 {
			return "", fmt.Errorf("sha mismatch for %s: expected %s got %s", p.ID, p.SHA256, sha)
		}
		if p.Size != 0 && size != p.Size {
			return "", fmt.Errorf("size mismatch for %s: expected %d got %d", p.ID, p.Size, size)
		}

		// idempotent: if dest exists and points to the same blob, continue
		if st, err := os.Lstat(dest); err == nil {
			if st.Mode()&os.ModeSymlink != 0 {
				if target, err := os.Readlink(dest); err == nil && target == cacheBlob {
					continue
				}
			} else {
				// regular file — verify sha
				if s, _, err := fileSHA256(dest); err == nil && s == sha {
					continue
				}
			}
			// fallback: remove mismatched dest
			_ = os.Remove(dest)
		}

		// try symlink (preferred on unix-like)
		if runtime.GOOS != "windows" {
			if err := os.Symlink(cacheBlob, dest); err == nil {
				continue
			}
			// fallthrough to copy on symlink failure
		}

		// copy as fallback (works on Windows)
		if err := copyFile(cacheBlob, dest); err != nil {
			return "", fmt.Errorf("copy blob: %w", err)
		}
	}

	return profilePath, nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	inf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer inf.Close()
	outf, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer outf.Close()
	_, err = io.Copy(outf, inf)
	return err
}
