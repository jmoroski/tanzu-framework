// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package ImgPkgClient defines functions to pull/push/List images
package cmd

// ImgPkgClient defines methods to pull/push/List images (similar to imgpkg)

type ImgPkgClient interface {
	ImgpkgCopyImagefromtar(sourceImageName string, destImageRepo string, customImageRepoCertificate string)
	ImgpkgCopytotar(sourceImageName string, destImageRepo string)
	ImgpkgPullImage(sourceImageName string, destDir string)
	ImgpkgTagListImage(sourceImageName string) []string
}
