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

/*
GetSystemdFileDescriptors retrieves the file descriptors passed by the init system as part of the socket-based activation logic.
It should behave as described [here](https://www.freedesktop.org/software/systemd/man/latest/sd_listen_fds.html).

If the unsetEnvironment parameter is true, the first call to GetSystemdFileDescriptors will unset the
$LISTEN_FDS, $LISTEN_PID, and $LISTEN_FDNAMES environment variables before returning (regardless of whether
the function call itself succeeded or not). Further calls will then fail, but the variables are no longer
inherited by child processes.
*/
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
	if !(pidEnvExists && fdsLenEnvExists) {
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
			if len(fdsNames[i]) == 0 {
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

/*
GetSystemdListeners returns net.Listener for each matching socket fd passed to this process.
The order of the file descriptors is preserved in the returned slice.
See:
- https://0pointer.de/blog/projects/socket-activation.html
- https://0pointer.net/blog/walkthrough-for-portable-services-in-go.html
*/
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
