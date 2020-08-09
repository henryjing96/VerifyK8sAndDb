package main

import (
	checkOperaiton "checkResource/checkOperation"
	"io"
	"net/http"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	responseMsg := checkOperaiton.DoCheck()
	io.WriteString(w, string(responseMsg))
}

func main() {
	http.HandleFunc("/checkResource", indexHandler)
	http.ListenAndServe(":44106", nil)
}

/*
package main

import (
    "fmt"
    "net/http"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "hello world")
}

func main() {
    http.HandleFunc("/", indexHandler)
    http.ListenAndServe(":8000", nil)
}
*/
