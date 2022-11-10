// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

type publishImagesFromTarOptions struct {
	tkgTarFilePath             string
	customImageRepoCertificate string
	pkgClient                  ImgPkgClient
}

var pushImage = &publishImagesFromTarOptions{}

var publishImagesfromtarCmd = &cobra.Command{
	Use:          "publish-image-from-tar",
	Short:        "Copy images from tar files to private repo",
	RunE:         publishImagesFromTar,
	SilenceUsage: true,
}

func init() {
	publishImagesfromtarCmd.Flags().StringVarP(&pushImage.tkgTarFilePath, "tkgTarFilePath", "", "", "Tar file path")
	publishImagesfromtarCmd.Flags().StringVarP(&pushImage.customImageRepoCertificate, "customRepoCertificate", "", "", "custom repo certificate")
}

func (pushImage *publishImagesFromTarOptions) pushImageToRepo() {
	yamlFile := filepath.Join(pushImage.tkgTarFilePath, "publish-images-fromtar.yaml")
	yfile, err := os.ReadFile(yamlFile)
	if err != nil {
		printErrorAndExit(errors.Wrapf(err, " Error while reading publish-images-fromtar.yaml file"))
	}

	data := make(map[string]string)
	err2 := yaml.Unmarshal(yfile, &data)

	if err2 != nil {
		printErrorAndExit(errors.Wrapf(err2, " Error while reading publish-images-fromtar.yaml file"))
	}

	for tarfile, path := range data {
		tarfile = filepath.Join(pushImage.tkgTarFilePath, tarfile)
		pushImage.pkgClient.imgpkgCopyImagefromtar(tarfile, path, pushImage.customImageRepoCertificate)
	}

}
func publishImagesFromTar(cmd *cobra.Command, args []string) error {
	pushImage.pkgClient = &imgpkgclient{}
	pushImage.pushImageToRepo()
	return nil
}
