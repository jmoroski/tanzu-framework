// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

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
	PkgClient       ImgPkgClient
}

var pullImage = &publishImagesToTarOptions{}

var PublishImagestotarCmd = &cobra.Command{
	Use:          "publish-image-to-tar",
	Short:        "Copy images from public repo to tar files",
	RunE:         publishImagesToTar,
	SilenceUsage: true,
}

func init() {
	PublishImagestotarCmd.Flags().StringVarP(&pullImage.tkgImageRepo, "tkgImageRepository", "", "projects.registry.vmware.com/tkg", "TKG public repository")
	PublishImagestotarCmd.Flags().StringVarP(&pullImage.tkgVersion, "tkgVersion", "", "", "TKG version")
	PublishImagestotarCmd.Flags().StringVarP(&pullImage.customImageRepo, "customImageRepo", "", "", "custom images repository for airgapped network")
}

func (pullImage *publishImagesToTarOptions) downloadTkgCompatibilityImage() {
	tkgCompatibilityRelativeImagePath := "tkg-compatibility"
	tkgCompatibilityImagePath := path.Join(pullImage.tkgImageRepo, tkgCompatibilityRelativeImagePath)
	imageTags := pullImage.PkgClient.ImgpkgTagListImage(tkgCompatibilityImagePath)
	sourceImageName := tkgCompatibilityImagePath + ":" + imageTags[len(imageTags)-1]
	tarFilename := tkgCompatibilityRelativeImagePath + "-" + imageTags[len(imageTags)-1] + ".tar"
	pullImage.PkgClient.ImgpkgCopytotar(sourceImageName, tarFilename)
	destRepo := path.Join(pullImage.customImageRepo, tkgCompatibilityRelativeImagePath)
	imageDetails[tarFilename] = destRepo
}

func (pullImage *publishImagesToTarOptions) downloadTkgBomAndComponentImages() error {
	tkgBomImagePath := path.Join(pullImage.tkgImageRepo, "tkg-bom")

	sourceImageName := tkgBomImagePath + ":" + pullImage.tkgVersion
	tarnames := "tkg-bom" + "-" + pullImage.tkgVersion + ".tar"
	destRepo := path.Join(pullImage.customImageRepo, tkgBomImagePath)
	imageDetails[tarnames] = destRepo
	pullImage.PkgClient.ImgpkgCopytotar(sourceImageName, tarnames)
	outputDir := "tmp"
	pullImage.PkgClient.ImgpkgPullImage(sourceImageName, outputDir)
	// read the tkg-bom file
	tkgBomFilePath := filepath.Join(outputDir, fmt.Sprintf("tkg-bom-%s.yaml", pullImage.tkgVersion))
	b, err := os.ReadFile(tkgBomFilePath)

	// read the tkg-bom file
	if err != nil {
		return errors.Wrapf(err, "read tkg-bom file from %s faild", tkgBomFilePath)
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
				pullImage.PkgClient.ImgpkgCopytotar(sourceImageName, tarname)
			}
		}
	}
	return nil
}

func (pullImage *publishImagesToTarOptions) downloadTkrCompatibilityImage(tkrCompatibilityRelativeImagePath string) (tkgVersion []string, err error) {
	// get the latest tag of tkr-compatibility image
	tkrCompatibilityImagePath := path.Join(pullImage.tkgImageRepo, tkrCompatibilityRelativeImagePath)
	imageTags := pullImage.PkgClient.ImgpkgTagListImage(tkrCompatibilityImagePath)
	// inspect the tkr-compatibility image to get the list of compatible tkrs
	tkrCompatibilityImageURL := tkrCompatibilityImagePath + ":" + imageTags[len(imageTags)-1]

	sourceImageName := tkrCompatibilityImageURL
	outputDir := "tmp"
	pullImage.PkgClient.ImgpkgPullImage(sourceImageName, outputDir)
	files, err := os.ReadDir("tmp")
	if err != nil {
		return nil, errors.Wrapf(err, "read directory tmp failed")
	}
	if len(files) != 1 || files[0].IsDir() {
		return nil, fmt.Errorf("tkr-compatibility image should only has exact one file inside")
	}
	tkrCompatibilityFilePath := filepath.Join("tmp", files[0].Name())
	b, err := os.ReadFile(tkrCompatibilityFilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "read tkr-compatibility file from %s faild", tkrCompatibilityFilePath)
	}
	tkrCompatibility := &tkrv1.CompatibilityMetadata{}
	if err := yaml.Unmarshal(b, tkrCompatibility); err != nil {
		return nil, errors.Wrapf(err, "Unmarshal tkr-compatibility file %s failed", tkrCompatibilityFilePath)
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
		return nil, fmt.Errorf("couldn't find the corresponding tkg-bom version in the tkr-compatibility file")
	}
	// imgpkg copy the tkr-compatibility image
	sourceImageName = tkrCompatibilityImageURL
	tarFilename := tkrCompatibilityRelativeImagePath + "-" + imageTags[len(imageTags)-1] + ".tar"
	destImageRepo := path.Join(pullImage.customImageRepo, tkrCompatibilityRelativeImagePath)
	imageDetails[tarFilename] = destImageRepo
	pullImage.PkgClient.ImgpkgCopytotar(sourceImageName, tarFilename)
	return tkrVersions, nil
}

func (pullImage *publishImagesToTarOptions) downloadTkrBomAndComponentImages(tkrVersion string) error {
	tkrTag := underscoredPlus(tkrVersion)
	tkrBomImagePath := path.Join(pullImage.tkgImageRepo, "tkr-bom")
	sourceImageName := tkrBomImagePath + ":" + tkrTag
	tarFilename := "tkr-bom" + "-" + tkrTag + ".tar"
	destImageRepo := path.Join(pullImage.customImageRepo, "tkr-bom")
	imageDetails[tarFilename] = destImageRepo
	pullImage.PkgClient.ImgpkgCopytotar(sourceImageName, tarFilename)

	sourceImageName = tkrBomImagePath + ":" + tkrTag
	outputDir := "tmp"
	pullImage.PkgClient.ImgpkgPullImage(sourceImageName, outputDir)

	// read the tkr-bom file
	tkrBomFilePath := filepath.Join("tmp", fmt.Sprintf("tkr-bom-%s.yaml", tkrVersion))
	b, err := os.ReadFile(tkrBomFilePath)
	if err != nil {
		return errors.Wrapf(err, "read tkr-bom file from %s faild", tkrBomFilePath)
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
				pullImage.PkgClient.ImgpkgCopytotar(sourceImageName, tarname)
			}
		}
	}
	return nil
}

func publishImagesToTar(cmd *cobra.Command, args []string) error {
	pullImage.PkgClient = &imgpkgclient{}
	if !IsTagValid(pullImage.tkgVersion) {
		return fmt.Errorf("Invalid TKG Tag %s", pullImage.tkgVersion)
	}
	if pullImage.tkgImageRepo == "" { //TODO : Put more validation
		return fmt.Errorf("Invalid tkgImageRepository %s", pullImage.tkgImageRepo)
	}
	if pullImage.customImageRepo == "" {
		return fmt.Errorf("Invalid customImageRepo %s", pullImage.customImageRepo)
	}
	pullImage.downloadTkgCompatibilityImage()
	pullImage.downloadTkgBomAndComponentImages()
	tkrVersions, err := pullImage.downloadTkrCompatibilityImage("tkr-compatibility")
	if err != nil {
		return errors.Wrapf(err, "Error while retrieving tkrVersions")
	}
	for _, tkrVersion := range tkrVersions {
		pullImage.downloadTkrBomAndComponentImages(tkrVersion)
	}

	data, _ := yaml.Marshal(&imageDetails)
	err2 := os.WriteFile("publish-images-fromtar.yaml", data, 0666)
	if err2 != nil {
		return errors.Wrapf(err2, "Error while writing publish-images-fromtar.yaml file")
	}
	//	fmt.Sprintf("Success! Copied a total number of %v images.\n", totalImgCopiedCounter)

	return nil
}
