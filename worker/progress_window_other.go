//go:build !windows

package worker

func ShowProgress(_ string) chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
func SetProgress(_ int, _ string) {}
func CloseProgress()               {}