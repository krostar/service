package netservice

import (
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"sync"
	"testing"

	"gotest.tools/v3/assert"
)

func setupSystemdEnv(t *testing.T, configure func(t *testing.T)) {
	a, fa := os.LookupEnv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey)
	b, fb := os.LookupEnv(_systemdSocketActivationEnvExpectedProgramIDKey)
	c, fc := os.LookupEnv(_systemdSocketActivationEnvListenFDNamesKey)
	assert.Check(t, os.Unsetenv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey))
	assert.Check(t, os.Unsetenv(_systemdSocketActivationEnvExpectedProgramIDKey))
	assert.Check(t, os.Unsetenv(_systemdSocketActivationEnvListenFDNamesKey))

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
			assert.Assert(t, err)
			return f
		}

		for range fdsLen {
			socket := localPath()
			sockets = append(sockets, socket.Name())
			assert.Check(t, socket.Close())
			assert.Check(t, os.Remove(socket.Name()))
		}
	}

	t.Cleanup(func() { _systemdTestModeFDMap = nil })
	_systemdTestModeFDMap = make(map[int]int)

	for i, socket := range sockets {
		l, err := net.Listen("unix", socket)
		assert.Assert(t, err)
		t.Cleanup(func() { _ = l.Close() })

		f, err := (l.(*net.UnixListener)).File()
		assert.Assert(t, err)

		_systemdTestModeFDMap[_systemdSocketActivationListenFDSStart+i] = int(f.Fd())

		if breakFD {
			assert.Check(t, f.Close())
		} else {
			t.Cleanup(func() { _ = f.Close() })
		}
	}
}

func Test_GetSystemdFileDescriptors(t *testing.T) {
	for name, test := range map[string]struct {
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
			emulateSystemdProvidingFileDescriptors(t, test.expectedFilesLen, false)
			setupSystemdEnv(t, func(t *testing.T) {
				for k, v := range test.env {
					t.Setenv(k, v)
				}
			})

			files, provided, err := GetSystemdFileDescriptors(test.unsetEnv)

			// check if env is unset
			for k := range test.env {
				_, found := os.LookupEnv(k)
				assert.Check(t, found != test.unsetEnv, k)
			}

			// check errors
			if test.expectedErrorContains != "" {
				assert.ErrorContains(t, err, test.expectedErrorContains)
			} else {
				assert.NilError(t, err)
			}

			assert.Check(t, test.expectedNotProvided != provided)

			if provided {
				// checks files
				assert.Equal(t, test.expectedFilesLen, len(files))
				for i, f := range files {
					assert.Check(t, int(f.Fd()) == _systemdTestModeFDMap[_systemdSocketActivationListenFDSStart+i])
					assert.Check(t, f.Name() == test.expectedFilesName[i])
				}
			}

			for _, file := range files {
				assert.Check(t, file.Close())
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
			assert.ErrorContains(t, err, "unable to retrieve file descriptors")
			assert.Check(t, listeners == nil)
		})

		t.Run("unable to get listener from file", func(t *testing.T) {
			emulateSystemdProvidingFileDescriptors(t, 1, true)
			setupSystemdEnv(t, func(t *testing.T) {
				t.Setenv(_systemdSocketActivationEnvExpectedProgramIDKey, strconv.Itoa(os.Getpid()))
				t.Setenv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey, "1")
			})
			listeners, err := GetSystemdListeners(false)
			assert.ErrorContains(t, err, "unable to create listener from file descriptor")
			assert.Check(t, listeners == nil)
		})
	})

	t.Run("ok", func(t *testing.T) {
		t.Run("without provided fds", func(t *testing.T) {
			setupSystemdEnv(t, nil)
			listeners, err := GetSystemdListeners(false)
			assert.Check(t, err)
			assert.Check(t, listeners == nil)
		})

		t.Run("with 0 provided fd", func(t *testing.T) {
			setupSystemdEnv(t, func(t *testing.T) {
				t.Setenv(_systemdSocketActivationEnvExpectedProgramIDKey, strconv.Itoa(os.Getpid()))
				t.Setenv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey, "0")
			})
			listeners, err := GetSystemdListeners(false)
			assert.Check(t, err)
			assert.Check(t, listeners != nil)
			assert.Check(t, len(listeners) == 0)
		})

		t.Run("with provided fd", func(t *testing.T) {
			emulateSystemdProvidingFileDescriptors(t, 1, false)
			setupSystemdEnv(t, func(t *testing.T) {
				t.Setenv(_systemdSocketActivationEnvExpectedProgramIDKey, strconv.Itoa(os.Getpid()))
				t.Setenv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey, "1")
			})

			proxies, err := GetSystemdListeners(false)
			assert.Check(t, err)
			assert.Check(t, len(proxies) == 1)

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()

				c1, errAccept := proxies[0].Accept()
				assert.Check(t, errAccept)

				read, errRead := io.ReadAll(c1)
				assert.Check(t, errRead)
				assert.Equal(t, string(read), "hello world")
				assert.Check(t, c1.Close())
			}()

			c2, err := net.Dial(proxies[0].Addr().Network(), proxies[0].Addr().String())
			assert.Assert(t, err)

			_, err = c2.Write([]byte("hello world"))
			assert.Check(t, err)
			assert.Check(t, c2.Close())

			wg.Wait()

			assert.Check(t, proxies[0].Close())
		})
	})
}
