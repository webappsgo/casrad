package web

import (
	"embed"
)

//go:embed static/css/* static/js/* static/fonts/* static/icons/* static/images/*
var Assets embed.FS

//go:embed templates/*
var Templates embed.FS

//go:embed themes/*
var Themes embed.FS

//go:embed docs/*
var Documentation embed.FS

//go:embed migrations/*
var Migrations embed.FS