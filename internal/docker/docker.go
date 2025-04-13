package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
)

type Client interface {
	ContainerList(context.Context, container.ListOptions) ([]container.Summary, error)
}

func GetRoutes(client Client, hostLabel string) (map[string]string, error) {
	containers, err := client.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing containers: %w", err)
	}

	routes := make(map[string]string)

	for _, c := range containers {
		hostLabel := c.Labels[hostLabel]
		if hostLabel == "" {
			continue
		}

		for _, net := range c.NetworkSettings.Networks {
			routes[strings.ToLower(hostLabel)] = net.IPAddress
			break
		}
	}

	return routes, nil
}
