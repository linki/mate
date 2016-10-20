package consumers

import (
	"fmt"

	"github.bus.zalan.do/teapot/mate/pkg"
)

// new StreamWriterProvider io.Writer instead of stdout
// or new file writrer better
type stdoutConsumer struct{}

func NewStdout() (Consumer, error) {
	return &stdoutConsumer{}, nil
}

func value(ep *pkg.Endpoint) string {
	if ep.IP != "" {
		return ep.IP
	}

	return ep.Hostname
}

func (d *stdoutConsumer) Sync(endpoints []*pkg.Endpoint) error {
	for _, e := range endpoints {
		fmt.Println("sync record:", e.DNSName, value(e))
	}

	return nil
}

func (d *stdoutConsumer) Process(endpoint *pkg.Endpoint) error {
	fmt.Println("process record:", endpoint.DNSName, value(endpoint))
	return nil
}
