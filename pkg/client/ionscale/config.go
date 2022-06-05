package ionscale

import (
	"github.com/mitchellh/go-homedir"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

const (
	DefaultDir         string      = "~/.ionscale"
	DefaultPermissions os.FileMode = 0700
)

func TokenFromFile() (string, error) {
	return valueFromFile("token")
}

func TailnetFromFile() (uint64, error) {
	v, err := valueFromFile("tailnet_id")
	if v == "" {
		return 0, nil
	}
	p, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, err
	}
	return p, err
}

func valueFromFile(name string) (string, error) {
	file, err := EnsureFile(name)
	if err != nil {
		return "", err
	}
	token, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(token), nil
}

func TokenToFile(token string) error {
	file, err := EnsureFile("token")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(file, []byte(token), 0600)
}

func TailnetToFile(id *uint64) error {
	file, err := EnsureFile("tailnet_id")
	if err != nil {
		return err
	}

	var v = ""

	if id != nil {
		v = strconv.FormatUint(*id, 10)
	}

	return ioutil.WriteFile(file, []byte(v), 0600)
}

func ConfigDir() string {
	return DefaultDir
}

func EnsureFile(file string) (string, error) {
	permission := DefaultPermissions
	dir := ConfigDir()
	dirPath, err := homedir.Expand(dir)
	if err != nil {
		return "", err
	}

	filePath := path.Clean(filepath.Join(dirPath, file))
	if err := os.MkdirAll(filepath.Dir(filePath), permission); err != nil {
		return "", err
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return "", err
		}
		defer file.Close()
	}

	return filePath, nil
}
