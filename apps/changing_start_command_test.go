package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/pivotal-cf-experimental/cf-test-helpers/generator"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

var _ = Describe("Changing an app's start command", func() {
	var appName string
		
	BeforeEach(func() {
		appName = RandomName()

		Expect(
			Cf(
				"push", appName,
				"-p", doraPath,
				"-d", IntegrationConfig.AppsDomain,
				"-c", "FOO=foo bundle exec rackup config.ru -p $PORT",
			),
		).To(Say("App started"))
	})

	AfterEach(func() {
		Expect(Cf("delete", appName, "-f")).To(Say("OK"))
	})

	It("takes effect after a restart, not requiring a push", func() {
		Eventually(Curling(AppUri(appName, "/env/FOO"))).Should(Say("foo"))

		var response QueryResponse

		ApiRequest("GET", "/v2/apps?q=name:"+appName, &response)

		Expect(response.Resources).To(HaveLen(1))

		appGuid := response.Resources[0].Metadata.Guid

		ApiRequest(
			"PUT",
			"/v2/apps/"+appGuid,
			nil,
			`{"command":"FOO=bar bundle exec rackup config.ru -p $PORT"}`,
		)

		Expect(Cf("stop", appName)).To(Say("OK"))

		Eventually(Curling(AppUri(appName, "/env/FOO"))).Should(Say("404"))

		Expect(Cf("start", appName)).To(Say("App started"))

		Eventually(Curling(AppUri(appName, "/env/FOO"))).Should(Say("bar"))
	})
})
