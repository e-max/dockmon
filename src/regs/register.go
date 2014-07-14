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
	logger.Debug("Create etcd client connected to %+v", machines)
	eclient := etcd.NewClient(machines)
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
					logger.Debug("No such container %s", noCont)
					h.stopMonitoring()
				}
				logger.Info("Got error while check container %s: %s", h.Container.ID, err)
			} else {
				h.register()
			}
		case <-h.stop:
			return
		}
	}
}

func (h *ContainerHandler) stopMonitoring() {
	logger.Info("Stop monitorint container %s", h)
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

	logger.Debug("Register container %s with ip = %s", name, ip)

	val, err := json.Marshal(ServiceInfo{ip, name, ports})

	if err != nil {
		return err
	}

	_, err = h.Set(key, string(val), h.ttl)

	if err != nil {
		return err
	}

	return nil
}
