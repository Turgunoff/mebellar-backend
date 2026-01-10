package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"mebellar-backend/handlers" // Handlerlarni ulaymiz

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "mebel_user"
	password = "MebelStrong2024!"
	dbname   = "mebellar_olami"
)

func main() {
	// 1. Bazaga ulanish
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Baza ulangan!")

	// 2. Marshrutlar (Routes)
	// /api/products ga murojaat bo'lsa, GetProducts ishlaydi
	http.HandleFunc("/api/products", handlers.GetProducts(db))

	// 3. Serverni yoqish
	fmt.Println("🚀 Server 8081-portda ishlayapti...")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
