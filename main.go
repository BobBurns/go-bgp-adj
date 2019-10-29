package main

import (
	"bytes"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
	"time"
)

// http handlers
func main() {
	funcMap := template.FuncMap{
		"alerts":   alerts,
		"ctime":    ctime,
		"totpeers": totpeers,
	}

	// parse html template
	t = template.Must(template.New("templates").Funcs(funcMap).ParseFiles("html/templates/home2.html"))

	// populate first array, then update in background
	fmt.Println("please wait for initial query to complete...")
	nqs := Query()
	results = nqs
	// start go routine concurrently
	fmt.Println("ok.")
	go func() {
		for {

			time.Sleep(61 * time.Second)
			nqs := Query()
			results = nqs
			nqs = nil
		}
	}()

	router := mux.NewRouter()
	/* change this to IP addr !! */
	sub := router.Host("localhost").Subrouter()
	sub.PathPrefix("/html/").Handler(http.StripPrefix("/html/", http.FileServer(http.Dir("html"))))
	sub.HandleFunc("/", queryHandler)

	// IdleTimeout requires go1.8
	server := http.Server{
		Addr:         ":8082",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      router,
	}
	fmt.Println("Server started at localhost:8082")
	log.Fatal(server.ListenAndServe())

}

//template functions
func ctime() string {
	return time.Now().Format(time.RFC822)
}
func alerts() string {
	return fmt.Sprintf("%d", alertq)
}
func totpeers() string {
	return fmt.Sprintf("%d", len(*results))
}

func queryHandler(w http.ResponseWriter, r *http.Request) {
	var b bytes.Buffer

	h := "home2.html"

	// results from query.go
	// results array is poplulated from separate thread
	err := t.ExecuteTemplate(&b, h, results)
	if err != nil {
		fmt.Fprintf(w, "Error with template: %s ", err)
		return
	}
	b.WriteTo(w)

}
