package checker

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fsouza/go-dockerclient"
)

const DEFAULT_TTL = 30

var endpoint = "unix:///var/run/docker.sock"

type Container struct {
	*docker.Client
	*docker.Container
	healthcheck    string
	healthcheckttl time.Duration
}

func ContainerById(cid string) (*Container, error) {
	client, err := docker.NewClient(endpoint)
	if err != nil {
		return nil, err
	}
	cont, err := client.InspectContainer(cid)
	logger.Debug("Found container ", cont.Name)
	if err != nil {
		return nil, err
	}
	container := new(Container)
	container.Client = client
	container.Container = cont
	container.healthcheckttl = DEFAULT_TTL

	if hchk, ok := findVariable("HEALTHCHECK", cont.Config.Env); ok {
		logger.Debug("HEALTHCHECK ", hchk)
		container.healthcheck = hchk
	}

	if value, ok := findVariable("HEALTHCHECKTTL", cont.Config.Env); ok {
		ttl, err := strconv.Atoi(value)
		if err != nil {
			logger.Warning("Wrong health ttl %s: use default %s\n", ttl, DEFAULT_TTL)
		} else {
			container.healthcheckttl = time.Duration(ttl)
		}
	}

	return container, nil
}

//Get container by name
func ContainerByName(name string) (*Container, error) {
	client, err := docker.NewClient(endpoint)
	if err != nil {
		return nil, err
	}

	cfg := docker.ListContainersOptions{}
	cfg.All = true
	containers, err := client.ListContainers(cfg)
	if err != nil {
		return nil, err
	}
	for _, c := range containers {
		for _, n := range c.Names {
			logger.Debug("Test name %s", n)
			if strings.TrimLeft(n, "/") == name {
				return ContainerById(c.ID)
			}
		}
	}
	return nil, fmt.Errorf("Container %s not found", name)
}

func (c *Container) Check() error {
	logger.Debug("Check container %s", c.ID)
	cont, err := c.Client.InspectContainer(c.ID)
	if err != nil {
		return err
	}
	c.Container = cont
	if !cont.State.Running {
		return fmt.Errorf("Container %s is not running", c.ID)
	}
	if c.healthcheck == "" {
		logger.Error("Container %s doesn't support check interface", c.Name)
		return fmt.Errorf("Container %s doesn't support check interface", c.Name)
	}
	err = runContainer(c.Client, c.Config.Image, c.healthcheck, c.NetworkSettings.IPAddress)
	if err != nil {
		return err
	}
	logger.Debug("Container %s is ok", c.ID)
	return nil
}

func runContainer(client *docker.Client, image string, command string, ip string) error {
	logger.Debug("Run image %s with com line %s to test %s\n", image, command, ip)
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
		var b bytes.Buffer
		options := docker.LogsOptions{
			container.ID,
			&b,
			false,
			true,
			true,
			true,
		}

		err = client.Logs(options)
		return fmt.Errorf("Check return error code %d: %s", code, b.Bytes())
	}

	return nil
}
