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

func (d *stdoutConsumer) Sync(endpoints []*pkg.Endpoint) error {
	for _, e := range endpoints {
		fmt.Println(e.DNSName, e.IP)
	}

	return nil
}

func (d *stdoutConsumer) Process(endpoint *pkg.Endpoint) error {
	fmt.Println(endpoint.DNSName, endpoint.IP)
	return nil
}
