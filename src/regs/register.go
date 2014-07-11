package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coreos/go-etcd/etcd"
	"github.com/fsouza/go-dockerclient"
)

type ServiceInfo struct {
	Ip    string
	Name  string
	Ports []docker.APIPort
}

type ContainerHandler struct {
	*Container
	*etcd.Client
	ttl uint64
}

func GetHandler(cname string, etcdHost string) (*ContainerHandler, error) {
	cont, err := ContainerByName(cname)
	if err != nil {
		return nil, err
	}

	machines := []string{etcdHost}
	fmt.Printf("machines %+v\n", machines)
	eclient := etcd.NewClient(machines)
	fmt.Printf("eclient %+v\n", eclient)
	handler := &ContainerHandler{cont, eclient, 300}
	return handler, nil

}

func (h *ContainerHandler) register() error {

	path := strings.Split(h.Container.Config.Image, "/")
	service := path[len(path)-1]
	key := fmt.Sprintf("/service/%s/%s", service, h.Container.ID)

	ip := h.Container.NetworkSettings.IPAddress
	name := h.Container.Name
	ports := h.Container.NetworkSettings.PortMappingAPI()
	val, err := json.Marshal(ServiceInfo{ip, name, ports})

	if err != nil {
		return err
	}

	resp, err := h.Set(key, string(val), h.ttl)

	if err != nil {
		return err
	}

	fmt.Printf("resp %+v\n", resp)

	return nil
}
