module github.com/matterpoll/matterpoll

go 1.12

require (
	bou.ke/monkey v1.0.1
	github.com/blang/semver v3.6.1+incompatible
	github.com/gorilla/mux v1.7.2
	github.com/mattermost/mattermost-server v5.14.0+incompatible
	github.com/nicksnyder/go-i18n/v2 v2.0.2
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.4.0
	golang.org/x/text v0.3.2
)

// Workaround for https://github.com/golang/go/issues/30831 and fallout.
replace github.com/golang/lint => github.com/golang/lint v0.0.0-20190227174305-8f45f776aaf1
