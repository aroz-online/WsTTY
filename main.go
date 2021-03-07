package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"imuslab.com/WsTTY/mod/aroz"
	"imuslab.com/WsTTY/mod/wsshell"
)

var (
	handler *aroz.ArozHandler
)

func SetupCloseHandler() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("\r- Shutting down WsTTY Service.")

		os.Exit(0)
	}()
}

func main() {
	//Create a platform dependent aroz service register
	if runtime.GOOS == "windows" {
		//Start the aoModule pipeline (which will parse the flags as well). Pass in the module launch information
		handler = aroz.HandleFlagParse(aroz.ServiceInfo{
			Name:        "WsTTY",
			Desc:        "arozos Websocket Terminal",
			Group:       "System Tools",
			IconPath:    "wstty/img/small_icon.png",
			Version:     "1.1",
			StartDir:    "wstty/console.html",
			SupportFW:   true,
			LaunchFWDir: "wstty/console.html",
			InitFWSize:  []int{740, 500},
		})

	} else {
		//Start the aoModule pipeline (which will parse the flags as well). Pass in the module launch information
		handler = aroz.HandleFlagParse(aroz.ServiceInfo{
			Name:        "WsTTY",
			Desc:        "arozos Websocket Terminal",
			Group:       "System Tools",
			IconPath:    "img/icons/wstty/small_icon.png",
			Version:     "1.1",
			StartDir:    "wstty/",
			SupportFW:   true,
			LaunchFWDir: "wstty/",
			InitFWSize:  []int{740, 500},
		})

	}

	SetupCloseHandler()

	//Start the gotty and rproxy it to the main system

	if runtime.GOOS == "windows" {
		//Switch to using wsshell module
		terminal := wsshell.NewWebSocketShellTerminal()
		http.HandleFunc("/tty/", terminal.HandleOpen)

		//Register the standard web services urls
		fs := http.FileServer(http.Dir("./web"))
		http.Handle("/", fs)

		//Any log println will be shown in the core system via STDOUT redirection. But not STDIN.
		log.Println("WsTTY (Windows Compatible Mode) started. Listening on " + handler.Port)
		err := http.ListenAndServe(handler.Port, nil)
		if err != nil {
			log.Fatal(err)
		}
	} else if runtime.GOOS == "linux" {
		//Use wstty directly
		absolutePath := ""
		if runtime.GOARCH == "amd64" {
			abs, _ := filepath.Abs("./gotty/gotty_linux_amd64")
			absolutePath = abs
		} else if runtime.GOARCH == "arm" {
			abs, _ := filepath.Abs("./gotty/gotty_linux_arm")
			absolutePath = abs
		} else if runtime.GOARCH == "arm64" {
			abs, _ := filepath.Abs("./gotty/ggotty_linux_arm64")
			absolutePath = abs
		} else {
			//Unsupported platform. Default use amd64
			abs, _ := filepath.Abs("./gotty/gotty_linux_amd64")
			absolutePath = abs
		}
		//Extract port number from listening addr
		tmp := strings.Split(handler.Port, ":")
		tmp = tmp[len(tmp)-1:]
		portOnly := strings.Join(tmp, "")

		log.Println("WsTTY Started. Listening on ", portOnly)

		//Start the gotty. This shoud be blocking by itself
		cmd := exec.Command(absolutePath, "-w", "-p", portOnly, "-a", "localhost", "--ws-origin", `\w*`, "bash", "--init-file", "bashstart")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			//Fail to start gotty. Disable this module
			ioutil.WriteFile(".disabled", []byte(""), 0755)
			return
		}
	} else {
		panic("Not supported platform: " + runtime.GOOS)
	}

}
