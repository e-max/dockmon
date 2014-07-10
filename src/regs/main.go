package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/coreos/go-etcd/etcd"
	"github.com/fsouza/go-dockerclient"
)

var (
	etcdHost = "localhost"
	endpoint = "unix:///var/run/docker.sock"
)

type Container struct {
	*docker.Client
	c *docker.Container
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
				fmt.Println("Found container ", c.Names)
				cont, err := client.InspectContainer(c.ID)
				if err != nil {
					return nil, err
				}

				return &Container{client, cont}, nil
			}
		}
	}
	return nil, fmt.Errorf("Container %s not found", name)
}

func (cont *Container) check() error {
	info, err := cont.InspectContainer(cont.c.ID)
	if err != nil {
		return err
	}
	envs := info.Config.Env
	image := info.Image
	fmt.Printf("image %+v\n", image)
	for _, env := range envs {
		vals := strings.Split(env, "=")
		name, value := vals[0], vals[1]
		if name == "HEALTHCHECK" {
			fmt.Println("HEALTHCHECK ", value)
			err := runContainer(cont.Client, image, value, info.NetworkSettings.IPAddress)
			fmt.Printf("result %+v\n", err)
		}
	}
	return nil
}

func getLinkedContainers() ([]string, error) {
	names := []string{}

	for _, e := range os.Environ() {
		vals := strings.Split(e, "=")
		if len(vals) != 2 {
			return nil, fmt.Errorf("Wrong options %s", e)
		}
		k := strings.ToUpper(vals[0])
		v := vals[1]
		if strings.HasSuffix(k, "_NAME") {
			//names = append(names, strings.TrimSuffix(k, "_NAME"))
			names = append(names, v)
		}
	}
	return names, nil
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

type ContainerHandler struct {
	Container
	etcdb *etcd.Client
}

func (h *ContainerHandler) register() error {
	return nil
}

func getEnvVariable(name string) (string, error) {
	for _, e := range os.Environ() {
		vals := strings.Split(e, "=")
		if len(vals) != 2 {
			return "", fmt.Errorf("Wrong options %s", e)
		}
		if strings.ToUpper(vals[0]) == strings.ToUpper(name) {
			return vals[1], nil
		}
	}
	return "", fmt.Errorf("Variable %s doesn't exist in enviroment")
}

func _checkLinked() error {
	fmt.Println("==============================================")
	names, err := getLinkedContainers()
	if err != nil {
		return err
	}

	if len(names) == 0 {
		return fmt.Errorf("No linked containers")
	}

	for _, name := range names {
		cont, err := ContainerByName(name)
		if err != nil {
			return err
		}
		err = cont.check()
		if err != nil {
			return err
		}

	}

	return nil

}

func checkByName(cname string) error {
	cont, err := ContainerByName(cname)
	if err != nil {
		return err
	}
	err = cont.check()
	if err != nil {
		return err
	}
	return nil
}

func main() {
	flag.StringVar(&etcdHost, "etcd-host", "localhost", "host where etcd is listenting")
	flag.Parse()
	//err := _checkLinked()
	cname := flag.Arg(0)
	fmt.Printf("cname %+v\n", cname)
	fmt.Printf("os.Args %+v\n", os.Args)
	err := checkByName(cname)
	if err != nil {
		fmt.Println("Error ", err)
	}

}
