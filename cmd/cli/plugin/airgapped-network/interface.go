// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package ImgPkgClient defines functions to pull/push/List images
package main

// ImgPkgClient defines methods to pull/push/List images (similar to imgpkg)

type ImgPkgClient interface {
	imgpkgCopyImagefromtar(sourceImageName string, destImageRepo string, customImageRepoCertificate string)
	imgpkgCopytotar(sourceImageName string, destImageRepo string)
	imgpkgPullImage(sourceImageName string, destDir string)
	imgpkgTagListImage(sourceImageName string) []string
}
