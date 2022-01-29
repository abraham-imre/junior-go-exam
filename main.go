package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lucasjones/reggen"
	"gopkg.in/validator.v2"
)

type Venue struct {
	Name     string `json:"name" validate:"nonzero"`
	Location string `json:"location" validate:"nonzero"`
}

type Event struct {
	Id          string `json:"id"`
	Name        string `json:"name" validate:"min=10"`
	Venue       Venue  `json:"venue" validate:"nonzero"`
	Description string `json:"description" validate:"min=30"`
	Date        string `json:"date" validate:"regexp=[0-9]{4}-[0-1][0-9]-[0-3][0-9]T[0-5][0-9]:[05][0-9]:[05][0-9]Z"`
}

func listEvents(c *gin.Context) {
	//events variable to store all the entities that will be parsed from event-list.json
	var events []Event
	jsonFile, err := os.Open("./test/event-list.json")
	if err != nil {
		log.Println(err)
	}
	//reading bytes from file
	byteValue, _ := ioutil.ReadAll(jsonFile)
	//bytevalue parsing into events var
	json.Unmarshal(byteValue, &events)
	log.Println("Successful data parse")

	//returning jsonvalue of events var
	c.Writer.Header().Set("Content-Type", "application/json")
	log.Println("Returning events")
	c.JSON(http.StatusOK, events)
}

func saveEvent(c *gin.Context) {
	var event Event
	//Data binding to model
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	}

	//Validate data given in request
	if err := validator.Validate(event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
	}

	//Generate id in the vains of already existing records (should implement a check if it's unique, highly improbable tho)
	idStr, err := reggen.Generate("[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}", 1)
	if err != nil {
		log.Println(err)
	}
	log.Println("Generated ID")
	event.Id = idStr
	c.Writer.Header().Set("Content-Type", "application/json")
	log.Println("Returning value of event after ID assign")

	c.JSON(http.StatusCreated, event)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := gin.Default()

	// Creating /api route prefix group
	ag := router.Group("/api")
	{
		// Registering list events handler
		ag.GET("/events", listEvents)

		// Registering event save handler
		ag.PUT("/events", saveEvent)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%v", port),
		Handler: router,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Printf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)

	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
