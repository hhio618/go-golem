module github.com/hhio618/go-golem

go 1.16

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/fatih/structs v1.1.0
	github.com/go-kit/kit v0.10.0
	github.com/hhio618/ya-go-client/ya-activity v0.0.0-00010101000000-000000000000
	github.com/hhio618/ya-go-client/ya-payment v0.0.0-00010101000000-000000000000
	github.com/hhio618/ya-go-client/ya-market v0.0.0-00010101000000-000000000000
	github.com/pkg/errors v0.8.1
	github.com/pmezard/go-difflib v1.0.0
	github.com/r3labs/sse/v2 v2.3.2
	github.com/shopspring/decimal v1.2.0
	github.com/stretchr/testify v1.6.1 // indirect
	go.uber.org/goleak v1.1.10
)

replace github.com/hhio618/ya-go-client/ya-activity => /home/ox26a/Projects/ya-go-client/pkg/ya-activity

replace github.com/hhio618/ya-go-client/ya-payment => /home/ox26a/Projects/ya-go-client/pkg/ya-payment

replace github.com/hhio618/ya-go-client/ya-market => /home/ox26a/Projects/ya-go-client/pkg/ya-market
