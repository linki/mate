package consumers

import (
	"fmt"
	"sync"

	"github.bus.zalan.do/teapot/mate/pkg"
)

// new StreamWriterProvider io.Writer instead of stdout
// or new file writrer better
type stdoutConsumer struct {
	sync.Mutex
}

func NewStdout() (*stdoutConsumer, error) {
	return &stdoutConsumer{}, nil
}

func (d *stdoutConsumer) Sync(endpoints []*pkg.Endpoint) error {
	d.Lock()
	defer d.Unlock()

	for _, e := range endpoints {
		fmt.Println(e.DNSName, e.IP)
	}

	return nil
}

func (d *stdoutConsumer) Process(endpoint *pkg.Endpoint) error {
	d.Lock()
	defer d.Unlock()

	fmt.Println(endpoint.DNSName, endpoint.IP)

	return nil
}
