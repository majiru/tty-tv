package main

import (
	"log"
	"net/http"
        "os"
        "os/exec"
        "flag"
        "io"
)

var (
    flags  = flag.NewFlagSet("", flag.ContinueOnError)
    stderr = flags.Bool("e", true, "redirect stderr")
    buffer_length = 1024
)

func init() {
    flags.Parse(os.Args[1:])
    if len(flags.Args()) < 1 {
        log.Fatal("command missing")
    }
}

func screen(){
    cmd := exec.Command(flags.Args()[0], flags.Args()[1:]...)

    //Pretend we're a shell
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    if *stderr {
        cmd.Stderr = os.Stdout
    }

    //Returns io.ReadCloser stdout pipe
    stdoutPipe, _ := cmd.StdoutPipe()

    cmd.Run()
}


func bufferShellOutput(w http.ResponseWriter, reader io.ReadCloser){
    buffer := make([]byte, buffer_length)
    for {

        n, err := reader.Read(buffer)
        if err != nil {
            reader.Close();
            break
        }

        data := buffer[0:n]
        w.Write(data)
        if f, ok := w.(http.Flusher); ok {
            f.Flush()
        }

        for i := 0; i < n; i++{
            buffer[i] = 0
        }

    }


}

func main() {

	http.HandleFunc("/api/screen", func(w http.ResponseWriter, r *http.Request) {
            fmt.Fprintf(w, "%s", "Nothing here")
	})

	// serve static files on `/`
	http.Handle("/", http.FileServer(http.Dir("./static")))

	log.Printf("serving on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
