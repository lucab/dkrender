// Copyright 2017 CoreOS Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dkrender

import (
	"net/http"

	"github.com/docker/notary/trustpinning"
	"github.com/opencontainers/go-digest"
)

// FetcherCfg contains configuration options for the docker fetcher.
type FetcherCfg struct {
	// DefaultTag specifies the default to use if missing (e.g. "latest")
	DefaultTag string
	// DigestAlgo specifies digest algorithm to use (for notary and manifest)
	DigestAlgo digest.Algorithm
	// DisableHTTPS is used to opt-out https, falling back to http
	DisableHTTPS bool
	// DisableNotary is used to opt-out notary check
	DisableNotary bool
	// Transport is used to customize the HTTP transport used by the client
	Transport *http.Transport
	// TrustPin configures trust pinning
	TrustPin trustpinning.TrustPinConfig
}

// DefaultFetchCfg builds the default fetcher configuration.
func DefaultFetchCfg() FetcherCfg {
	return FetcherCfg{
		DefaultTag:    "latest",
		DigestAlgo:    digest.Canonical,
		DisableHTTPS:  false,
		DisableNotary: false,
		Transport:     http.DefaultTransport.(*http.Transport),
		TrustPin:      trustpinning.TrustPinConfig{},
	}
}

// Client is the docker registry fetcher client
type Client struct {
	cfg FetcherCfg
}

// NewClient builds a new Client from the given fetcher configuration.
func NewClient(fetchCfg FetcherCfg) (Client, error) {
	return Client{
		cfg: fetchCfg,
	}, nil
}
