package main

import (
	"encoding/json"
	"fmt"
)

var (
	outputJSON  bool
	outputQuiet bool
	outputPlain bool
)

func emitJSON(v any) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("json encode: %w", err)
	}
	fmt.Println(string(out))
	return nil
}
