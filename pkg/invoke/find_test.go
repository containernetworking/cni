package invoke_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/appc/cni/pkg/invoke"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("FindInPath", func() {
	var (
		multiplePaths  []string
		pluginName     string
		pluginDir      string
		anotherTempDir string
	)

	BeforeEach(func() {
		tempDir, err := ioutil.TempDir("", "cni-find")
		Expect(err).NotTo(HaveOccurred())

		plugin, err := ioutil.TempFile(tempDir, "a-cni-plugin")

		anotherTempDir, err = ioutil.TempDir("", "nothing-here")

		multiplePaths = []string{anotherTempDir, tempDir}
		pluginDir, pluginName = filepath.Split(plugin.Name())
	})

	Context("when multiple paths are provided", func() {
		It("returns only the path to the plugin", func() {
			pluginPath, err := invoke.FindInPath(pluginName, multiplePaths)
			Expect(err).NotTo(HaveOccurred())
			Expect(pluginPath).To(Equal(filepath.Join(pluginDir, pluginName)))
		})
	})

	Context("when an error occurs", func() {
		Context("when no paths are provided", func() {
			It("returns an error noting no paths were provided", func() {
				_, err := invoke.FindInPath(pluginName, []string{})
				Expect(err).To(MatchError("no paths provided"))
			})
		})

		Context("when no plugin is provided", func() {
			It("returns an error noting the plugin name wasn't found", func() {
				_, err := invoke.FindInPath("", multiplePaths)
				Expect(err).To(MatchError("no plugin name provided"))
			})
		})

		Context("when the plugin cannot be found", func() {
			It("returns an error noting the path", func() {
				pathsWithNothing := []string{anotherTempDir}
				_, err := invoke.FindInPath(pluginName, pathsWithNothing)
				Expect(err).To(MatchError(fmt.Sprintf("failed to find plugin %q in path %s", pluginName, pathsWithNothing)))
			})
		})
	})
})
