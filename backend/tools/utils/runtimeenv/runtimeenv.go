package runtimeenv

import "os"

const (
	Kubernetes = "kubernetes"
	Docker     = "docker"
	Local      = "local"
)

func RuntimeEnvironment() string {
	if env := os.Getenv("RUNTIME_ENV"); env != "" {
		return env
	}
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		return Kubernetes
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return Docker
	}
	return Local
}

