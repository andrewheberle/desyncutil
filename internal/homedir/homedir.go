// This package is a replacement for [github.com/mitchellh/go-homedir.Dir] as
// [github.com/mitchellh/go-homedir] is no longer maintained.
package homedir

import "os"

// Dir is used to replace [github.com/mitchellh/go-homedir.Dir] as this is no
// longer maintained and the functionality now exists in [os]
func Dir() (string, error) {
	return os.UserHomeDir()
}
