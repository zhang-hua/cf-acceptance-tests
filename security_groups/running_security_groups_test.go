package security_groups_test

import (
	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"strings"

	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers"
)

var _ = Describe("Security Groups", func() {

	type AppResource struct {
		Metadata struct {
			Url string
		}
	}
	type AppsResponse struct {
		Resources []AppResource
	}

	type Stat struct {
		Stats struct {
			Host string
			Port int
		}
	}
	type StatsResponse map[string]Stat

	type DoraCurlResponse struct {
		Stdout     string
		Stderr     string
		ReturnCode int `json:"return_code"`
	}

	var serverAppName, clientAppName, securityGroupName string

	BeforeEach(func() {
		serverAppName = generator.RandomName()
		clientAppName = generator.RandomName()

		Expect(cf.Cf("push", serverAppName, "-p", helpers.NewAssets().Dora).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("push", clientAppName, "-p", helpers.NewAssets().Dora).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", serverAppName, "-f").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("delete", clientAppName, "-f").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		if securityGroupName != "" {
			cf.AsUser(context.AdminUserContext(), func() {
				Expect(cf.Cf("delete-security-group", securityGroupName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})
		}
	})

	// this test assumes the default running security groups block access to the DEAs
	// the test takes advantage of the fact that the DEA ip address and internal container ip address
	//  are discoverable via the cc api and dora's myip endpoint
	It("allows previously-blocked ip traffic after applying a security group, and re-blocks it when the group is removed", func() {
		// gather app url
		var appsResponse AppsResponse
		cfResponse := cf.Cf("curl", fmt.Sprintf("/v2/apps?q=name:%s", serverAppName)).Wait(DEFAULT_TIMEOUT).Out.Contents()
		json.Unmarshal(cfResponse, &appsResponse)
		serverAppUrl := appsResponse.Resources[0].Metadata.Url

		// gather app stats for dea ip and app port
		var statsResponse StatsResponse
		cfResponse = cf.Cf("curl", fmt.Sprintf("%s/stats", serverAppUrl)).Wait(DEFAULT_TIMEOUT).Out.Contents()
		json.Unmarshal(cfResponse, &statsResponse)
		host := statsResponse["0"].Stats.Host
		port := statsResponse["0"].Stats.Port

		// gather container ip
		curlResponse := helpers.CurlApp(serverAppName, "/myip")
		containerIp := strings.TrimSpace(curlResponse)

		// test app egress rules
		var doraCurlResponse DoraCurlResponse
		curlResponse = helpers.CurlApp(clientAppName, fmt.Sprintf("/curl/%s/%d", host, port))
		json.Unmarshal([]byte(curlResponse), &doraCurlResponse)
		Expect(doraCurlResponse.ReturnCode).ToNot(Equal(0))

		// apply security group
		rules := fmt.Sprintf(
			`[{"destination":"%s","ports":"%d","protocol":"tcp"},
        {"destination":"%s","ports":"%d","protocol":"tcp"}]`,
			host, port, containerIp, port)

		file, _ := ioutil.TempFile(os.TempDir(), "CATS-sg-rules")
		defer os.Remove(file.Name())
		file.WriteString(rules)

		rulesPath := file.Name()
		securityGroupName = fmt.Sprintf("CATS-SG-%s", generator.RandomName())

		cf.AsUser(context.AdminUserContext(), func() {
			Expect(cf.Cf("create-security-group", securityGroupName, rulesPath).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(cf.Cf("bind-security-group", securityGroupName, context.RegularUserContext().Org, context.RegularUserContext().Space).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
		Expect(cf.Cf("restart", clientAppName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		// test app egress rules
		curlResponse = helpers.CurlApp(clientAppName, fmt.Sprintf("/curl/%s/%d", host, port))
		json.Unmarshal([]byte(curlResponse), &doraCurlResponse)
		Expect(doraCurlResponse.ReturnCode).To(Equal(0))

		// unapply security group
		cf.AsUser(context.AdminUserContext(), func() {
			Expect(cf.Cf("unbind-security-group", securityGroupName, context.RegularUserContext().Org, context.RegularUserContext().Space).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
		Expect(cf.Cf("restart", clientAppName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		// test app egress rules
		curlResponse = helpers.CurlApp(clientAppName, fmt.Sprintf("/curl/%s/%d", host, port))
		json.Unmarshal([]byte(curlResponse), &doraCurlResponse)
		Expect(doraCurlResponse.ReturnCode).ToNot(Equal(0))
	})

	FIt("allows or denies traffic during staging based on default staging security rules", func() {
		buildpack := fmt.Sprintf("CATS-SGBP-%s", generator.RandomName())
		testAppName := generator.RandomName()

		// create bp zip
		buildpackZip := fmt.Sprintf("%s/%s.zip", os.TempDir(), buildpack)
		Expect(runner.Run("zip", "-r", buildpackZip, helpers.NewAssets().SecurityGroupBuildpack)).To(Exit(0))
		defer os.Remove(buildpackZip)

		// upload bp
		cf.AsUser(context.AdminUserContext(), func() {
			Expect(cf.Cf("create-buildpack", buildpack, buildpackZip, "999").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

		// push app with public
		// - push app with bp and --no-start
		// - set-env TESTURI host:port
		// - cf start app
		// - cf logs app --recent should contain CURL_EXIT=0
		Expect(cf.Cf("push", testAppName, "-b", buildpack, "-p", helpers.NewAssets().HelloWorld, "--no-start").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("set-env", testAppName, "TESTURI", "www.google.com").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("start", testAppName).Wait(CF_PUSH_TIMEOUT)).To(Exit(1))
		Expect(cf.Cf("logs", testAppName, "--recent").Wait(CF_PUSH_TIMEOUT)).To(ContainSubstring("CURL_EXIT=0"))

		// push app with private
		// - find dea info
		// - push app with bp and --no-start
		// - set-env TESTURI host:port
		// - cf start app
		// - cf logs app --recent should contain CURL_EXIT=not 0 (should CURL_EXIT=, not CURL_EXIT=0
		// Expect(cf.Cf("push", testAppName, "-b", buildpack, "-p", helpers.NewAssets().HelloWorld, "--no-start").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		// Expect(cf.Cf("set-env", testAppName, "TESTURI", privateUri).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		// Expect(cf.Cf("start", testAppName).Wait(CF_PUSH_TIMEOUT)).To(Exit(1))

		// // expectation also needs to say "does contain CURL_EXIT=(not 0)"
		// Expect(cf.Cf("logs", testAppName, "--recent").Wait(CF_PUSH_TIMEOUT)).ToNot(ContainSubstring("CURL_EXIT=0"))

		cf.AsUser(context.AdminUserContext(), func() {
			Expect(cf.Cf("delete-buildpack", buildpack, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
	})

})
