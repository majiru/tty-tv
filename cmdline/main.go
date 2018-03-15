package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"
)

var (
	flags         = flag.NewFlagSet("", flag.ContinueOnError)
	stderr        = flags.Bool("e", true, "redirect stderr")
	buffer_length = 1024
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
	cmd.Stdout = os.Stdout
	if *stderr {
		cmd.Stderr = os.Stdout
	}

	//Returns io.ReadCloser stdout pipe
	stdoutPipe, err := cmd.StdoutPipe()

	if err != nil {
		log.Fatal(err)
	}

	cmd.Run()

	buffer := make([]byte, buffer_length)
	for {
		n, _ := stdoutPipe.Read(buffer)
		data := buffer[0:n]

		for _, item := range data {
			c <- item
		}

		for i := 0; i < n; i++ {
			buffer[i] = 0
		}
	}

}

func bufferShellOutput(w http.ResponseWriter, c chan byte) {
	for {
		select {
		case data := <-c:
			w.Write(tmp)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

func main() {
	ch := make(chan byte, buffer_length)
	screen(ch)
	http.HandleFunc("/api/screen", func(w http.ResponseWriter, r *http.Request) {
		bufferShellOutput(w, ch)
	})

	// serve static files on `/`
	http.Handle("/", http.FileServer(http.Dir("./static")))

	log.Printf("serving on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
