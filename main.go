package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

var (
	restartTimeout int
	cooldownSec    int
)

// sendAppriseMessage executes the apprise command.
func sendAppriseMessage(body string) {
	cmd := exec.Command("apprise", "-t", "Docker Watchdog", "-b", body)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to send notification: %v. Output: %s", err, out)
	}
}

func main() {
	// Config via flags
	flag.IntVar(&restartTimeout, "timeout", 10, "Container restart timeout in seconds")
	flag.IntVar(&cooldownSec, "cooldown", 30, "Pause between restarts per container (seconds)")
	flag.Parse()

	ctx := context.Background()

	// Create a new Docker client
	cli, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		logMessage := fmt.Sprintf("Failed to create Docker client: %v", err)
		sendAppriseMessage(logMessage)
		log.Fatalf(logMessage)
	}
	logMessage := fmt.Sprintf("Successfully connected to Docker daemon.")
	log.Println(logMessage)

	// Send notification
	sendAppriseMessage(logMessage)

	// Filter for container health status events
	eventFilters := filters.NewArgs()
	eventFilters.Add("type", "container")
	eventFilters.Add("event", "health_status")

	// Get the event stream
	eventChan, errChan := cli.Events(ctx, types.EventsOptions{Filters: eventFilters})

	// Track last restart times to avoid rapid loops
	lastRestart := make(map[string]time.Time)

	// Main loop to read events from the stream
	for {
		select {
		case event := <-eventChan:
			if event.Action != "health_status: unhealthy" {
				continue
			}

			// Apply cooldown
			if time.Since(lastRestart[event.Actor.ID]) < time.Duration(cooldownSec)*time.Second {
				continue
			}
			lastRestart[event.Actor.ID] = time.Now()

			containerName := strings.TrimPrefix(event.Actor.Attributes["name"], "/")
			containerID := event.Actor.ID[:12]

			logMessage := fmt.Sprintf("Unhealthy container detected: %s (%s)", containerName, containerID)
			log.Printf(logMessage)
			sendAppriseMessage(logMessage)

			stopOpts := container.StopOptions{Timeout: &restartTimeout}

			if err := cli.ContainerRestart(ctx, event.Actor.ID, stopOpts); err != nil {
				logMessage = fmt.Sprintf("Failed to restart container %s: %v", containerName, err)
				log.Printf(logMessage)
				sendAppriseMessage(logMessage)
			}

			logMessage = fmt.Sprintf("Restarted: %s (%s)", containerName, containerID)
			log.Printf(logMessage)
			sendAppriseMessage(logMessage)

		case err := <-errChan:
			log.Printf("Error from Docker event stream: %v", err)
			log.Println("Attempting to re-establish connection...")
			time.Sleep(5 * time.Second)
			newCli, newErr := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
			if newErr != nil {
				log.Fatalf("Failed to re-create Docker client: %v", newErr)
			}
			log.Println("Successfully re-established connection to Docker daemon.")
			cli = newCli
			eventChan, errChan = cli.Events(ctx, types.EventsOptions{Filters: eventFilters})
		}
	}
}
