package config

import (
	"encoding/hex"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

type Duration time.Duration

func (d *Duration) UnmarshalText(b []byte) error {
	v, err := time.ParseDuration(string(b))
	if err != nil {
		return err
	}
	*d = Duration(v)
	return nil
}

type HexString []byte

func (d *HexString) UnmarshalText(b []byte) error {
	v, err := hex.DecodeString(string(b))
	if err != nil {
		return err
	}
	*d = HexString(v)
	return nil
}

type Path string

func (p *Path) UnmarshalText(b []byte) error {
	*p = ToPath(string(b))
	return nil
}

func ToPath(path string) Path {
	return Path(expandHomeVar(path))
}

func expandHomeVar(path string) string {
	segments := strings.Split(path, string(os.PathSeparator))
	if len(segments) == 0 {
		return path
	}
	username, isHomePath := strings.CutPrefix(segments[0], "~")
	if !isHomePath {
		return path
	}

	var u *user.User
	var err error
	if username == "" {
		u, err = user.Current()
	} else {
		u, err = user.Lookup(username)
	}
	if err != nil {
		return path
	}

	if u.HomeDir == "" {
		return path
	}

	segments[0] = u.HomeDir
	return filepath.Join(segments...)
}
