package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	if err := gin.New().Run(); err != nil {
		log.Fatal(err)
	}
}
