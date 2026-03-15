package key

import "fmt"

func errShortBuffer(field string) error {
	return fmt.Errorf("buffer too short for %s", field)
}
