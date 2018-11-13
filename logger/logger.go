package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type elk struct {
	Environment string `json:"environment"`
	Release     string `json:"release"`
}

type elkIndex struct {
	elk
	Timestamp string      `json:"@timestamp"`
	RequestID string      `json:"requestID"`
	Level     string      `json:"level"`
	Message   string      `json:"message"`
	Args      interface{} `json:"args"`
}

var requestID = ""
var elkBase = elk{}

func getRequestID() string {
	if requestID == "" {
		newRequestID, err := exec.Command("uuidgen").Output()
		if err != nil {
			log.Fatal(err)
		}

		requestID = string(newRequestID)
	}

	return requestID
}

func createInstance() {

	env := os.Getenv("environment")

	if env == "" {
		env = "local"
	}

	release := os.Getenv("release")

	if release == "" {
		release = "1.0.0"
	}

	elkBase = elk{
		Environment: env,
		Release:     release,
	}
}

func init() {
	createInstance()
}

func getHost(date string) string {

	host := os.Getenv("elk_host")

	if host == "" || !strings.Contains(host, "http") {
		host = "http://localhost:9200"
	}
	index := os.Getenv("elk_index")

	if index == "" {
		index = "logger"
	}
	return fmt.Sprintf("%s/%s-%s-%s/logs", host, index, elkBase.Environment, date)
}

func logger(level, message string, args interface{}) {

	now := time.Now().UTC()
	index := elkIndex{
		Timestamp: now.Format(time.RFC3339),
		RequestID: getRequestID(),
		Level:     level,
		Message:   message,
		Args:      args,
	}
	index.Environment = elkBase.Environment
	index.Release = elkBase.Release

	bytesRepresentation, err := json.Marshal(index)
	if err != nil {
		log.Fatalln(err)
	}

	resp, err := http.Post(getHost(now.Format("2006-01-02")), "application/json", bytes.NewBuffer(bytesRepresentation))
	if err != nil {
		log.Fatalln(err)
	}

	var result map[string]interface{}

	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != 201 {
		log.Printf("Error send logs to elasticsearch")
	}
}

// Info sends to the elastic search the INFO type logs
func Info(message string, args interface{}) {
	go logger("INFO", message, args)
}

// Error sends to the elastic search the ERROR type logs
func Error(message string, args interface{}) {
	go logger("ERROR", message, args)
}
