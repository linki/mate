package consumers

import (
	"fmt"
	"sync"

	log "github.com/Sirupsen/logrus"

	"github.com/zalando-incubator/mate/interfaces"
	"github.com/zalando-incubator/mate/pkg"
)

// new StreamWriterProvider io.Writer instead of stdout
// or new file writrer better
type stdoutConsumer struct{}

func NewStdout() (interfaces.Consumer, error) {
	return &stdoutConsumer{}, nil
}

func value(ep *pkg.Endpoint) string {
	return fmt.Sprintf("%s - %s", ep.IP, ep.Hostname)
}

func (d *stdoutConsumer) Sync(endpoints []*pkg.Endpoint) error {
	for _, e := range endpoints {
		fmt.Println("sync record:", e.DNSName, value(e))
	}

	return nil
}

func (d *stdoutConsumer) Consume(endpoints <-chan *pkg.Endpoint, errors chan<- error, done <-chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	log.Infoln("[Stdout] Listening for events...")

	for {
		select {
		case e, ok := <-endpoints:
			if !ok {
				log.Info("[Stdout] channel closed")
				return
			}

			log.Infof("[Stdout] Processing (%s, %s, %s)\n", e.DNSName, e.IP, e.Hostname)

			err := d.Process(e)
			if err != nil {
				errors <- err
			}
		case <-done:
			log.Info("[Stdout] Exited consuming loop.")
			return
		}
	}
}

func (d *stdoutConsumer) Process(endpoint *pkg.Endpoint) error {
	fmt.Println("process record:", endpoint.DNSName, value(endpoint))
	return nil
}
