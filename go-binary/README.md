# Go Templater


## Requirements

- kubectl v1.32.1 or higher
- helm 3.17.0 or higher
- git

## Prerequisites
### Setup pre-commit hooks
Pre-commit is used to enforce coding standards (YAML-Intendation, Coding Style...).  
When you commit your changes to a branch, pre-commit-hook runs scripts defined in _.pre-commit-config.yaml_.
Changing the pre-commit config effects everyone who is using this repository.
1. Install pre-commit: https://pre-commit.com/#install
2. Open a terminal in this repository
3. Run ```pre-commit install --install-hooks``` to set up the git hook scripts.

Pre-commit will now check your changes when you commit.  
Alternatively you can set up git to automatically install hooks in repositories which use pre-commit.
https://pre-commit.com/#automatically-enabling-pre-commit-on-repositories  
https://pre-commit.com/#pre-commit-init-templatedir

#### Debug pre-commit
When running pre-commit it will output to (git) console. You can see passed/failed tests and the relevant files.  
Some tests will change your commit on the fly.
In that case pre-commit will still report failure and you just need run commit a second time.
#### Disable tests
Here is how to do it: https://pre-commit.com/#temporarily-disabling-hooks  
`SKIP=flake8 git commit -m "foo"` will disable the test "flake8".

## Folder structure

```
.
├── README.md
├── config.schema.json
├── go.mod
├── go.sum
├── main.go
└── templates
```



## Build go binary

```
go build -o config-template .
```

or use the makefile

or use goreleaser with:

```
goreleaser release --clean --skip=publish --config .goreleaser.yml
```
