// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	tkrv1 "github.com/vmware-tanzu/tanzu-framework/apis/run/pkg/tkr/v1"
)

var imageDetails = map[string]string{}

var totalImgCopiedCounter int = 0

type publishImagesToTarOptions struct {
	tkgImageRepo    string
	tkgVersion      string
	tarFilePath     string
	customImageRepo string
	pkgclient       ImgPkgClient
}

var pullImage = &publishImagesToTarOptions{}

var publishImagestotarCmd = &cobra.Command{
	Use:          "publish-image-to-tar",
	Short:        "Copy images from public repo to tar files",
	RunE:         publishImagesToTar,
	SilenceUsage: true,
}

func init() {
	publishImagestotarCmd.Flags().StringVarP(&pullImage.tkgImageRepo, "tkgImageRepository", "", "projects.registry.vmware.com/tkg", "TKG public repository")
	publishImagestotarCmd.Flags().StringVarP(&pullImage.tkgVersion, "tkgVersion", "", "", "TKG version")
	publishImagestotarCmd.Flags().StringVarP(&pullImage.customImageRepo, "customImageRepo", "", "", "custom images repository for airgapped network")
}

func (pullImage *publishImagesToTarOptions) downloadTkgCompatibilityImage() {
	fmt.Sprintf("--- start the process of tkg-compatibility ---")
	tkgCompatibilityRelativeImagePath := "tkg-compatibility"
	tkgCompatibilityImagePath := path.Join(pullImage.tkgImageRepo, tkgCompatibilityRelativeImagePath)
	imageTags := pullImage.pkgclient.imgpkgTagListImage(tkgCompatibilityImagePath)
	sourceImageName := tkgCompatibilityImagePath + ":" + imageTags[len(imageTags)-1]
	tarFilename := tkgCompatibilityRelativeImagePath + "-" + imageTags[len(imageTags)-1] + ".tar"
	pullImage.pkgclient.imgpkgCopytotar(sourceImageName, tarFilename)
	destRepo := path.Join(pullImage.customImageRepo, tkgCompatibilityRelativeImagePath)
	imageDetails[tarFilename] = destRepo
	fmt.Sprintf("--- finish the process of tkg-compatibility ---\n")

}

func (pullImage *publishImagesToTarOptions) downloadTkgBomAndComponentImages() {
	fmt.Sprintf("--- start the process of tkg-bom ---\n")
	tkgBomImagePath := path.Join(pullImage.tkgImageRepo, "tkg-bom")

	sourceImageName := tkgBomImagePath + ":" + pullImage.tkgVersion
	tarnames := "tkg-bom" + "-" + pullImage.tkgVersion + ".tar"
	destRepo := path.Join(pullImage.customImageRepo, tkgBomImagePath)
	imageDetails[tarnames] = destRepo
	pullImage.pkgclient.imgpkgCopytotar(sourceImageName, tarnames)
	outputDir := "tmp"
	pullImage.pkgclient.imgpkgPullImage(sourceImageName, outputDir)
	// read the tkg-bom file
	tkgBomFilePath := filepath.Join(outputDir, fmt.Sprintf("tkg-bom-%s.yaml", pullImage.tkgVersion))
	b, err := os.ReadFile(tkgBomFilePath)

	// read the tkg-bom file
	if err != nil {
		printErrorAndExit(errors.Wrapf(err, "read tkg-bom file from %s faild", tkgBomFilePath))
	}
	tkgBom, err := tkrv1.NewBom(b)
	// imgpkg copy each component's artifacts
	components, err := tkgBom.Components()
	for _, compInfos := range components {
		for _, compInfo := range compInfos {
			for _, imageInfo := range compInfo.Images {
				sourceImageName = filepath.Join(pullImage.tkgImageRepo, imageInfo.ImagePath) + ":" + imageInfo.Tag
				destImageRepo := path.Join(pullImage.customImageRepo, imageInfo.ImagePath)
				imageInfo.ImagePath = replaceSlash(imageInfo.ImagePath)
				tarname := imageInfo.ImagePath + "-" + imageInfo.Tag + ".tar"
				imageDetails[tarname] = destImageRepo
				pullImage.pkgclient.imgpkgCopytotar(sourceImageName, tarname)
			}
		}
	}
}

func (pullImage *publishImagesToTarOptions) downloadTkrCompatibilityImage(tkrCompatibilityRelativeImagePath string) []string {
	fmt.Sprintf("--- start the process of tkr-compatibility ---\n")
	// get the latest tag of tkr-compatibility image
	tkrCompatibilityImagePath := path.Join(pullImage.tkgImageRepo, tkrCompatibilityRelativeImagePath)
	imageTags := pullImage.pkgclient.imgpkgTagListImage(tkrCompatibilityImagePath)
	// inspect the tkr-compatibility image to get the list of compatible tkrs
	tkrCompatibilityImageURL := tkrCompatibilityImagePath + ":" + imageTags[len(imageTags)-1]

	sourceImageName := tkrCompatibilityImageURL
	outputDir := "tmp"
	pullImage.pkgclient.imgpkgPullImage(sourceImageName, outputDir)
	files, err := os.ReadDir("tmp")
	if err != nil {
		printErrorAndExit(errors.Wrapf(err, "read directory tmp failed"))
	}
	if len(files) != 1 || files[0].IsDir() {
		printErrorAndExit(fmt.Errorf("tkr-compatibility image should only has exact one file inside"))
	}
	tkrCompatibilityFilePath := filepath.Join("tmp", files[0].Name())
	b, err := os.ReadFile(tkrCompatibilityFilePath)
	if err != nil {
		printErrorAndExit(errors.Wrapf(err, "read tkr-compatibility file from %s faild", tkrCompatibilityFilePath))
	}
	tkrCompatibility := &tkrv1.CompatibilityMetadata{}
	if err := yaml.Unmarshal(b, tkrCompatibility); err != nil {
		printErrorAndExit(errors.Wrapf(err, "Unmarshal tkr-compatibility file %s failed", tkrCompatibilityFilePath))
	}
	// find the corresponding tkg-bom entry
	var tkrVersions []string
	var found = false
	for _, compatibilityInfo := range tkrCompatibility.ManagementClusterVersions {
		if compatibilityInfo.TKGVersion == pullImage.tkgVersion {
			found = true
			tkrVersions = compatibilityInfo.SupportedKubernetesVersions
			break
		}
	}
	if !found {
		printErrorAndExit(fmt.Errorf("couldn't find the corresponding tkg-bom version in the tkr-compatibility file"))
	}
	// imgpkg copy the tkr-compatibility image
	sourceImageName = tkrCompatibilityImageURL
	tarFilename := tkrCompatibilityRelativeImagePath + "-" + imageTags[len(imageTags)-1] + ".tar"
	destImageRepo := path.Join(pullImage.customImageRepo, tkrCompatibilityRelativeImagePath)
	imageDetails[tarFilename] = destImageRepo
	pullImage.pkgclient.imgpkgCopytotar(sourceImageName, tarFilename)

	fmt.Sprintf("--- finish the process of tkr-compatibility ---\n")
	return tkrVersions
}

func (pullImage *publishImagesToTarOptions) downloadTkrBomAndComponentImages(tkrVersion string) {
	fmt.Sprintf("--- start the process of tkr-bom ---\n")
	tkrTag := underscoredPlus(tkrVersion)
	tkrBomImagePath := path.Join(pullImage.tkgImageRepo, "tkr-bom")
	sourceImageName := tkrBomImagePath + ":" + tkrTag
	tarFilename := "tkr-bom" + "-" + tkrTag + ".tar"
	destImageRepo := path.Join(pullImage.customImageRepo, "tkr-bom")
	imageDetails[tarFilename] = destImageRepo
	pullImage.pkgclient.imgpkgCopytotar(sourceImageName, tarFilename)

	sourceImageName = tkrBomImagePath + ":" + tkrTag
	outputDir := "tmp"
	pullImage.pkgclient.imgpkgPullImage(sourceImageName, outputDir)

	// read the tkr-bom file
	tkrBomFilePath := filepath.Join("tmp", fmt.Sprintf("tkr-bom-%s.yaml", tkrVersion))
	b, err := os.ReadFile(tkrBomFilePath)
	if err != nil {
		printErrorAndExit(errors.Wrapf(err, "read tkr-bom file from %s faild", tkrBomFilePath))
	}
	tkgBom, err := tkrv1.NewBom(b)
	// imgpkg copy each component's artifacts
	components, err := tkgBom.Components()
	for _, compInfos := range components {
		for _, compInfo := range compInfos {
			for _, imageInfo := range compInfo.Images {
				sourceImageName = filepath.Join(pullImage.tkgImageRepo, imageInfo.ImagePath) + ":" + imageInfo.Tag
				destImageRepo := path.Join(pullImage.customImageRepo, imageInfo.ImagePath)
				imageInfo.ImagePath = replaceSlash(imageInfo.ImagePath)
				tarname := imageInfo.ImagePath + "-" + imageInfo.Tag + ".tar"
				imageDetails[tarname] = destImageRepo
				pullImage.pkgclient.imgpkgCopytotar(sourceImageName, tarname)
			}
		}
	}
	fmt.Sprintf("--- finish the process of tkr-bom ---\n")
}

func publishImagesToTar(cmd *cobra.Command, args []string) error {
	pullImage.pkgclient = &imgpkgclient{}
	if !IsTagValid(pullImage.tkgVersion) {
		printErrorAndExit(fmt.Errorf("Invalid TKG Tag %s", pullImage.tkgVersion))
	}
	if pullImage.tkgImageRepo == "" { //TODO : Put more validation
		printErrorAndExit(fmt.Errorf("Invalid tkgImageRepository %s", pullImage.tkgImageRepo))
	}
	if pullImage.customImageRepo == "" {
		printErrorAndExit(fmt.Errorf("Invalid customImageRepo %s", pullImage.customImageRepo))
	}
	pullImage.downloadTkgCompatibilityImage()
	pullImage.downloadTkgBomAndComponentImages()
	tkrVersions := pullImage.downloadTkrCompatibilityImage("tkr-compatibility")
	for _, tkrVersion := range tkrVersions {
		pullImage.downloadTkrBomAndComponentImages(tkrVersion)
	}

	data, _ := yaml.Marshal(&imageDetails)
	err := os.WriteFile("publish-images-fromtar.yaml", data, 0666)
	if err != nil {
		printErrorAndExit(errors.Wrapf(err, "Error while writing publish-images-fromtar.yaml file"))
	}
	fmt.Sprintf("Success! Copied a total number of %v images.\n", totalImgCopiedCounter)

	return nil
}
