// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"sort"
	"strconv"
	"strings"

	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/pkg/errors"

	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/bundle"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/cmd"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"

	v1 "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/v1"
)

type imgpkgclient struct {
}

func (pkgClient *imgpkgclient) imgpkgCopyImagefromtar(sourceImageName string, destImageRepo string, customImageRepoCertificate string) {
	confUI := ui.NewConfUI(ui.NewNoopLogger())
	copyOptions := cmd.NewCopyOptions(confUI)
	copyOptions.Concurrency = 1
	copyOptions.TarFlags.TarSrc = sourceImageName
	copyOptions.RepoDst = destImageRepo
	if customImageRepoCertificate != "" {
		copyOptions.RegistryFlags.CACertPaths = []string{customImageRepoCertificate}
	}
	err := copyOptions.Run()
	if err != nil {
		printErrorAndExit(errors.Wrapf(err, "failed!\n"))
	}
}

func (pkgClient *imgpkgclient) imgpkgCopytotar(sourceImageName string, destImageRepo string) {
	confUI := ui.NewConfUI(ui.NewNoopLogger())
	copyOptions := cmd.NewCopyOptions(confUI)
	copyOptions.TarFlags.Resume = true
	copyOptions.Concurrency = 5
	reg, err := registry.NewSimpleRegistry(registry.Opts{})
	if err != nil {
		printErrorAndExit(errors.Wrapf(err, "failed!\n"))
	}
	newBundle := bundle.NewBundle(sourceImageName, reg)
	isBundle, _ := newBundle.IsBundle()
	if isBundle {
		copyOptions.BundleFlags = cmd.BundleFlags{Bundle: sourceImageName}
	} else {
		copyOptions.ImageFlags = cmd.ImageFlags{Image: sourceImageName}
	}
	copyOptions.TarFlags.TarDst = destImageRepo
	err = copyOptions.Run()
	if err != nil {
		printErrorAndExit(errors.Wrapf(err, "failed!\n"))
	}
	totalImgCopiedCounter++
}

func (pkgClient *imgpkgclient) imgpkgPullImage(sourceImageName string, destDir string) {
	var outputBuf, errorBuf bytes.Buffer
	writerUI := ui.NewWriterUI(&outputBuf, &errorBuf, nil)
	pullOptions := cmd.NewPullOptions(writerUI)
	pullOptions.OutputPath = destDir
	pullOptions.ImageFlags = cmd.ImageFlags{Image: sourceImageName}
	pullOptions.Run()
}

func (pkgClient *imgpkgclient) imgpkgTagListImage(sourceImageName string) []string {
	tagInfo, _ := v1.TagList(sourceImageName, false, registry.Opts{})
	var imageTags []string
	for _, tag := range tagInfo.Tags {
		imageTags = append(imageTags, tag.Tag)
	}
	sort.SliceStable(imageTags, func(i, j int) bool {
		vi, err := strconv.Atoi(strings.TrimPrefix(imageTags[i], "v"))
		if err != nil {
			printErrorAndExit(errors.Wrapf(err, "parse tkg-compatibility image tag %s failed", imageTags[i]))
		}
		vj, err := strconv.Atoi(strings.TrimPrefix(imageTags[j], "v"))
		if err != nil {
			printErrorAndExit(errors.Wrapf(err, "parse tkg-compatibility image tag %s failed", imageTags[j]))
		}
		return vi < vj
	})
	return imageTags
}
