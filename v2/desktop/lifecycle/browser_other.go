//go:build !windows && !darwin

package lifecycle

func OpenBrowser(string) error {
	return nil
}
