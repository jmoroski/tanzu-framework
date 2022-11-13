// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-framework/cmd/cli/plugin/airgapped-network/cmd"
	"github.com/vmware-tanzu/tanzu-framework/cmd/cli/plugin/airgapped-network/fakes"
	"github.com/vmware-tanzu/tanzu-framework/tkg/utils"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Publish image from tar file")
}

var _ = Describe("pushImageToRepo()", func() {
	pushImage := &cmd.PublishImagesFromTarOptions{}

	BeforeEach(func() {
		pushImage.PkgClient = &fakes.ImgPkgClientFake{}

	})

	When("publish-images-fromtar.yaml, which contain tar file name and destination repo path, doesn't existed", func() {
		It("should return err", func() {
			err := pushImage.PushImageToRepo()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("while reading publish-images-fromtar.yaml file"))
		})
	})
	When("publish-images-fromtar.yaml, which contain tar file name and destination repo path, has wrong format", func() {
		It("should return err", func() {
			pushImage.TkgTarFilePath = "./testdata"
			err := utils.CopyFile("./testdata/publish-images-fromtar_with_error.yaml", "./testdata/publish-images-fromtar.yaml")
			Expect(err).ToNot(HaveOccurred())
			err = pushImage.PushImageToRepo()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Error while parsing publish-images-fromtar.yaml file"))
		})
	})

})
