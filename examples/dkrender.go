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

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/context"
	"github.com/lucab/dkrender"
)

func main() {
	// Note: this turns on full debug logging
	logrus.SetLevel(logrus.DebugLevel)

	err := run()
	if err != nil {
		logrus.Fatal(err)
	}
}

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: dkrender $IMAGE")
	}
	imgName := os.Args[1]
	curUser, err := user.Current()
	if err != nil {
		return err
	}
	dstFile := filepath.Join(curUser.HomeDir, "dkr_image.tgz")
	dir, err := ioutil.TempDir(os.TempDir(), "dkrender_")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	fmt.Printf("writing layers to tmpdir %s\n", dir)

	// First, check image trust, fetch the manifest and download layers
	ctx := context.Background()
	fetchCfg := dkrender.DefaultFetchCfg()
	client, err := dkrender.NewClient(fetchCfg)
	if err != nil {
		return err
	}
	layers, err := client.DockerV2Fetch(ctx, dir, imgName)
	if err != nil {
		return err
	}
	defer dkrender.CloseLayers(layers)
	fmt.Printf("Got %d layers to render...\n", len(layers))

	// Then, squash and render the final tarball
	fp, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer fp.Close()
	if err := dkrender.SquashLayers(layers, fp); err != nil {
		defer os.Remove(dstFile)
		return err
	}

	fmt.Printf("Successfully rendered image to %q\n", dstFile)
	return nil
}
