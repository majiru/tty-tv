package main

import (
	"flag"
	"github.com/kr/pty"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

var (
	flags                = flag.NewFlagSet("", flag.ContinueOnError)
	stderr               = flags.Bool("e", true, "redirect stderr")
	maxReadBufferLength  = 64 * 1024 //max amount of characters to take in per poll
	minWriteBufferLength = 20        //min amount of characters per each update of the web socket
)

func init() {
	flags.Parse(os.Args[1:])
	if len(flags.Args()) < 1 {
		log.Fatal("command missing")
	}

	//Try and set the buffer to two times the size of one whole screen of text
	width, height, err := terminal.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		maxReadBufferLength = width * height * 2
	}
}

func screen(c chan byte) {
	cmd := exec.Command(flags.Args()[0], flags.Args()[1:]...)

	//Pass the command into a new tty
	ptmx, _ := pty.Start(cmd)
	defer func() { _ = ptmx.Close() }()

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

	defer func() { _ = terminal.Restore(int(os.Stdin.Fd()), oldState) }()
	go func() { _, _ = io.Copy(ptmx, os.Stdin) }()

	//Read stdout and send it to the web socket
	buffer := make([]byte, maxReadBufferLength)
	for {
		n, _ := ptmx.Read(buffer)
		data := buffer[0:n]
		os.Stdout.Write(data) //Let the user see what's going on

		for _, item := range data {
			c <- item
		}

		//Reset the buffer
		for i := 0; i < n; i++ {
			buffer[i] = 0
		}
	}

}

func bufferShellOutput(w http.ResponseWriter, c chan byte) {
	i := 0 //Lazy way of buffering
	buffer := make([]byte, minWriteBufferLength)
	for {
		select {
		case data := <-c:
			buffer[i] = data
			i++
			if i == minWriteBufferLength-1 {
				w.Write(buffer)
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
				i = 0
			}
		}
	}
}

func main() {
	ch := make(chan byte, maxReadBufferLength)

	go screen(ch)
	http.HandleFunc("/api/screen", func(w http.ResponseWriter, r *http.Request) {
		bufferShellOutput(w, ch)
	})

	// serve static files on `/`
	http.Handle("/", http.FileServer(http.Dir("./static")))

	log.Printf("serving on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
