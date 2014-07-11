package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

const DEFAULT_TTL int = 30

type Container struct {
	*docker.Client
	*docker.Container
	healthcheck    string
	healthcheckttl int
}

func ContainerByName(name string) (*Container, error) {
	client, _ := docker.NewClient(endpoint)
	cfg := docker.ListContainersOptions{}
	containers, err := client.ListContainers(cfg)
	if err != nil {
		return nil, err
	}
	for _, c := range containers {
		for _, n := range c.Names {
			if strings.TrimLeft(n, "/") == name {
				cont, err := client.InspectContainer(c.ID)
				fmt.Println("Found container ", cont.Name)
				if err != nil {
					return nil, err
				}
				container := new(Container)
				container.Client = client
				container.Container = cont
				container.healthcheckttl = DEFAULT_TTL
				for _, env := range cont.Config.Env {
					vals := strings.Split(env, "=")
					name, value := vals[0], vals[1]
					if name == "HEALTHCHECK" {
						fmt.Println("HEALTHCHECK ", value)
						container.healthcheck = value
					}
					if name == "HEALTHCHECKTTL" {
						fmt.Println("HEALTHCHECKTTL ", value)
						ttl, err := strconv.Atoi(value)
						if err != nil {
							fmt.Printf("Wrong health ttl %s \n", ttl)
						} else {
							container.healthcheckttl = ttl
						}

					}

				}
				return container, nil
			}
		}
	}
	return nil, fmt.Errorf("Container %s not found", name)
}

func (c *Container) check() error {
	if c.healthcheck == "" {
		return fmt.Errorf("Container %s doesn't support check interface", c.Name)
	}
	err := runContainer(c.Client, c.Config.Image, c.healthcheck, c.NetworkSettings.IPAddress)
	if err != nil {
		return err
	}
	return nil
}

func runContainer(client *docker.Client, image string, command string, ip string) error {
	fmt.Printf("Run image %s with com line %s to test %s\n", image, command, ip)
	config := new(docker.Config)
	config.Image = image
	config.Entrypoint = []string{command}
	config.Cmd = []string{ip}

	options := docker.CreateContainerOptions{Config: config}
	container, err := client.CreateContainer(options)
	if err != nil {
		return err
	}

	hostConfig := new(docker.HostConfig)
	err = client.StartContainer(container.ID, hostConfig)
	if err != nil {
		return err
	}

	code, err := client.WaitContainer(container.ID)
	if err != nil {
		return err
	}

	if code != 0 {
		return fmt.Errorf("Check return error code %d", code)
	}

	return nil
}
