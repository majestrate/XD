package version

import "fmt"

const Name = "XD"

var Major = "0"

var Minor = "3"

var Patch = "3"

var Git string

func Version() string {
	v := fmt.Sprintf("%s-%s.%s.%s", Name, Major, Minor, Patch)
	if len(Git) > 0 {
		v += fmt.Sprintf("-%s", Git)
	}
	return v
}
