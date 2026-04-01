package main

import (
	"appointment-service/internal/app"
)

func main() {
	app.Run("8081", "http://localhost:8080")
}
