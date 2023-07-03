package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/eiannone/keyboard"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"os/user"
	"strings"
)

var configFile string

func main() {
	// 错误信息 忽略代码堆栈信息
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Error:", err)
		}
	}()

	// 配置文件
	configFile = GetConfFile()

	// 命令行参数逻辑
	switch len(os.Args) {
	case 1:
		Choose()
	case 2:
		command := os.Args[1]
		switch command {
		case "version":
			fmt.Println("Version: 0.1.1")
			fmt.Println("Author : :lan")
		case "add":
			Add()
		case "del":
			Delete()
		case "list":
			List()
		default:
			fmt.Println("sshgo add : 进入添加配置流程")
			fmt.Println("sshgo del : 进入删除配置流程")
			fmt.Println("sshgo list: 查看当前配置列表")
		}
	default:
		fmt.Println("Usage: sshgo")
		fmt.Println("Usage: sshgo < add | del | list>")
		os.Exit(1)
	}
}

/* ====== 类 ====== */

type ServerConfig struct {
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type ServerListConfig struct {
	Servers []ServerConfig `json:"servers"`
}

/* ====== 命令行指令 ====== */

func Add() {
	name := promptUserInput("Server Name: ")
	ip := promptUserInput("Server Addr: ")
	port := promptUserInput("Server Port: ")
	username := promptUserInput("Username: ")
	password := promptUserInput("Password: ")

	newServer := ServerConfig{
		Name:     name,
		IP:       ip,
		Port:     port,
		Username: username,
		Password: password,
	}

	config := readConfigFile()
	config.Servers = append(config.Servers, newServer)
	writeConfigFile(config)
}

func List() {
	config := readConfigFile()
	servers := config.Servers
	for i := 0; i < len(servers); i++ {
		fmt.Printf("%-5s | %s@%s\n", servers[i].Name, servers[i].Username, servers[i].IP)
	}
}

func Delete() {
	config := readConfigFile()
	servers := config.Servers
	newServers := ServerListConfig{}

	clearConsole()
	fmt.Println("Server List:")
	List()

	name := promptUserInput("\n输入 all 清空全部配置\nPlease Input Remove Name: ")
	for i, server := range servers {
		if server.Name == name {
			// 通过省略号 ... 将第二个切片展开为单个元素序列
			newServers.Servers = append(servers[:i], servers[i+1:]...)
		} else if server.Name == "all" {
			newServers.Servers = []ServerConfig{}
		}
	}
	writeConfigFile(newServers)
}

/* ====== 工具函数 ====== */

func GetConfFile() string {
	// 返回配置文件的绝对路径
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("未获取到当前用户目录，配置文件初始化失败！")
		panic(err)
	}
	return currentUser.HomeDir + "/.sshgo"
}

func errHandler(err error) {
	if err != nil {
		panic(err)
		return
	}
}

func clearConsole() {
	// 清空控制台输出
	fmt.Print("\033[H\033[2J")
}

func promptUserInput(prompt string) string {
	// 提示用户输入
	for {
		fmt.Print(prompt)
		var input string
		fmt.Scanln(&input)
		input = strings.TrimSpace(input)
		if input == "" {
			fmt.Println("Input cannot be empty. Please try again.")
		}
		return input
	}
}

func readConfigFile() ServerListConfig {
	// 文件如果不存在 则跳出
	_, err := os.Stat(configFile)
	if err != nil {
		return ServerListConfig{}
	}

	// 文件若存在 则读取配置
	var servers ServerListConfig

	file, err := os.Open(configFile)
	if err != nil {
		fmt.Println("打开配置文件失败:", configFile)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.Decode(&servers)
	return servers
}

func writeConfigFile(servers ServerListConfig) {
	serversStr, err := json.Marshal(servers)
	if err != nil {
		fmt.Println("writeConfigFile json marshal err: ", err)
		return
	}

	var out bytes.Buffer
	err = json.Indent(&out, serversStr, "", "\t")
	if err != nil {
		fmt.Println("writeConfigFile json indent err: ", err)
		return
	}

	file, _ := os.Create(configFile)
	defer file.Close()
	_, err = out.WriteTo(file)
	if err != nil {
		fmt.Println("writeConfigFile err: ", err)
		return
	}
}

func Choose() {
	err := keyboard.Open()
	if err != nil {
		panic(err)
	}
	defer keyboard.Close()

	servers := readConfigFile().Servers

	// 默认选中第一行
	selectedIndex := 0
	clearConsole()
	fmt.Println("Server List:")

	// 首次默认选中
	for i, server := range servers {
		if i == selectedIndex {
			fmt.Printf("> %-5s | %s@%s\n", server.Name, server.Username, server.IP)
		} else {
			fmt.Printf("  %-5s | %s@%s\n", server.Name, server.Username, server.IP)
		}
	}

	for {
		// 响应键盘上下键
		_, key, _ := keyboard.GetKey()

		if key == keyboard.KeyArrowUp && selectedIndex > 0 {
			selectedIndex--
		} else if key == keyboard.KeyArrowDown && selectedIndex < len(servers)-1 {
			selectedIndex++
		} else if key == keyboard.KeyEnter {
			break
		}

		// 根据选择刷新箭头位置
		clearConsole()
		fmt.Println("Server List:")
		for i, server := range servers {
			if i == selectedIndex {
				fmt.Printf("> %-5s | %s@%s\n", server.Name, server.Username, server.IP)
			} else {
				fmt.Printf("  %-5s | %s@%s\n", server.Name, server.Username, server.IP)
			}
		}
	}

	selectedServer := servers[selectedIndex]
	fmt.Printf("\nConnect to %-5s | %s@%s...\n", selectedServer.Name, selectedServer.Username, selectedServer.IP)

	Connect(&selectedServer)
}

func Connect(server *ServerConfig) {
	// 配置连接 SSH 服务器
	config := &ssh.ClientConfig{
		User: server.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(server.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 发起连接
	client, err := ssh.Dial("tcp", server.IP+":"+server.Port, config)
	errHandler(err)
	defer client.Close()

	// 创建一个新的会话
	session, err := client.NewSession()
	errHandler(err)
	defer session.Close()

	// 将标准输入设置为原始模式，用户输入直接传递给程序，以便可以进行交互式shell会话
	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	errHandler(err)
	defer terminal.Restore(fd, oldState) // 退出程序后恢复原终端结构体指针

	// 创建伪终端（PTY）
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // 打开回显
		ssh.TTY_OP_ISPEED: 14400, // 输入速度 14.4k
		ssh.TTY_OP_OSPEED: 14400, // 输出速度 14.4k
	}
	// 获取当前终端的行数和列数 给到伪终端
	termWidth, termHeight, _ := terminal.GetSize(fd)
	err = session.RequestPty("xterm-256color", termHeight, termWidth, modes)
	errHandler(err)

	// 将会话的输入输出连接到标准输入输出
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	// 启动 shell 会话，进入远程服务器
	err = session.Shell()
	errHandler(err)

	err = session.Wait()
	errHandler(err)
}
