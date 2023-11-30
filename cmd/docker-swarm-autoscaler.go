package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"slices"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/luizfonseca/docker-swarm-autoscaler/internal/config"
)

var appConfig = config.Config{
	Interval: 5 * time.Second,
	Services: []config.ConfigService{
		{
			Name:        "traefik",
			Enabled:     true,
			StackName:   "olc",
			MaxReplicas: 3,
			Thresholds: config.ConfigServiceThreshold{
				Cpu: config.ServiceThreshold{
					Percent:           0.2,
					Metric:            "average",
					ScaleUpDuration:   "10s",
					ScaleDownDuration: "10s",
				},
			},
		},
		{
			Name:        "grafana",
			Enabled:     true,
			StackName:   "olc",
			MaxReplicas: 2,
			Thresholds: config.ConfigServiceThreshold{
				Cpu: config.ServiceThreshold{
					Percent:           0.2,
					Metric:            "average",
					ScaleUpDuration:   "10s",
					ScaleDownDuration: "10s",
				},
			},
		},
	},
}

type StatsChanInput struct {
	ServiceName string
	StatsIo     io.ReadCloser
}

func main() {
	log.Print("Starting...")
	log.Printf("Monitoring %d services", len(appConfig.Services))

	var ignoredServicesIds []string
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create the dockerClient
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatalf("Error creating docker client: %s", err)
	}
	defer dockerClient.Close()

	// Create ticker channel
	ticker := time.NewTicker(appConfig.Interval)
	defer ticker.Stop()

	statsChannel := make(chan StatsChanInput, 1)
	defer close(statsChannel)

	// Go func that receives stats
	go func() {
		for msg := range statsChannel {

			var statsResult types.Stats
			err := json.NewDecoder(msg.StatsIo).Decode(&statsResult)
			if err != nil {
				log.Printf("Failed to parse container stats for  service '%s' %s", msg.ServiceName, err)
				return
			}

			log.Printf("Received CPU stats for container %s: %.2f%%", msg.ServiceName, calculateCPUPercent(statsResult.PreCPUStats, &statsResult))
		}
	}()

	// Every X duration, collect metrics from each service defined in the configuration
	for d := range ticker.C {
		log.Printf("Metrics inspect at %s", d)

		for _, svc := range appConfig.Services {

			go func(configSvc config.ConfigService) {

				var defaultFilters []filters.KeyValuePair

				swarmName := configSvc.Name
				// Check if we need to generate a service name matching the stack
				if configSvc.StackName != "" {
					swarmName = configSvc.StackName + "_" + swarmName
					defaultFilters = append(defaultFilters, filters.KeyValuePair{Key: "label", Value: "com.docker.stack.namespace=" + configSvc.StackName})
				}

				defaultFilters = append(defaultFilters, filters.KeyValuePair{Key: "name", Value: swarmName})

				// If we can't find the service, report and move on
				if slices.Contains[[]string, string](ignoredServicesIds, swarmName) {
					log.Print("Ignoring errored service: " + swarmName)
					return
				}

				dockerServices, err := dockerClient.ServiceList(ctx, types.ServiceListOptions{
					Status:  true,
					Filters: filters.NewArgs(defaultFilters...),
				})

				if err != nil || len(dockerServices) == 0 {
					log.Printf("Error getting services: %s", err)
					ignoredServicesIds = append(ignoredServicesIds, swarmName)

					return
				}

				for _, dockerSvc := range dockerServices {
					go func(service swarm.Service) {
						dc, err := dockerClient.ContainerList(ctx, types.ContainerListOptions{
							Filters: filters.NewArgs(
								filters.KeyValuePair{Key: "label", Value: "com.docker.swarm.service.name=" + service.Spec.Name},
							),
						})

						if err != nil || len(dc) == 0 {
							log.Printf("Could not find containers for service '%s': %s", service.Spec.Name, err)
							return
						}

						// Skip checks if no containers can be found for service
						if len(dc) == 0 {
							return
						}

						for _, container := range dc {
							stats, err := dockerClient.ContainerStats(ctx, container.ID, false)
							if err != nil {
								log.Printf("Error getting container stats: %s", err)
								return
							}

							statsChannel <- StatsChanInput{
								ServiceName: service.Spec.Name,
								StatsIo:     stats.Body,
							}
						}

					}(dockerSvc)
				}

			}(svc)

		}

	}
}

// func collectStats(svc config.ConfigService) {

// }

func calculateCPUPercent(prev types.CPUStats, v *types.Stats) float64 {
	// default
	cpuPercent := 0.0
	// calculate the change for the cpu usage of the container in between readings
	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage) - float64(prev.CPUUsage.TotalUsage)
	// calculate the change for the entire system between readings
	systemDelta := float64(v.CPUStats.SystemUsage) - float64(prev.SystemUsage)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(v.CPUStats.OnlineCPUs) * 100.0
	}
	return cpuPercent
}
