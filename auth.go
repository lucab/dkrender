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
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/registry/client/auth"
	"github.com/docker/distribution/registry/client/auth/challenge"
	"github.com/docker/distribution/registry/client/transport"
)

func tokenAuth(trustServerURL string, baseTransport *http.Transport, img string) (http.RoundTripper, error) {
	authTransport := transport.NewTransport(baseTransport)
	pingClient := &http.Client{
		Transport: authTransport,
	}
	endpoint, err := url.Parse(trustServerURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse remote trust server url (%s): %s", trustServerURL, err.Error())
	}
	if endpoint.Scheme != "https" && endpoint.Scheme != "http" {
		return nil, fmt.Errorf("unknown trust server URL scheme, got %v", trustServerURL)
	}
	subPath, err := url.Parse(path.Join(endpoint.Path, "/v2") + "/")
	if err != nil {
		return nil, fmt.Errorf("Failed to parse v2 subpath: %s", err.Error())
	}
	endpoint = endpoint.ResolveReference(subPath)
	req, err := http.NewRequest("GET", endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := pingClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if (resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices) &&
		resp.StatusCode != http.StatusUnauthorized {
		return nil, fmt.Errorf("could not reach %s: %d", trustServerURL, resp.StatusCode)
	}

	challengeManager := challenge.NewSimpleManager()
	if err := challengeManager.AddResponse(resp); err != nil {
		return nil, err
	}

	ps := passwordStore{}

	actions := []string{"pull"}

	tokenHandler := auth.NewTokenHandler(authTransport, ps, img, actions...)
	basicHandler := auth.NewBasicHandler(ps)

	modifier := auth.NewAuthorizer(challengeManager, tokenHandler, basicHandler)

	// Try to authenticate read only repositories using basic username/password authentication
	return newAuthRoundTripper(transport.NewTransport(baseTransport, modifier),
		transport.NewTransport(baseTransport, auth.NewAuthorizer(challengeManager, auth.NewTokenHandler(authTransport, passwordStore{}, img, actions...)))), nil
}

type passwordStore struct {
}

func (ps passwordStore) Basic(u *url.URL) (string, string) {
	return "", ""
}

// to comply with the CredentialStore interface
func (ps passwordStore) RefreshToken(u *url.URL, service string) string {
	return ""
}

// to comply with the CredentialStore interface
func (ps passwordStore) SetRefreshToken(u *url.URL, service string, token string) {
}

// authRoundTripper tries to authenticate the requests via multiple HTTP transactions (until first succeed)
type authRoundTripper struct {
	trippers []http.RoundTripper
}

func newAuthRoundTripper(trippers ...http.RoundTripper) http.RoundTripper {
	return &authRoundTripper{trippers: trippers}
}

func (a *authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {

	var resp *http.Response
	// Try all run all transactions
	for _, t := range a.trippers {
		/*
			logrus.WithFields(logrus.Fields{
				"url":     req.URL,
				"headers": req.Header,
			}).Debug("sending request")
		*/
		var err error
		resp, err = t.RoundTrip(req)

		// Reject on error
		if err != nil {
			return resp, err
		}
		// Stop when request is authorized/unknown error
		if resp.StatusCode != http.StatusUnauthorized {
			/*
				logrus.WithFields(logrus.Fields{
					"response": resp,
				}).Debug("got authenticated")
			*/
			return resp, nil
		}
	}

	// None of the above worked, return the last response
	logrus.Debug("authentication failed")
	return resp, nil
}
