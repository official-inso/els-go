// Example: set service/app/env defaults once on the client.
//
// You do NOT need to pass WithServiceName on every call. Set AppSlug,
// ServiceName, DeploymentEnv and AppVersion in Config and the SDK attaches
// them to every entry automatically.
package main

import (
	"errors"
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:        os.Getenv("ELS_API_KEY"),
		AppSlug:       "homedata",     // project / application
		ServiceName:   "sensorloader", // microservice within the project
		DeploymentEnv: "PRODUCTION",
		AppVersion:    os.Getenv("BUILD_VERSION"), // e.g. set by your CI
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// No WithServiceName needed — appSlug/serviceName/env/version are applied.
	client.CaptureError(errors.New("disk almost full"), els.WithURL("/internal/disk"))
	client.CaptureMessage("startup complete", els.LevelInfo)

	client.Flush()
}
