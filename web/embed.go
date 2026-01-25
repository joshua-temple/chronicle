package web

import "embed"

//go:embed dist/*
var WebFS embed.FS
