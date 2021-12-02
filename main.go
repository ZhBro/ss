package main

import (
	"bufio"
	"easySsh/config"
	"easySsh/utils"
	"errors"
	"flag"
	"fmt"
	"github.com/howeyc/gopass"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var sshId string

var sshInfo = flag.String("add", "", "server information. Example: root@10.0.0.1:22")
var password string
var server config.Ssh

func init() {
	flag.Parse()

	var err error
	if len(os.Args) == 1 {
		config.ShowSshList()
		sshId, err = getSshId()
		if err != nil {
			fmt.Printf("something wrong: %s", err.Error())
			os.Exit(1)
		}

		*sshInfo, password, err = config.ServerConfig.GetServerById(sshId)
		if err != nil {
			fmt.Printf("something wrong: %s", err.Error())
			os.Exit(1)
		}
	}

	if *sshInfo != "" {
		server, err = parseSshInfo(*sshInfo)
		if err != nil {
			fmt.Printf("something wrong: %s", err.Error())
			os.Exit(1)
		}

		var pwd []byte
		if password == "" {
			fmt.Println("Password:")
			pwd, err = gopass.GetPasswd()
			if err != nil {
				fmt.Printf("something wrong: %s", err.Error())
				os.Exit(1)
			}
		} else {
			pwd, err = utils.AesDecrypt([]byte(password), []byte(config.ServerConfig.Cipher))
			if err != nil {
				fmt.Printf("something wrong: %s", err.Error())
				os.Exit(1)
			}
		}

		server.Password = string(pwd)
	} else {
		fmt.Printf("no server")
		os.Exit(0)
	}

}

func main() {

	session, err := connect(server.User, server.Address, server.Password, server.Port)
	if err != nil {
		fmt.Printf("something wrong: %s", err.Error())
		os.Exit(1)
	}
	defer session.Close()

	encPwd, err := utils.AesEncrypt([]byte(server.Password), []byte(config.ServerConfig.Cipher))
	if err != nil {
		fmt.Printf("something wrong: %s", err.Error())
		os.Exit(1)
	}
	err = config.ServerConfig.UpdateConfig(*sshInfo, string(encPwd))
	if err != nil {
		fmt.Printf("something wrong: %s", err.Error())
		os.Exit(1)
	}

	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(fd, oldState)

	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	termWidth, termHeight, err := terminal.GetSize(fd)
	if err != nil {
		panic(err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 1,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm-256color", termHeight, termWidth, modes); err != nil {
		log.Fatal(err)
	}

	go func() {
		sigwinchCh := make(chan os.Signal, 1)
		signal.Notify(sigwinchCh, syscall.SIGWINCH)
		for {
			select {
			case _ = <-sigwinchCh:
				newWidth, newHeight, _ := terminal.GetSize(fd)
				if newWidth != termWidth || newHeight != termHeight {
					termWidth, termHeight = newWidth, newHeight
					session.WindowChange(termHeight, termWidth)
				}
			}
		}
	}()

	err = session.Run("bash")
	if err != nil {
		err = session.Run("sh")
		if err != nil {
			fmt.Println("remote server can not user bash or sh.")
			os.Exit(1)
		}
	}
}

func connectKey(user string, host string, port int) (*ssh.Session, error) {
	var (
		addr         string
		auth         []ssh.AuthMethod
		clientConfig *ssh.ClientConfig
		client       *ssh.Client
		session      *ssh.Session
		err          error
	)

	auth = make([]ssh.AuthMethod, 0)

	privateKey := `-----BEGIN RSA PRIVATE KEY-----
private key data
                                                                                                                                                -----END RSA PRIVATE KEY-----`
	key, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, err
	}
	auth = append(auth, ssh.PublicKeys(key))

	clientConfig = &ssh.ClientConfig{
		User:    user,
		Auth:    auth,
		Timeout: 30 * time.Second,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	addr = host + ":" + strconv.Itoa(port)
	fmt.Println(addr)

	if client, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}

	if session, err = client.NewSession(); err != nil {
		return nil, err
	}

	return session, nil
}

//ssh remote server
/**
user string default root
host string remote-ip
port int default 22
*/
func connect(user string, host string, password string, port int) (*ssh.Session, error) {
	var (
		addr         string
		auth         []ssh.AuthMethod
		clientConfig *ssh.ClientConfig
		client       *ssh.Client
		session      *ssh.Session
		err          error
	)

	auth = make([]ssh.AuthMethod, 0)

	auth = append(auth, ssh.Password(password))

	clientConfig = &ssh.ClientConfig{
		User:    user,
		Auth:    auth,
		Timeout: 30 * time.Second,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	addr = host + ":" + strconv.Itoa(port)
	fmt.Println(addr)

	if client, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}

	if session, err = client.NewSession(); err != nil {
		return nil, err
	}

	return session, nil
}

func getSshId() (string, error) {
	fmt.Println("Index:")
	reader := bufio.NewReader(os.Stdin)
	cmdString, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return cmdString[:len(cmdString)-1], nil
}

func parseSshInfo(s string) (config.Ssh, error) {
	b := []byte(s)
	var ssh config.Ssh

	index := strings.IndexByte(s, '@')
	if index >= 0 {
		ssh.User = string(b[:index])
		b = b[index+1:]
	} else {
		ssh.User = "root"
	}

	index = strings.IndexByte(string(b), ':')
	if index >= 0 {
		ssh.Address = string(b[:index])
		port, err := strconv.Atoi(string(b[index+1:]))
		if err != nil {
			return config.Ssh{}, err
		}
		ssh.Port = port
	} else {
		ssh.Address = string(b)
		ssh.Port = 22
	}

	address := net.ParseIP(ssh.Address)
	if address == nil {
		err := errors.New("address is illegal")
		return config.Ssh{}, err
	}

	if ssh.Port < 0 || ssh.Port > 65535 {
		err := errors.New("port is illegal")
		return config.Ssh{}, err
	}
	return ssh, nil
}
