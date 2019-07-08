package setup

import (
	"fmt"
	"os"
)

func GetEnv(k string, def ...string) string {
	v, ok := os.LookupEnv(k)
	if !ok {
		if len(def) == 0 {
			setupLog.Log(fmt.Sprintf("No such environment key %q", k))
			os.Exit(2)
		}
		return def[0]
	}
	return v
}
