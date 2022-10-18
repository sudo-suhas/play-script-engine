.PHONY: all
all:fmt lint test

# HELP sourced from https://gist.github.com/prwhite/8168133

# Add help text after each target name starting with '\#\#'
# A category can be added with @category

HELP_FUNC = \
    %help; \
    while(<>) { \
        if(/^([a-z0-9_-]+):.*\#\#(?:@(\w+))?\s(.*)$$/) { \
            push(@{$$help{$$2}}, [$$1, $$3]); \
        } \
    }; \
    print "usage: make [target]\n\n"; \
    for ( sort keys %help ) { \
        print "$$_:\n"; \
        printf("  %-30s %s\n", $$_->[0], $$_->[1]) for @{$$help{$$_}}; \
        print "\n"; \
    }

help:           ##@help show this help
	@perl -e '$(HELP_FUNC)' $(MAKEFILE_LIST)


# DEV SETUP #############

install-gotest:
	go install github.com/rakyll/gotest@latest

install-formatters: ##@dev_setup install gci and gofumpt code formatters
	go install github.com/daixiang0/gci@v0.8.0
	go install mvdan.cc/gofumpt@latest

install-linter: ##@dev_setup install golangci-lint into ./bin/
	bin/install-linter

setup: install-formatters install-linter install-gotest ##@dev_setup setup lint

# LINT #############

imports: install-formatters ##@lint does a goimports
	# Use grep to hide noisy command output
	gci write ./ --section standard --section default --section "Prefix(github.com/sudo-suhas/play-script-engine)" --skip-generated

fmt: imports##@lint does a go fmt (stricter variant)
	gofumpt -l -w -extra .

lint: install-linter ##@lint lint source
	./bin/golangci-lint --config=".golangci-prod.toml" --max-same-issues=0 --max-issues-per-linter=0 run

# TESTS #############

test: install-gotest ##@tests run tests
	gotest -p=1 -mod=readonly ./...
