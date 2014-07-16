package checker

import (
	"fmt"
	"sync"

	"github.com/fsouza/go-dockerclient"
)

//Listener of docker events which start monitoring on every started container
type Listener struct {
	monitors map[string]*ContainerMonitor
	client   *docker.Client
	etcdHost string
	stop     chan bool
	wg       sync.WaitGroup
}

//GetListener return new Listener
func GetListener(etcdHost string) (*Listener, error) {
	client, err := docker.NewClient(endpoint)
	if err != nil {
		return nil, err
	}
	listener := new(Listener)
	listener.monitors = make(map[string]*ContainerMonitor)
	listener.client = client
	listener.etcdHost = etcdHost
	listener.stop = make(chan bool, 1)
	return listener, nil
}

//Run process of monitoring docker events
func (l *Listener) Run() error {
	err := l.startExisted()
	if err != nil {
		return err
	}
	err = l.listen()
	return err
}

//Stop monitor containers
func (l *Listener) Stop() {
	logger.Info("Stop listening")
	l.stop <- true
}

func (l *Listener) startExisted() error {
	logger.Debug("Start monitoring already existed containers")
	cfg := docker.ListContainersOptions{}
	cfg.All = true
	containers, err := l.client.ListContainers(cfg)
	if err != nil {
		return err
	}
	for _, c := range containers {
		mon, err := l.addMonitor(c.ID)
		if err != nil {
			return fmt.Errorf("cannot create monitor for container %s: %s", c, err)
		}
		l.wg.Add(1)
		go func() {
			defer l.wg.Done()
			mon.StartMonitoring()
		}()
	}
	return nil
}

func (l *Listener) addMonitor(cid string) (*ContainerMonitor, error) {
	cont, err := ContainerByID(cid)
	if err != nil {
		return nil, err
	}
	monitor, err := GetMonitor(cont, l.etcdHost)
	if err != nil {
		return nil, err
	}
	l.monitors[monitor.ID] = monitor
	return monitor, nil
}

func (l *Listener) listen() error {
	events := make(chan *docker.APIEvents)
	l.client.AddEventListener(events)

	for {
		select {
		case event := <-events:
			switch event.Status {
			case "start":
				monitor, err := l.addMonitor(event.ID)
				if err != nil {
					logger.Error("Got error when try start monitor container %s: %s", event.ID, err)
				}
				l.wg.Add(1)
				go func() {
					defer l.wg.Done()
					monitor.StartMonitoring()
				}()
			case "stop":
				if monitor, ok := l.monitors[event.ID]; ok {
					monitor.StopMonitoring()
					delete(l.monitors, event.ID)
				}
			}
		case <-l.stop:
			for _, monitor := range l.monitors {
				monitor.StopMonitoring()
				events = nil
			}
			l.wg.Wait()
			return nil
		}
	}
	return nil
}
