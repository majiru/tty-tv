package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/gordonklaus/portaudio"
	"github.com/gorilla/websocket"
	"github.com/kr/pty"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	flags               = flag.NewFlagSet("", flag.ContinueOnError)
	stderr              = flags.Bool("e", true, "redirect stderr")
	maxReadBufferLength = 64 * 1024 //max amount of characters to take in per poll
	command             []string
	logFile             io.WriteCloser
)

func init() {
	// Set up logging output.
	// We include O_SYNC so that the logs are always written synchronously;
	// if the program crashes, no logging will be pending write.
	const logFileFlags = os.O_CREATE | os.O_WRONLY | os.O_APPEND | os.O_SYNC
	var err error
	logFile, err = os.OpenFile("tty-tv-debug.log", logFileFlags, 0755)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(logFile)

	flags.Parse(os.Args[1:])
	if len(flags.Args()) < 1 {
		shell := os.Getenv("SHELL")
		log.Printf("Command not found, defaulting to os shell (%s)", shell)
		command = append(command, shell, "-i")
	} else {
		command = append(command, flags.Args()[0:]...)
	}

	//Try and set the buffer to two times the size of one whole screen of text
	width, height, err := terminal.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		maxReadBufferLength = width * height * 2
	}
}

func screen(c chan []byte) {
	cmd := exec.Command(command[0], command[1:]...)

	//Pass the command into a new tty
	ptmx, _ := pty.Start(cmd)

	//Grab the parent terminal size
	sysch := make(chan os.Signal, syscall.SIGWINCH)
	signal.Notify(sysch, syscall.SIGWINCH)
	go func() {
		for range sysch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				log.Printf("Could not grab the size from the parent: %s", err)
			}
		}
	}()
	sysch <- syscall.SIGWINCH

	//Turn the input raw
	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatal(err)
	}

	go func() { _, _ = io.Copy(ptmx, os.Stdin) }()

	//Read stdout and send it to the web socket
	buffer := make([]byte, maxReadBufferLength)
	for {
		n, err := ptmx.Read(buffer)
		if err != nil {
			_ = terminal.Restore(int(os.Stdin.Fd()), oldState)
			ptmx.Close()
			os.Exit(0)
		}
		data := buffer[0:n]
		os.Stdout.Write(data) //Let the user see what's going on
		c <- data

	}

}

func writeToWebRaw(w http.ResponseWriter, c chan []byte) {
	for {
		data := <-c
		w.Write(data)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

}

func writeToWebSocket(w http.ResponseWriter, r *http.Request, c chan []byte) {
	upgrader := websocket.Upgrader{}
	u, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer u.Close()
	for {
		msg := <-c
		log.Printf("writing websocket message to client %v: %q", r.Host, msg)
		err = u.WriteMessage(websocket.BinaryMessage, msg)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func captureAudio(c chan []byte) {
	portaudio.Initialize()
	defer portaudio.Terminate()

	inputBuffer := make([]byte, 256)
	stream, err := portaudio.OpenDefaultStream(1, 0, 44100, len(inputBuffer), inputBuffer)
	if err != nil {
		log.Fatal(err)
	}

	defer stream.Close()

	for {
		stream.Start()
		stream.Read()
		c <- inputBuffer
	}
}

func checkForSocket(w http.ResponseWriter, r *http.Request, c chan []byte) {
	if _, ok := r.Header["Upgrade"]; ok {
		log.Printf("websocket connection from %v", r.Host)
		writeToWebSocket(w, r, c)
	} else {
		log.Printf("raw web connection from %v", r.Host)
		writeToWebRaw(w, c)
	}

}

func main() {
	defer logFile.Close()

	textChan := make(chan []byte, maxReadBufferLength)
	audioChan := make(chan []byte, 64)
	go screen(textChan)
	go captureAudio(audioChan)

	http.HandleFunc("/api/screen", func(w http.ResponseWriter, r *http.Request) {
		checkForSocket(w, r, textChan)
	})

	http.HandleFunc("/api/sound", func(w http.ResponseWriter, r *http.Request) {
		checkForSocket(w, r, audioChan)
	})

	// serve static files on `/`
	http.Handle("/", http.FileServer(http.Dir("./static")))

	fmt.Println("serving on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
