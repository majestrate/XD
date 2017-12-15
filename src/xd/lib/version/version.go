package version

import "fmt"

const Name = "XD"

const Major = "0"

const Minor = "1"

const Patch = "0"

var Git string

func Version() string {
	v := fmt.Sprintf("%s-%s.%s.%s", Name, Major, Minor, Patch)
	if len(Git) > 0 {
		v += fmt.Sprintf("-%s", Git)
	}
	return v
}
