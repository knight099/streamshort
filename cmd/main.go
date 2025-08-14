package main

import (
	"fmt"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"gorm.io/gorm"

)

func handler(w http.ResponseWriter, r *http.Request){
	fmt.Fprintf("w, Server started")
}

func main() {
	http.HandleFunc("/", handler)

	fmt.Println("Server starting at 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Error starting server", err)
		
	}

}