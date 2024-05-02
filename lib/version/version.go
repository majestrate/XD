package version

import (
	"fmt"
	"github.com/majestrate/XD/lib/constants"
)

const Name = "XD"

var Major = "0"

var Minor = "4"

var Patch = "6"

var Git string

func Version() string {
	v := fmt.Sprintf("%s-%s.%s.%s", Name, Major, Minor, Patch)
	if len(Git) > 0 && constants.UseGitVersion {
		v += fmt.Sprintf("-%s", Git)
	}
	return v
}
