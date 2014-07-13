package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	ttl    uint64
	ticker *time.Ticker
	stop   chan bool
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
	handler := &ContainerHandler{cont, eclient, 300, nil, nil}
	return handler, nil

}

func (h *ContainerHandler) startMonitoring() {
	h.ticker = time.NewTicker(h.healthcheckttl*time.Second - 1)
	h.stop = make(chan bool, 1)
	for {
		select {
		case <-h.ticker.C:
			err := h.check()
			if err != nil {
				if noCont, ok := err.(*docker.NoSuchContainer); ok {
					h.stopMonitoring()
					fmt.Printf("ERROR 2 %+v\n", noCont)
				}
				fmt.Printf("GOT ERROR  %#v\n", err)
			} else {
				h.register()
			}
		case <-h.stop:
			fmt.Println("GOT IN STOP")
			return
		}
	}
	fmt.Println("EXIT ")
}

func (h *ContainerHandler) stopMonitoring() {
	if h.ticker != nil {
		h.ticker.Stop()
		h.stop <- true
	}
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
