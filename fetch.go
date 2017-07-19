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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/context"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	regclient "github.com/docker/distribution/registry/client"
	"github.com/docker/notary/client"
	"github.com/docker/notary/tuf/data"
	digest "github.com/opencontainers/go-digest"
)

// DockerV2Fetch fetches a DockerV2 image and stores it as a tgz
// containing the rendered rootfs, returning image details.
func (cl Client) DockerV2Fetch(ctx context.Context, destDir, refIn string) ([]*os.File, error) {
	refIn = strings.TrimPrefix(refIn, "docker://")

	// Parse docker image reference
	defaultTag := "latest"
	if cl.cfg.DefaultTag != "" {
		defaultTag = cl.cfg.DefaultTag
	}
	normalizedNamed, err := reference.ParseNormalizedNamed(refIn)
	if err != nil {
		return nil, err
	}
	if reference.IsNameOnly(normalizedNamed) {
		imgString := normalizedNamed.String() + ":" + defaultTag
		normalizedNamed, err = reference.ParseNormalizedNamed(imgString)
		if err != nil {
			return nil, err
		}
	}
	familiarNamed, err := reference.WithName(reference.Path(normalizedNamed))
	if err != nil {
		return nil, err
	}
	normalizedTagged, ok := normalizedNamed.(reference.NamedTagged)
	if !ok {
		return nil, fmt.Errorf("missing tag")
	}
	tag := normalizedTagged.Tag()
	logrus.WithFields(logrus.Fields{
		"canonical tagged name": normalizedTagged,
		"familiar name":         familiarNamed,
	}).Debug("parsed image reference")

	// Determine registry and notary endpoints
	scheme := "https://"
	if cl.cfg.DisableHTTPS {
		scheme = "http://"
	}
	regHost := reference.Domain(normalizedNamed)
	notaryHost := regHost
	if regHost == "docker.io" {
		regHost = "registry-1.docker.io"
		notaryHost = "notary.docker.io"
	}
	regURL := scheme + regHost
	notaryURL := scheme + notaryHost
	logrus.WithFields(logrus.Fields{
		"registry": regURL,
		"notary":   notaryURL,
	}).Debug("got remote endpoints")

	tmp, err := ioutil.TempDir(os.TempDir(), "notary")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	// Retrieve trusted hash from notary
	gun := data.GUN(normalizedNamed.Name())
	nrt, err := tokenAuth(notaryURL, cl.cfg.Transport, gun.String())
	if err != nil {
		return nil, err
	}
	notaryClient, err := client.NewFileCachedNotaryRepository(tmp, gun, notaryURL, nrt, nil, cl.cfg.TrustPin)
	if err != nil {
		return nil, err
	}
	t, err := notaryClient.GetTargetByName(tag)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("no hashes in signed tag")
	}
	hash := digest.NewDigestFromBytes(cl.cfg.DigestAlgo, t.Hashes["sha256"])
	logrus.WithFields(logrus.Fields{
		"sha256": hash,
	}).Debug("got hash for trusted tag")

	// Check for a manifest with matching hash
	rrt, err := tokenAuth(regURL, cl.cfg.Transport, familiarNamed.Name())
	if err != nil {
		return nil, err
	}
	repo, err := regclient.NewRepository(ctx, familiarNamed, regURL, rrt)
	if err != nil {
		return nil, err
	}
	svc, err := repo.Manifests(ctx, nil)
	if err != nil {
		return nil, err
	}
	blobsStore := repo.Blobs(ctx)
	hasHash, err := svc.Exists(ctx, hash)
	if err != nil {
		return nil, err
	}
	if !hasHash {
		return nil, fmt.Errorf("manifest not found")
	}
	logrus.WithFields(logrus.Fields{
		"registry": regHost,
		"sha256":   hash,
		"tag":      normalizedTagged.Tag(),
	}).Debug("manifest exists on registry")

	manif, err := svc.Get(ctx, hash)
	if err != nil {
		return nil, err
	}

	layersLen := len(manif.References())
	layers := []*os.File{}
	for i, desc := range manif.References() {
		logrus.Infof("downloading layer %d/%d", i+1, layersLen)
		if desc.MediaType != schema2.MediaTypeLayer && desc.MediaType != schema1.MediaTypeManifestLayer {
			continue
		}
		blob, err := blobsStore.Get(ctx, desc.Digest)
		if err != nil {
			return nil, err
		}
		logrus.WithFields(logrus.Fields{
			"digest":    desc.Digest,
			"layer":     i,
			"length":    len(blob),
			"mediatype": desc.MediaType,
		}).Debug("got blob")
		destName := filepath.Join(destDir, desc.Digest.String()+".tgz")
		if err := ioutil.WriteFile(destName, blob, 0755); err != nil {
			return nil, err
		}
		fp, err := os.Open(destName)
		if err != nil {
			return nil, err
		}
		layers = append(layers, fp)
	}

	return layers, nil
}
