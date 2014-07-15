package checker

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fsouza/go-dockerclient"
)

// Default check period
const DefaultTTL = 30

var endpoint = "unix:///var/run/docker.sock"

// NotSupportCheckError is error which tell us that analyzed container doesn't contain enviroment variable HEALTHCHECK in his Dockerfile
type NotSupportCheckError struct {
	*Container
}

func (e NotSupportCheckError) Error() string {
	return fmt.Sprintf("Container %s doesn't provide method to check health", e.Container)
}

// Container supported healthcheck
type Container struct {
	*docker.Client
	*docker.Container
	healthcheck    string
	healthcheckttl time.Duration
}

func (c *Container) String() string {
	return fmt.Sprintf("%s: %s", strings.TrimLeft(c.Container.Name, "/"), c.Container.ID[:16])
}

// ContainerByID return Container by his id
func ContainerByID(cid string) (*Container, error) {
	client, err := docker.NewClient(endpoint)
	if err != nil {
		return nil, err
	}
	cont, err := client.InspectContainer(cid)
	logger.Debug("Found container %s", cont.Name)
	if err != nil {
		return nil, err
	}
	container := new(Container)
	container.Client = client
	container.Container = cont
	container.healthcheckttl = DefaultTTL

	if hchk, ok := findVariable("HEALTHCHECK", cont.Config.Env); ok {
		logger.Debug("HEALTHCHECK = %s ", hchk)
		container.healthcheck = hchk
	}

	if value, ok := findVariable("HEALTHCHECKTTL", cont.Config.Env); ok {
		ttl, err := strconv.Atoi(value)
		if err != nil {
			logger.Warning("Wrong health ttl %s: use default %s\n", ttl, DefaultTTL)
		} else {
			logger.Debug("HEALTHCHECKTTL = %s", ttl)
			container.healthcheckttl = time.Duration(ttl)
		}
	}

	return container, nil
}

//ContainerByName find container by name
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
				return ContainerByID(c.ID)
			}
		}
	}
	return nil, fmt.Errorf("container %s not found", name)
}

//Check if container is alive and fuction properly
func (c *Container) Check() error {
	logger.Debug("Check container %s", c)
	cont, err := c.Client.InspectContainer(c.ID)
	if err != nil {
		return err
	}
	c.Container = cont
	if !cont.State.Running {
		return fmt.Errorf("container %s is not running", c)
	}
	if c.healthcheck == "" {
		logger.Error("Container %s doesn't support check interface", c)
		return &NotSupportCheckError{c}
	}
	err = runContainer(c.Client, c.Config.Image, c.healthcheck, c.NetworkSettings.IPAddress, true)
	if err != nil {
		return err
	}
	logger.Debug("Container %s is ok", c.ID)
	return nil
}

func removeContainer(client *docker.Client, cid string) error {
	logger.Warning("REMOVE CONTAINER %s", cid)
	err := client.RemoveContainer(docker.RemoveContainerOptions{cid, true, true})
	if err != nil {
		return err
	}
	logger.Warning("DONE CONTAINER %s", cid)

	return nil
}

func runContainer(client *docker.Client, image string, command string, ip string, remove bool) error {
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

	if remove {
		logger.Warning("START CONTAINER %s", container.ID)
		//defer client.RemoveContainer(docker.RemoveContainerOptions{container.ID, true, true})
		defer removeContainer(client, container.ID)
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
		return fmt.Errorf("check return error code %d: %s", code, b.Bytes())
	}

	return nil
}
