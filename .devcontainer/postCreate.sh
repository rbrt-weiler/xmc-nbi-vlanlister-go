#!/bin/bash

# Usual VS Code suspects
#go get -u -v github.com/mdempsky/gocode
go get -u -v github.com/stamblerre/gocode
go get -u -v github.com/rogpeppe/godef
go get -u -v golang.org/x/tools/cmd/goimports
go get -u -v golang.org/x/lint/golint
go get -u -v github.com/ramya-rao-a/go-outline
go get -u -v github.com/uudashr/gopkgs/v2/cmd/gopkgs

# Project-specific dependencies
go get -u -v github.com/360EntSecGroup-Skylar/excelize
go get -u -v github.com/joho/godotenv
go get -u -v gitlab.com/rbrt-weiler/go-module-envordef
go get -u -v gitlab.com/rbrt-weiler/go-module-xmcnbiclient
