package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gitlab.com/david_mbuvi/go_asterisks"
)

var version = "1.0.0"

// Get user input for password, masked with asterisk
func getUserPassword(ENV_VAR_NAME string) {
	ENV_VAR_NAME_TEXT := strings.ReplaceAll(ENV_VAR_NAME, "_", " ")
	fmt.Print(ENV_VAR_NAME_TEXT + " (Your LDAP password): ")

	// The password provided from the terminal, echoing as asterisks.
	password, err := go_asterisks.GetUsersPassword("", true, os.Stdin, os.Stdout)
	if err != nil {
		fmt.Println(err.Error())
	}
	password_encoded := base64.StdEncoding.EncodeToString([]byte(password))

	file_name := "/tmp/.cs.env"

	// open file for writing
	file, err := os.OpenFile(file_name, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	// Write text to the file.
	_, err = file.WriteString(ENV_VAR_NAME + "=" + password_encoded + "\n")
	if err != nil {
		fmt.Println(err)
		return
	}
}

func getUserConfig(ENV_VAR_NAME string, DEFAULT_VALUE string) {
	ENV_VAR_NAME_TEXT := strings.ReplaceAll(ENV_VAR_NAME, "_", " ")
	reader := bufio.NewReader(os.Stdin)
	if ENV_VAR_NAME == "OCP_USERNAME" {
		fmt.Print(ENV_VAR_NAME_TEXT + " (Your LDAP username): ")
		// } else if ENV_VAR_NAME == "OCP_PASSWORD" {
		// 	fmt.Print(ENV_VAR_NAME_TEXT + " (Your LDAP password): ")
	} else {
		fmt.Print(ENV_VAR_NAME_TEXT + " (press Enter to use default " + DEFAULT_VALUE + "): ")
	}
	input, _ := reader.ReadString('\n')
	value := input[:len(input)-1]
	if value == "" {
		value = DEFAULT_VALUE
	}
	// if ENV_VAR_NAME == "OCP_PASSWORD" {
	// 	// bytePassword, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
	// 	// password := string(bytePassword)
	// 	value = base64.StdEncoding.EncodeToString([]byte(value))
	// }

	file_name := "/tmp/.cs.env"

	// open file for writing
	file, err := os.OpenFile(file_name, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	// Write text to the file.
	_, err = file.WriteString(ENV_VAR_NAME + "=" + value + "\n")
	if err != nil {
		fmt.Println(err)
		return
	}
}

func connectVPN(vpnName string) {
	exec.Command("osascript", "-e", "tell application \"/Applications/Tunnelblick.app\"", "-e", "disconnect all", "-e", "end tell").Output()
	time.Sleep(1 * time.Second)
	cmd := exec.Command("osascript", "-e", "tell application \"/Applications/Tunnelblick.app\"", "-e", fmt.Sprintf("connect \"%s\"", vpnName), "-e", "end tell")
	_, err := cmd.Output()

	if err != nil {
		fmt.Println("Error executing command:", err)
		return
	}
}

func verifyVpnStatus(vpnName string) bool {
	cmd := exec.Command("osascript", "-e", "tell application \"/Applications/Tunnelblick.app\"", "-e",
		fmt.Sprintf("get state of configurations where name = \"%s\"", vpnName), "-e", "end tell")
	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	return strings.Contains(string(output), "CONNECTED")
}

func loginOpenshift(hostName string, userName string, userPassword string) {
	var cluster string
	if hostName == "https://api.c-th1n.ascendmoney.io:6443" {
		cluster = "OKD (dev, qa, sandbox, hotfix)"
	} else if hostName == "https://api.a-th1n.ascendmoney.io:6443" {
		cluster = "OCP (performance, staging)"
	}
	cmd := exec.Command("oc", "login", hostName, "-u="+userName, "-p="+userPassword, "--insecure-skip-tls-verify=false")
	output, err := cmd.Output()
	if err != nil {
		os.Exit(1)
	}
	fmt.Println(string(output))
	fmt.Printf("Now you are using %s \n", cluster)

}

func loginEks(hostName string, userName string, userPassword string) {
	cmd := exec.Command("cloudopscli", "login")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
	fmt.Printf("Now you are using EKS cluster (qa/performance)\n")

}

func userConfigure() {
	// prepare information
	usr, _ := user.Current()
	dest := fmt.Sprintf("%s/.cs.env", usr.HomeDir)
	os.Remove("/tmp/.cs.env")

	getUserConfig("OKD_HOST", "https://api.c-th1n.ascendmoney.io:6443")
	getUserConfig("OCP_HOST", "https://api.a-th1n.ascendmoney.io:6443")
	getUserConfig("OCP_VPN_NAME", "thp-vpncen_nonprod-admin_v1")
	getUserConfig("EKS_VPN_NAME", "centralize-cloudops")
	getUserConfig("VPN_TIMEOUT_IN_SECONDS", "80")
	getUserConfig("OCP_USERNAME", "")
	getUserPassword("OCP_PASSWORD")

	cmd := exec.Command("cp", "/tmp/.cs.env", dest)
	output, err := cmd.Output()
	if err != nil {
		os.Exit(1)
	}
	fmt.Println(string(output))
}

func getEnvVars() (string, string, string, string, string, string, string) {
	usr, _ := user.Current()
	dotenvFilePath := fmt.Sprintf("%s/.cs.env", usr.HomeDir)
	err := godotenv.Load(dotenvFilePath)
	if err != nil {
		log.Fatal("Error loading dotenv file", err)
	}
	OKD_HOST := os.Getenv("OKD_HOST")
	OCP_HOST := os.Getenv("OCP_HOST")
	OCP_VPN_NAME := os.Getenv("OCP_VPN_NAME")
	EKS_VPN_NAME := os.Getenv("EKS_VPN_NAME")
	VPN_TIMEOUT_IN_SECONDS := os.Getenv("VPN_TIMEOUT_IN_SECONDS")
	OCP_USERNAME := os.Getenv("OCP_USERNAME")
	OCP_PASSWORD, _ := base64.StdEncoding.DecodeString(os.Getenv("OCP_PASSWORD"))
	return OKD_HOST, OCP_HOST, OCP_VPN_NAME, EKS_VPN_NAME, VPN_TIMEOUT_IN_SECONDS, OCP_USERNAME, string(OCP_PASSWORD)
}

func durationToInt(d time.Duration) int {
	return int(d.Seconds())
}

func connectAndLogin(vpnName string, hostName string, userName string, userPassword string, vpnTimeout int, loginFunc func(string, string, string)) {
	if verifyVpnStatus(vpnName) {
		loginFunc(hostName, userName, userPassword)
	} else {
		connectVPN(vpnName)
		startTime := time.Now()

		for {
			if verifyVpnStatus(vpnName) {
				loginFunc(hostName, userName, userPassword)
				break
			} else if durationToInt(time.Duration(time.Since(startTime))) >= vpnTimeout {
				fmt.Println("Timeout reached, unable to connect to VPN")
				break
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func main() {
	var command string
	if len(os.Args) > 1 {
		args := os.Args[1:]
		command = args[0]
	} else {
		fmt.Println("Error. Please run 'cs help' to see the usage")
		os.Exit(1)
	}

	switch {
	case command == "configure":
		userConfigure()
		fmt.Println("\nDone configuration! Please run 'cs help' to see the usage.")
		os.Exit(0)
	case command == "version":
		fmt.Printf("Version: %s\n", version)
	case command == "help":
		fmt.Println(`
Desc: Easy cluster switching utility
Pre-requistes: oc, cloudopscli, tunnelblick installed
Usage:   
  Init configuration:     cs configure
  Access cluster:         cs {cluster_name}
  Example, to access okd: cs okd
  Available clusters:
	okd => OKD cluster (dev/qa/hotfix/sandbox)
	ocp => OCP cluster (performance/staging)
	eks => EKS cluster (qa/performance)
		`)
		os.Exit(0)

	case command == "okd" || command == "ocp" || command == "eks":
		OKD_HOST, OCP_HOST, OCP_VPN_NAME, EKS_VPN_NAME, VPN_TIMEOUT_IN_SECONDS, OCP_USERNAME, OCP_PASSWORD := getEnvVars()
		VPN_TIMEOUT_IN_SECONDS_INT, _ := strconv.Atoi(VPN_TIMEOUT_IN_SECONDS)
		fmt.Printf("Accessing %s cluster...\n", command)

		if command == "eks" {
			connectAndLogin(EKS_VPN_NAME, OCP_HOST, OCP_USERNAME, OCP_PASSWORD, VPN_TIMEOUT_IN_SECONDS_INT, loginEks)
		} else if command == "ocp" {
			connectAndLogin(OCP_VPN_NAME, OCP_HOST, OCP_USERNAME, OCP_PASSWORD, VPN_TIMEOUT_IN_SECONDS_INT, loginOpenshift)
		} else if command == "okd" {
			connectAndLogin(OCP_VPN_NAME, OKD_HOST, OCP_USERNAME, OCP_PASSWORD, VPN_TIMEOUT_IN_SECONDS_INT, loginOpenshift)
		}

	default:
		fmt.Printf("Error! No cluster or command '%s' available. Please run 'cs help' to see the usage.", command)
	}
}
