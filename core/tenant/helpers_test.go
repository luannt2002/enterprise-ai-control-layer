package tenant

import "github.com/luannt2002/enterprise-ai-control-layer/data/normalization"

func normalizationOptions(rt *Runtime) normalization.Options {
	return normalization.Options{Now: rt.Now()}
}
