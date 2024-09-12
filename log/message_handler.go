package log

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/drycc/logger/storage"
)

const (
	podPattern              = `(\w.*)-(\w.*)-(\w.*)-(\w.*)`
	controllerPattern       = `^(INFO|WARN|DEBUG|ERROR)\s+(\[(\S+)\])+:(.*)`
	controllerContainerName = "drycc-controller"
	timeFormat              = "2006-01-02T15:04:05-07:00"
)

var (
	controllerRegex = regexp.MustCompile(controllerPattern)
)

func handle(rawMessage []byte, storageAdapter storage.Adapter) error {
	message := new(Message)
	if err := json.Unmarshal(rawMessage, message); err != nil {
		return err
	}
	if fromController(message) {
		if err := storageAdapter.Write(getApplicationFromControllerMessage(message), buildControllerLogMessage(message)); err != nil {
			fmt.Printf("storage message error, %v, %v", err, storageAdapter)
		}

	}
	return nil
}

func fromController(message *Message) bool {
	matched, _ := regexp.MatchString(controllerContainerName, message.Kubernetes.ContainerName)
	return matched
}

func getApplicationFromControllerMessage(message *Message) string {
	return controllerRegex.FindStringSubmatch(message.Log)[3]
}

func buildControllerLogMessage(message *Message) string {
	l := controllerRegex.FindStringSubmatch(message.Log)
	return fmt.Sprintf("%s drycc[controller]: %s %s", message.Time.Format(timeFormat), l[1], strings.Trim(l[4], " "))
}
