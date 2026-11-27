package upgrade

import (
	"runtime"
	"strings"
)

func Current(kind string) string {
	return kind + "/" + runtime.GOOS + "/" + runtime.GOARCH
}

func FilenameOK(runtimeID, name string) bool {
	parts := strings.Split(runtimeID, "/")
	if len(parts) != 3 || parts[2] == "" || parts[2] == "unknown" || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return false
	}
	kind, osName, arch := parts[0], parts[1], parts[2]
	switch kind {
	case "go":
		base := "mxonepeer-" + osName + "-" + arch
		if osName == "windows" {
			return name == base+".exe"
		}
		return name == base
	case "python":
		return name == "mxonepeer-python-"+osName+"-"+arch+".zip"
	default:
		return false
	}
}
