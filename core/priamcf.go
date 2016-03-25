package core
import (
	"io"
)

// publish command
func PriamCfPublish(trace bool, configFile, manifestFile string, errw, outw io.Writer) {

	// when cf execs a plugin it sets stdin and stdout but not stderr
	log := &logr{traceOn: trace, errw: errw, outw: outw}

	if cfg := newAppConfig(log, configFile); cfg != nil {
		if ctx := initCtx(cfg, true); ctx != nil {
			publishApps(ctx, manifestFile)
		}
	}
}

