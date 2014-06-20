package runner

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var CommandInterceptor = func(cmd *exec.Cmd) *exec.Cmd {
	return cmd
}

func Run(executable string, args ...string) *gexec.Session {
	cmd := exec.Command(executable, args...)
	started := time.Now()

	startColor := ""
	endColor := ""
	if !config.DefaultReporterConfig.NoColor {
		startColor = "\x1b[32m"
		endColor = "\x1b[0m"
	}

	if config.DefaultReporterConfig.Verbose {
		fmt.Println("\n", startColor, "> ", strings.Join(cmd.Args, " "), endColor)
	}

	sess, err := gexec.Start(CommandInterceptor(cmd), ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	Eventually(sess, 90).Should(gexec.Exit())

	if config.DefaultReporterConfig.Verbose {
		duration := time.Since(started)
		fmt.Println("\n", startColor, ">> ", strings.Join(cmd.Args, " "), "took", duration.Seconds(), "s", endColor)
	}

	return sess
}

func Curl(args ...string) *gexec.Session {
	args = append([]string{"-s"}, args...)
	return Run("curl", args...)
}
