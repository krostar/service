package netservice

import (
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/krostar/test"
)

//nolint:usetesting // we can't use t.SetEnv because we want to basically unset env, and restore it after test
func setupSystemdEnv(t *testing.T, configure func(t *testing.T)) {
	a, fa := os.LookupEnv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey)
	b, fb := os.LookupEnv(_systemdSocketActivationEnvExpectedProgramIDKey)
	c, fc := os.LookupEnv(_systemdSocketActivationEnvListenFDNamesKey)
	test.Assert(t, os.Unsetenv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey) == nil)
	test.Assert(t, os.Unsetenv(_systemdSocketActivationEnvExpectedProgramIDKey) == nil)
	test.Assert(t, os.Unsetenv(_systemdSocketActivationEnvListenFDNamesKey) == nil)

	if configure != nil {
		configure(t)
	}

	// reset with old values
	t.Cleanup(func() {
		if fa {
			_ = os.Setenv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey, a)
		} else {
			_ = os.Unsetenv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey)
		}
		if fb {
			_ = os.Setenv(_systemdSocketActivationEnvExpectedProgramIDKey, b)
		} else {
			_ = os.Unsetenv(_systemdSocketActivationEnvExpectedProgramIDKey)
		}
		if fc {
			_ = os.Setenv(_systemdSocketActivationEnvListenFDNamesKey, c)
		} else {
			_ = os.Unsetenv(_systemdSocketActivationEnvListenFDNamesKey)
		}
	})
}

func emulateSystemdProvidingFileDescriptors(t *testing.T, fdsLen int, breakFD bool) {
	var sockets []string
	{
		// taken from golang.org/x/net@v0.34.0/nettest/nettest.go:223
		localPath := func() *os.File {
			dir := ""
			if runtime.GOOS == "darwin" {
				dir = "/tmp"
			}
			f, err := os.CreateTemp(dir, "go-nettest")
			test.Require(t, err == nil)
			return f
		}

		for range fdsLen {
			socket := localPath()
			sockets = append(sockets, socket.Name())
			test.Assert(t, socket.Close() == nil)
			test.Assert(t, os.Remove(socket.Name()) == nil)
		}
	}

	t.Cleanup(func() { _systemdTestModeFDMap = nil })
	_systemdTestModeFDMap = make(map[int]int)

	for i, socket := range sockets {
		l, err := net.Listen("unix", socket)
		test.Require(t, err == nil)
		t.Cleanup(func() { _ = l.Close() })

		f, err := (l.(*net.UnixListener)).File()
		test.Require(t, err == nil)

		_systemdTestModeFDMap[_systemdSocketActivationListenFDSStart+i] = int(f.Fd())

		if breakFD {
			test.Assert(t, f.Close() == nil)
		} else {
			t.Cleanup(func() { _ = f.Close() })
		}
	}
}

func Test_GetSystemdFileDescriptors(t *testing.T) {
	for name, tt := range map[string]struct {
		env                   map[string]string
		unsetEnv              bool
		expectedErrorContains string
		expectedFilesLen      int
		expectedFilesName     []string
		expectedNotProvided   bool
	}{
		"env unset": {expectedNotProvided: true},

		"invalid pid format": {
			env: map[string]string{
				"LISTEN_PID": "notint",
				"LISTEN_FDS": "1",
			},
			expectedErrorContains: "invalid env LISTEN_PID format, expected integer",
		},
		"pid for a different program": {
			env: map[string]string{
				"LISTEN_PID": strconv.Itoa(os.Getpid() + 1),
				"LISTEN_FDS": "1",
			},
			expectedErrorContains: "LISTEN_PID returned an unexpected pid",
		},
		"invalid fds len format": {
			env: map[string]string{
				"LISTEN_PID": strconv.Itoa(os.Getpid()),
				"LISTEN_FDS": "notint",
			},
			expectedErrorContains: "invalid env LISTEN_FDS format, expected integer",
		},
		"invalid fds names len": {
			env: map[string]string{
				"LISTEN_PID":     strconv.Itoa(os.Getpid()),
				"LISTEN_FDS":     "2",
				"LISTEN_FDNAMES": "a:b:c:d",
			},
			expectedErrorContains: "unexpected number of fds names (4 != 2)",
		},
		"ok with unset env": {
			env: map[string]string{
				"LISTEN_PID": strconv.Itoa(os.Getpid()),
				"LISTEN_FDS": "1",
			},
			expectedFilesLen:  1,
			expectedFilesName: []string{"LISTEN_FD_3"},
			unsetEnv:          true,
		},
		"ok with no fds names": {
			env: map[string]string{
				"LISTEN_PID": strconv.Itoa(os.Getpid()),
				"LISTEN_FDS": "2",
			},
			expectedFilesLen:  2,
			expectedFilesName: []string{"LISTEN_FD_3", "LISTEN_FD_4"},
		},
		"ok with partial fds names": {
			env: map[string]string{
				"LISTEN_PID":     strconv.Itoa(os.Getpid()),
				"LISTEN_FDS":     "3",
				"LISTEN_FDNAMES": "a::c",
			},
			expectedFilesLen:  3,
			expectedFilesName: []string{"a", "LISTEN_FD_4", "c"},
		},
		"ok with full fds names": {
			env: map[string]string{
				"LISTEN_PID":     strconv.Itoa(os.Getpid()),
				"LISTEN_FDS":     "4",
				"LISTEN_FDNAMES": "a:b:c:d",
			},
			expectedFilesLen:  4,
			expectedFilesName: []string{"a", "b", "c", "d"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			emulateSystemdProvidingFileDescriptors(t, tt.expectedFilesLen, false)
			setupSystemdEnv(t, func(t *testing.T) {
				for k, v := range tt.env {
					t.Setenv(k, v)
				}
			})

			files, provided, err := GetSystemdFileDescriptors(tt.unsetEnv)

			// check if env is unset
			for k := range tt.env {
				_, found := os.LookupEnv(k)
				test.Assert(t, found != tt.unsetEnv)
			}

			// check errors
			if tt.expectedErrorContains != "" {
				test.Assert(t, err != nil && strings.Contains(err.Error(), tt.expectedErrorContains))
			} else {
				test.Assert(t, err == nil)
			}

			test.Assert(t, tt.expectedNotProvided != provided)

			if provided {
				test.Assert(t, len(files) == tt.expectedFilesLen)
				for i, f := range files {
					test.Assert(t, int(f.Fd()) == _systemdTestModeFDMap[_systemdSocketActivationListenFDSStart+i])
					test.Assert(t, f.Name() == tt.expectedFilesName[i])
				}
			}

			for _, file := range files {
				test.Assert(t, file.Close() == nil)
			}
		})
	}
}

func Test_GetSystemdListeners(t *testing.T) {
	t.Run("ko", func(t *testing.T) {
		t.Run("unable to get provided file descriptors", func(t *testing.T) {
			setupSystemdEnv(t, func(t *testing.T) {
				t.Setenv(_systemdSocketActivationEnvExpectedProgramIDKey, "foo")
				t.Setenv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey, "foo")
			})
			listeners, err := GetSystemdListeners(false)
			test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to retrieve file descriptors"))
			test.Assert(t, listeners == nil)
		})

		t.Run("unable to get listener from file", func(t *testing.T) {
			emulateSystemdProvidingFileDescriptors(t, 1, true)
			setupSystemdEnv(t, func(t *testing.T) {
				t.Setenv(_systemdSocketActivationEnvExpectedProgramIDKey, strconv.Itoa(os.Getpid()))
				t.Setenv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey, "1")
			})
			listeners, err := GetSystemdListeners(false)
			test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to create listener from file descriptor"))
			test.Assert(t, listeners == nil)
		})
	})

	t.Run("ok", func(t *testing.T) {
		t.Run("without provided fds", func(t *testing.T) {
			setupSystemdEnv(t, nil)
			listeners, err := GetSystemdListeners(false)
			test.Assert(t, err == nil)
			test.Assert(t, listeners == nil)
		})

		t.Run("with 0 provided fd", func(t *testing.T) {
			setupSystemdEnv(t, func(t *testing.T) {
				t.Setenv(_systemdSocketActivationEnvExpectedProgramIDKey, strconv.Itoa(os.Getpid()))
				t.Setenv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey, "0")
			})
			listeners, err := GetSystemdListeners(false)
			test.Assert(t, err == nil)
			test.Assert(t, listeners != nil)
			test.Assert(t, len(listeners) == 0)
		})

		t.Run("with provided fd", func(t *testing.T) {
			emulateSystemdProvidingFileDescriptors(t, 1, false)
			setupSystemdEnv(t, func(t *testing.T) {
				t.Setenv(_systemdSocketActivationEnvExpectedProgramIDKey, strconv.Itoa(os.Getpid()))
				t.Setenv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey, "1")
			})

			proxies, err := GetSystemdListeners(false)
			test.Assert(t, err == nil)
			test.Assert(t, len(proxies) == 1)

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()

				c1, errAccept := proxies[0].Accept()
				test.Assert(t, errAccept == nil)

				read, errRead := io.ReadAll(c1)
				test.Assert(t, errRead == nil)
				test.Assert(t, string(read) == "hello world")
				test.Assert(t, c1.Close() == nil)
			}()

			c2, err := net.Dial(proxies[0].Addr().Network(), proxies[0].Addr().String())
			test.Require(t, err == nil)

			_, err = c2.Write([]byte("hello world"))
			test.Assert(t, err == nil)
			test.Assert(t, c2.Close() == nil)

			wg.Wait()

			test.Assert(t, proxies[0].Close() == nil)
		})
	})
}
