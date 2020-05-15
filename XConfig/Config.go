package XConfig

import (
	"errors"
	"github.com/widuu/goini"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)
const(
	//Debug Version
	Debug bool = true
	// Version
	Version string = "1.1"
)

type Config struct {
	conf *goini.Config
	Lock sync.Mutex
}
func InitConfig(ElfPath string) *Config{
	ConfigFile := ElfPath + "config.ini"
	conf := &Config{conf:goini.SetConfig(ConfigFile)}
	return conf
}
func (c *Config) Get(section string,key string) string{
	c.Lock.Lock()
	CValue := c.conf.GetValue(section,key)
	c.Lock.Unlock()
	if CValue == "no value"{
		return ""
	}else{
		return string(CValue)
	}
}

func GetCurrentPath() (string, error) {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	i := strings.LastIndex(path, "/")
	if i < 0 {
		i = strings.LastIndex(path, "\\")
	}
	if i < 0 {
		return "", errors.New(`error: Can't find "/" or "\".`)
	}
	return string(path[0 : i+1]), nil
}