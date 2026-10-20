package tenant

import "github.com/luannt2002/enterprise-ai-systems/data/normalization"

func normalizationOptions(rt *Runtime) normalization.Options {
	return normalization.Options{Now: rt.Now()}
}
