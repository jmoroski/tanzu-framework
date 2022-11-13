// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package cmd ImgPkgClient defines functions to pull/push/List images
package cmd

type ImgPkgClient interface {
	ImgpkgCopyImagefromtar(sourceImageName string, destImageRepo string, customImageRepoCertificate string)
	ImgpkgCopytotar(sourceImageName string, destImageRepo string)
	ImgpkgPullImage(sourceImageName string, destDir string)
	ImgpkgTagListImage(sourceImageName string) []string
}
