package config

import (
	"easySsh/utils"
	"errors"
	"fmt"
	ini "gopkg.in/ini.v1"
	"math/rand"
	"os"
	"strconv"
)

type Config struct {
	WorkPath string
	IniCfg *ini.File
	Section *ini.Section
	Cipher string
	Servers map[string]*Server
}

type Server struct {
	Index int
	Password string
}

type Ssh struct {
	User string
	Address string
	Port int
	Password string
}

var ServerConfig *Config
func init() {
	var err error
	ServerConfig, err = New()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = ServerConfig.LoadConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func New() (*Config, error) {
	homePath := os.Getenv("HOME")
	workDir := homePath + "/.easy-ssh"
	if !utils.IsExists(workDir) {
		err := os.MkdirAll(workDir, 0777)
		if err != nil {
			return &Config{}, err
		}
	}
	cfg, section, err := loadIni(workDir + "/servers.conf")
	if err != nil {
		return &Config{}, err
	}
	cipher, err := readSymCipher(workDir + "/.key")
	if err != nil {
		return &Config{}, err
	}
	return &Config{workDir, cfg, section, cipher, nil}, nil
}

func (c *Config)LoadConfig() error {
	var err error
	servers, err := c.readServers()
	if err != nil {
		return err
	}
	c.Servers = servers

	return nil
}

func (c *Config)UpdateConfig(ssh string, encPwd string) error {
	if c.Servers[ssh] == nil {
		c.Servers[ssh] = &Server{}
	}
	c.Servers[ssh].Password = encPwd
	c.Section.Key(ssh).SetValue(encPwd)
	err := c.IniCfg.SaveTo(c.WorkPath + "/servers.conf")
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) GetServerById(id string) (string, string, error) {
	index, err := strconv.Atoi(id)
	if err != nil {
		return "", "", err
	}
	for key, value := range c.Servers {
		if value.Index == index {
			return key, value.Password, nil
		}
	}
	return "", "", errors.New("index is out of range")
}

func readSymCipher(path string) (string, error) {
	exist := utils.IsExists(path)
	if exist {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	} else {
		cipher := createSymCipher()
		err := os.WriteFile(path, []byte(cipher), 0666)
		if err != nil {
			return "", err
		}
		return cipher, nil
	}
}

// RandomString returns a random string with a fixed length
func randomString(n int) string {
	var defaultLetters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	var letters []rune

	letters = defaultLetters

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}

func createSymCipher() string {
	return randomString(32)
}


func (c Config) readServers() (map[string]*Server,error) {
	serversConfig := c.WorkPath + "/servers.conf"
	if !utils.IsExists(serversConfig) {
		err := os.WriteFile(serversConfig, []byte(""), 0666)
		if err != nil {
			return nil, err
		}
		return nil, nil
	} else {
		keys := c.Section.KeyStrings()
		m := make(map[string]*Server)
		for i, key := range keys {
			s := Server{}
			s.Index = i
			s.Password = c.Section.Key(key).Value()
			m[key] = &s
		}
		return m, nil
	}
}

func loadIni(path string) (*ini.File, *ini.Section, error) {
	serversConfig := path
	if !utils.IsExists(serversConfig) {
		err := os.WriteFile(serversConfig, []byte(""), 0666)
		if err != nil {
			return nil, nil, err
		}
	}
	cfg, err := ini.Load(path)
	if err != nil {
		return nil, nil, err
	}
	return cfg, cfg.Section(""), nil
}

func ShowSshList() {
	for key, value := range ServerConfig.Servers {
		fmt.Printf("%d.      %s\n", value.Index, key)
	}
}


func SaveSshConfig(server string, password string, key string, path string) {
	serversConfig := path + "/servers.conf"
	encData, err := utils.AesEncrypt([]byte(password), []byte(key))
	if err != nil {
		return
	}
	data := fmt.Sprintf("%s %s", server, encData)
	err = utils.AppendToFile(data, serversConfig)
	if err != nil {
		return
	}
}
