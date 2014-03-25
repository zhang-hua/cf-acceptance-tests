package helpers

import (
	"github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

var config = LoadConfig()

var AdminUserContext = cf.NewUserContext(
	config.ApiEndpoint,
	config.AdminUser,
	config.AdminPassword,
	config.Org,
	config.Space,
	config.LoginFlags)

var RegularUserContext = cf.NewUserContext(
	config.ApiEndpoint,
	config.User,
	config.Password,
	config.Org,
	config.Space,
	config.LoginFlags)

