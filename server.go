package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatal("error parsing your template.")
	}

	t.Execute(w, nil)
	log.Println("Finished HTTP request at", r.URL.Path)
}

func main() {
	filePath := flag.String("f", "access.log", "File to be watched for changestream")
	serverPort := flag.String("p", "8000", "File to be watched for changestream")
	flag.Parse()

	b := NewBroker()
	b.start()

	fw, err := NewFileWatcher(*filePath)
	if err != nil {
		log.Fatalln("Err while creating file watcher", err.Error())
	}

	fw.Subscribe(b.messages)
	fw.MockLogger()

	http.Handle("/events/", b)

	http.Handle("/", http.HandlerFunc(handler))
	go func() {
		http.ListenAndServe(fmt.Sprintf(":%s", *serverPort), nil)
	}()

	log.Println("server started on port", *serverPort)

	ch := make(chan bool)
	<-ch
}
