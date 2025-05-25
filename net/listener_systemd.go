package netservice

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"

	"go.uber.org/multierr"
)

const ( // see https://www.freedesktop.org/software/systemd/man/latest/sd_listen_fds.html
	_systemdSocketActivationListenFDSStart                = 3 // equivalent to the "#define SD_LISTEN_FDS_START 3" of sd_listen_fds
	_systemdSocketActivationEnvNumberOfFileDescriptorsKey = "LISTEN_FDS"
	_systemdSocketActivationEnvExpectedProgramIDKey       = "LISTEN_PID"
	_systemdSocketActivationEnvListenFDNamesKey           = "LISTEN_FDNAMES"
)

var _systemdTestModeFDMap map[int]int //nolint:gochecknoglobals // we need this variable for testing purposes as there is no way to provide file descriptors that exist in our tests otherwise

// GetSystemdFileDescriptors retrieves file descriptors passed by systemd for socket activation.
//
// This function implements the systemd socket activation protocol as described in:
// https://www.freedesktop.org/software/systemd/man/latest/sd_listen_fds.html
//
// It reads environment variables set by systemd and returns the file descriptors
// that were pre-bound and passed to this process for socket activation.
//
// If unsetEnvironment is true, the systemd environment variables will be cleared after reading them.
// This prevents child processes from inheriting these variables and is recommended for
// production use to maintain proper systemd semantics.
//
// Environment variables read:
//   - LISTEN_PID: must match current process PID
//   - LISTEN_FDS: number of file descriptors being passed
//   - LISTEN_FDNAMES: optional colon-separated names for the descriptors
//
// If systemd socket activation is not active (environment variables not set):
// returns (nil, false, nil) - no error, but no file descriptors available
//
// If systemd socket activation is detected but invalid:
// returns (nil, true, error) - indicates systemd activation was attempted but failed
//
// If successful:
//   - returns ([]*os.File, true, nil) - file descriptors ready for use
//   - sets CloseOnExec on all file descriptors to prevent inheritance
//   - optionally unsets environment variables if unsetEnvironment is true
func GetSystemdFileDescriptors(unsetEnvironment bool) ([]*os.File, bool, error) {
	if unsetEnvironment {
		//nolint:errcheck // regardless of result we want to try to unset those env variables
		defer func() {
			_ = os.Unsetenv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey)
			_ = os.Unsetenv(_systemdSocketActivationEnvExpectedProgramIDKey)
			_ = os.Unsetenv(_systemdSocketActivationEnvListenFDNamesKey)
		}()
	}

	rawPID, pidEnvExists := os.LookupEnv(_systemdSocketActivationEnvExpectedProgramIDKey)
	rawFDsLen, fdsLenEnvExists := os.LookupEnv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey)
	if !pidEnvExists || !fdsLenEnvExists {
		return nil, false, nil
	}

	{ // checks whether the provided FDS are for this program
		pid, err := strconv.Atoi(rawPID)
		if err != nil {
			return nil, true, fmt.Errorf("invalid env %s format, expected integer: %v", _systemdSocketActivationEnvExpectedProgramIDKey, err)
		}

		if cpid := os.Getpid(); pid != cpid {
			return nil, true, fmt.Errorf("LISTEN_PID returned an unexpected pid (%d != %d)", cpid, pid)
		}
	}

	fdsLen, err := strconv.Atoi(rawFDsLen)
	if err != nil {
		return nil, true, fmt.Errorf("invalid env %s format, expected integer: %v", _systemdSocketActivationEnvNumberOfFileDescriptorsKey, err)
	}

	var fdsNames []string
	{ // names are optional
		if rawFDNames, exists := os.LookupEnv(_systemdSocketActivationEnvListenFDNamesKey); exists {
			fdsNames = strings.Split(rawFDNames, ":")
			if fdsNamesLen := len(fdsNames); fdsNamesLen > 0 && fdsNamesLen != fdsLen {
				return nil, true, fmt.Errorf("unexpected number of fds names (%d != %d)", fdsNamesLen, fdsLen)
			}
		}

		if len(fdsNames) == 0 {
			fdsNames = make([]string, fdsLen)
		}

		for i := range fdsLen {
			if fdsNames[i] == "" {
				fdsNames[i] = "LISTEN_FD_" + strconv.Itoa(i+_systemdSocketActivationListenFDSStart)
			}
		}
	}

	fds := make([]*os.File, fdsLen)
	for i := range fdsLen {
		fd := i + _systemdSocketActivationListenFDSStart
		if _systemdTestModeFDMap != nil {
			fd = _systemdTestModeFDMap[fd]
		}
		syscall.CloseOnExec(fd) // avoid further inheritance to children of the calling process
		fds[i] = os.NewFile(uintptr(fd), fdsNames[i])
	}

	return fds, true, nil
}

// GetSystemdListeners converts systemd file descriptors to net.Listener instances.
//
// This is the high-level function for systemd socket activation that handles the
// complete process of retrieving file descriptors from systemd and converting
// them to usable net.Listener instances.
//
// Functionalities:
//   - retrieves file descriptors using GetSystemdFileDescriptors
//   - converts each file descriptor to a net.Listener using net.FileListener
//   - maintains the order of file descriptors as provided by systemd
//   - handles cleanup of file descriptors after listener creation
//   - provides proper error handling with resource cleanup
//
// If unsetEnvironment is true, clears systemd environment variables after use.
// The order of listeners matches the order specified in systemd socket configuration:
//
//	[Socket]
//	ListenStream=8080    # listeners[0]
//	ListenStream=8443    # listeners[1]
//	ListenDatagram=9090  # listeners[2]
//
// References:
//   - https://0pointer.de/blog/projects/socket-activation.html
//   - https://0pointer.net/blog/walkthrough-for-portable-services-in-go.html
//   - https://www.freedesktop.org/software/systemd/man/systemd.socket.html
func GetSystemdListeners(unsetEnvironment bool) ([]net.Listener, error) {
	fds, provided, err := GetSystemdFileDescriptors(unsetEnvironment)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve file descriptors: %w", err)
	}

	if !provided {
		return nil, nil
	}

	listeners := make([]net.Listener, len(fds))

	closeAll := func() error {
		var errs []error
		for _, fd := range fds {
			errs = append(errs, fd.Close())
		}
		for _, l := range listeners {
			if l != nil {
				errs = append(errs, l.Close())
			}
		}
		return multierr.Combine(errs...)
	}

	for i, fd := range fds {
		listener, err := net.FileListener(fd)
		if err != nil {
			return nil, fmt.Errorf("unable to create listener from file descriptor %s: %w, closing: %v", fd.Name(), err, closeAll())
		}
		if err := fd.Close(); err != nil {
			return nil, fmt.Errorf("unable to close file descriptor %s after listener creation: %w, closing: %v", fd.Name(), err, closeAll())
		}
		listeners[i] = listener
	}

	return listeners, nil
}
