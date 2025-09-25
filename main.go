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

// runAppriseCommand executes the apprise command.
func runAppriseCommand(title, body string) {
	cmd := exec.Command("apprise", "-t", title, "-b", body)
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

func main() {
	// Config via flags
	flag.IntVar(&restartTimeout, "timeout", 10, "Restart timeout in seconds")
	flag.IntVar(&cooldownSec, "cooldown", 30, "Cooldown between restarts per container (seconds)")
	flag.Parse()

	ctx := context.Background()

	// Create a new Docker client
	cli, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}

	// Send "service started" notification
	runAppriseCommand("Docker Watchdog", "Service started")
	log.Printf("Running")

	// Filter for container health status events
	filters := filters.NewArgs()
	filters.Add("type", "container")
	filters.Add("event", "health_status")

	// Get the event stream
	eventChan, errChan := cli.Events(ctx, types.EventsOptions{Filters: filters})

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
			containerID := event.ID[:12]

			logMessage := fmt.Sprintf("Unhealthy container detected: %s (%s)", containerName, containerID)
			log.Printf(logMessage)

			timeoutSec := restartTimeout
			stopOpts := container.StopOptions{Timeout: &timeoutSec}

			log.Printf("Restarting container: %s", containerName)
			if err := cli.ContainerRestart(ctx, event.Actor.ID, stopOpts); err != nil {
				log.Printf("Failed to restart container %s: %v", containerName, err)
			}

			appriseMessage := fmt.Sprintf("Restarted: %s", containerName)
			runAppriseCommand("Docker Watchdog", appriseMessage)

		case err := <-errChan:
			log.Printf("Error from Docker event stream: %v", err)
			log.Println("Attempting to re-establish connection...")
			time.Sleep(5 * time.Second)
			newCli, newErr := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
			if newErr != nil {
				log.Fatalf("Failed to re-create Docker client: %v", newErr)
			}
			cli = newCli
			eventChan, errChan = cli.Events(ctx, types.EventsOptions{Filters: filters})
		}
	}
}
