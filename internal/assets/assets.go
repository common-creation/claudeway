package assets

import (
	_ "embed"
)

//go:embed Dockerfile
var DockerfileContent string

//go:embed entrypoint.sh
var EntrypointContent string