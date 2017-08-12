package version

import "fmt"

const Name = "XD"

const Major = "0"

const Minor = "0"

const Patch = "7"

var Git string

func Version() string {
	return fmt.Sprintf("%s-%s.%s.%s%s", Name, Major, Minor, Patch, Git)
}
