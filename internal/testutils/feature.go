package testutils

import (
	"bytes"
	"errors"
	"sync"
	"testing"

	"github.com/cilium/ebpf/internal"
	"github.com/cilium/ebpf/internal/unix"
)

var (
	kernelVersionOnce sync.Once
	kernelVersion     internal.Version
)

func mustKernelVersion() internal.Version {
	kernelVersionOnce.Do(func() {
		var uname unix.Utsname
		err := unix.Uname(&uname)
		if err != nil {
			panic(err)
		}

		end := bytes.IndexByte(uname.Release[:], 0)
		release := string(uname.Release[:end])

		kernelVersion, err = internal.NewVersion(release)
		if err != nil {
			panic(err)
		}
	})

	return kernelVersion
}

func CheckFeatureTest(t *testing.T, fn func() error) {
	t.Helper()

	err := fn()
	if err == nil {
		return
	}

	var ufe *internal.UnsupportedFeatureError
	if errors.As(err, &ufe) {
		checkKernelVersion(t, ufe)
	} else {
		t.Error("Feature test failed:", err)
	}
}

func SkipIfNotSupported(tb testing.TB, err error) {
	var ufe *internal.UnsupportedFeatureError
	if errors.As(err, &ufe) {
		checkKernelVersion(tb, ufe)
		tb.Skip(ufe.Error())
	}
	if errors.Is(err, internal.ErrNotSupported) {
		tb.Skip(err.Error())
	}
}

func checkKernelVersion(tb testing.TB, ufe *internal.UnsupportedFeatureError) {
	if ufe.MinimumVersion.Unspecified() {
		return
	}

	kernelVersion := mustKernelVersion()
	if ufe.MinimumVersion.Less(kernelVersion) {
		tb.Helper()
		tb.Fatalf("Feature '%s' isn't supported even though kernel %s is newer than %s",
			ufe.Name, kernelVersion, ufe.MinimumVersion)
	}
}

func SkipOnOldKernel(tb testing.TB, minVersion, feature string) {
	tb.Helper()

	minv, err := internal.NewVersion(minVersion)
	if err != nil {
		tb.Fatalf("Invalid version %s: %s", minVersion, err)
	}

	if mustKernelVersion().Less(minv) {
		tb.Skipf("Test requires at least kernel %s (due to missing %s)", minv, feature)
	}
}
