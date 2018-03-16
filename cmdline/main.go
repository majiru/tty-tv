package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"
)

var (
	flags        = flag.NewFlagSet("", flag.ContinueOnError)
	stderr       = flags.Bool("e", true, "redirect stderr")
	bufferLength = 1024 //Buffer size for stdout to websocket
)

func init() {
	flags.Parse(os.Args[1:])
	if len(flags.Args()) < 1 {
		log.Fatal("command missing")
	}
}

func screen(c chan byte) {
	cmd := exec.Command(flags.Args()[0], flags.Args()[1:]...)

	//Pretend we're a shell
	cmd.Stdin = os.Stdin
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	go cmd.Run()

	//Read stdout and send it to the web socket
	buffer := make([]byte, bufferLength)
	for {
		n, _ := stdoutPipe.Read(buffer)
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
	buffer := make([]byte, bufferLength)
	for {
		select {
		case data := <-c:
			buffer[i] = data
			i++
			if i == bufferLength {
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
	ch := make(chan byte, bufferLength)

	go screen(ch)
	http.HandleFunc("/api/screen", func(w http.ResponseWriter, r *http.Request) {
		bufferShellOutput(w, ch)
	})

	// serve static files on `/`
	http.Handle("/", http.FileServer(http.Dir("./static")))

	log.Printf("serving on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
