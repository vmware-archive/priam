package core

import (
	"io"
)

// publish command
func PriamCfPublish(trace bool, configFile, manifestFile string, errw, outw io.Writer) {

	log := &logr{traceOn: trace, errw: errw, outw: outw}

	if cfg := newAppConfig(log, configFile); cfg != nil {
		if ctx := initCtx(cfg, true); ctx != nil {
			publishApps(ctx, manifestFile)
		}
	}
}
