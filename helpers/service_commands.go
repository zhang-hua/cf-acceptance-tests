package helpers

import (
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"time"
	"encoding/json"
	"errors"
)

func MakePlanPublic(url string) {
	jsonMap := make(map[string]bool)
	jsonMap["public"] = true
	planJson, _ := json.Marshal(jsonMap)
	Expect(Cf("curl", url, "-X", "PUT", "-d", string(planJson))).To(ExitWithTimeout(0, 5*time.Second))
}

type RequireExpectation struct {
	actual interface{}
}

func Require(actual interface{}) RequireExpectation {
	return RequireExpectation {
		actual: actual,
	}
}

func (e RequireExpectation) To(matcher OmegaMatcher, optionalDescription ...interface{}) {
	result := Expect(e.actual).To(matcher, optionalDescription...)
	if !result {
		panic(errors.New("Failed Expectation"))
	}
}

func Recover() {

}
