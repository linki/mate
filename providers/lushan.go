package providers

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"

	log "github.com/Sirupsen/logrus"
	"github.com/zalando/go-tokens/tokens"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/go-openapi/runtime"
	openapi "github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"github.bus.zalan.do/teapot/kyusu/pkg/lushan"
	apiclient "github.bus.zalan.do/teapot/lushan/client"
)

const (
	defaultLushanServer = "http://127.0.0.1:4180"
	defaultAuthURL      = "https://token.services.auth.zalando.com/oauth2/access_token"
	tokenID             = "mate"
)

type lushanProvider struct {
	client   *apiclient.Lushan
	authInfo runtime.ClientAuthInfoWriter
}

// var params struct {
// 	lushanServer string
// 	authURL      *url.URL
// 	token        string
// }

var tokenScopes = []string{"uid"}

func init() {
	kingpin.Flag("lushan-server", "The address of the Lushan API server.").Default(defaultLushanServer).StringVar(&params.lushanServer)
	kingpin.Flag("auth-url", "URL of the access token issuer.").Default(defaultAuthURL).URLVar(&params.authURL)
	kingpin.Flag("token", "Provide a token if you don't want it to be managed by this tool.").StringVar(&params.token)
}

func (a *lushanProvider) Endpoints() ([]*Endpoint, error) {
	resp, err := a.client.Clusters.ListClusters(nil, a.authInfo)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving list of desired clusters: %v", err)
	}
	desiredClusters := resp.Payload

	log.Debugln("List of desired clusters:")
	for _, c := range desiredClusters {
		log.Debugf("- %s/%s (%s)", c.ProjectID, c.Name, c.Endpoint)
	}

	for _, c := range desiredClusters {
		log.Debugf("- %s/%s (%s)", c.ProjectID, c.Name, c.Endpoint)
	}

	ret := make([]*Endpoint, 0, len(desiredClusters))

	for _, svc := range desiredClusters {
		ret = append(ret, &Endpoint{
			DNSName: svc.Name,
			IP:      svc.Endpoint,
		})
	}

	return ret, nil
}

func NewLushanProvider() (*lushanProvider, error) {
	var authInfo openapi.ClientAuthInfoWriter
	if params.token != "" {
		authInfo = httptransport.BearerToken(params.token)
	} else {
		reqs := []tokens.ManagementRequest{
			tokens.NewPasswordRequest(tokenID, tokenScopes...),
		}
		tokensManager, err := tokens.Manage(params.authURL.String(), reqs)
		if err != nil {
			return nil, fmt.Errorf("Failed to setup tokens manager: %v", err)
		}
		authInfo = &lushan.OAuthWriter{tokensManager, tokenID}
	}

	url, err := url.Parse(params.lushanServer)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse Lushan API server URL: %v", err)
	}

	lushan := NewLushanClient(url, &Options{
		Insecure: true,
		Debug:    true,
	})

	return &lushanProvider{
		client:   lushan,
		authInfo: authInfo,
	}, nil
}

type Options struct {
	Insecure bool
	Debug    bool
}

func NewLushanClient(server *url.URL, options *Options) *apiclient.Lushan {
	// create the transport
	transport := httptransport.New(server.Host, server.Path, []string{server.Scheme})
	transport.Debug = options.Debug

	// extract the underlying transport
	t := transport.Transport.(*http.Transport)

	// initialize its tls config
	if t.TLSClientConfig == nil {
		t.TLSClientConfig = &tls.Config{}
	}

	// skip tls verification
	t.TLSClientConfig.InsecureSkipVerify = options.Insecure

	// create the API client, with the transport
	client := apiclient.New(transport, strfmt.Default)

	// return the client
	return client
}
