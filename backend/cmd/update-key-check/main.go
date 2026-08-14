// Command update-key-check is a CI probe for the linker-injected update key.
package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/submerge/submerge/backend/internal/updater"
)

func main() {
	key, err := updater.EmbeddedPublicKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(base64.StdEncoding.EncodeToString(key))
}
