package service

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modwtsapi32 *windows.LazyDLL = windows.NewLazySystemDLL("wtsapi32.dll")
	modkernel32 *windows.LazyDLL = windows.NewLazySystemDLL("kernel32.dll")
	modadvapi32 *windows.LazyDLL = windows.NewLazySystemDLL("advapi32.dll")
	moduserenv  *windows.LazyDLL = windows.NewLazySystemDLL("userenv.dll")

	procWTSEnumerateSessionsW        *windows.LazyProc = modwtsapi32.NewProc("WTSEnumerateSessionsW")
	procWTSGetActiveConsoleSessionId *windows.LazyProc = modkernel32.NewProc("WTSGetActiveConsoleSessionId")
	procWTSQueryUserToken            *windows.LazyProc = modwtsapi32.NewProc("WTSQueryUserToken")
	procDuplicateTokenEx             *windows.LazyProc = modadvapi32.NewProc("DuplicateTokenEx")
	procCreateEnvironmentBlock       *windows.LazyProc = moduserenv.NewProc("CreateEnvironmentBlock")
	procCreateProcessAsUser          *windows.LazyProc = modadvapi32.NewProc("CreateProcessAsUserW")
)
var (
	reader *myReader = nil

	proc_stdin_wr windows.Handle
)

const (
	WTS_CURRENT_SERVER_HANDLE uintptr = 0
)

type WTS_CONNECTSTATE_CLASS int

const (
	WTSActive WTS_CONNECTSTATE_CLASS = iota
	WTSConnected
	WTSConnectQuery
	WTSShadow
	WTSDisconnected
	WTSIdle
	WTSListen
	WTSReset
	WTSDown
	WTSInit
)

type SECURITY_IMPERSONATION_LEVEL int

const (
	SecurityAnonymous SECURITY_IMPERSONATION_LEVEL = iota
	SecurityIdentification
	SecurityImpersonation
	SecurityDelegation
)

type TOKEN_TYPE int

const (
	TokenPrimary TOKEN_TYPE = iota + 1
	TokenImpersonazion
)

type SW int

const (
	SW_HIDE            SW = 0
	SW_SHOWNORMAL         = 1
	SW_NORMAL             = 1
	SW_SHOWMINIMIZED      = 2
	SW_SHOWMAXIMIZED      = 3
	SW_MAXIMIZE           = 3
	SW_SHOWNOACTIVATE     = 4
	SW_SHOW               = 5
	SW_MINIMIZE           = 6
	SW_SHOWMINNOACTIVE    = 7
	SW_SHOWNA             = 8
	SW_RESTORE            = 9
	SW_SHOWDEFAULT        = 10
	SW_MAX                = 1
)

type WTS_SESSION_INFO struct {
	SessionID      windows.Handle
	WinStationName *uint16
	State          WTS_CONNECTSTATE_CLASS
}

const (
	CREATE_UNICODE_ENVIRONMENT uint32 = 0x00000400
	CREATE_NO_WINDOW                  = 0x08000000
	CREATE_NEW_CONSOLE                = 0x00000010
)

// GetCurrentUserSessionId will attempt to resolve
// the session ID of the user currently active on
// the system.
func GetCurrentUserSessionId() (windows.Handle, error) {
	sessionList, err := WTSEnumerateSessions()
	if err != nil {
		return 0xFFFFFFFF, fmt.Errorf("get current user session token: %s", err)
	}

	for i := range sessionList {
		if sessionList[i].State == WTSActive {
			return sessionList[i].SessionID, nil
		}
	}

	if sessionId, _, err := procWTSGetActiveConsoleSessionId.Call(); sessionId == 0xFFFFFFFF {
		return 0xFFFFFFFF, fmt.Errorf("get current user session token: call native WTSGetActiveConsoleSessionId: %s", err)
	} else {
		return windows.Handle(sessionId), nil
	}
}

// WTSEnumerateSession will call the native
// version for Windows and parse the result
// to a Golang friendly version
func WTSEnumerateSessions() ([]*WTS_SESSION_INFO, error) {
	var (
		sessionInformation windows.Handle      = windows.Handle(0)
		sessionCount       int                 = 0
		sessionList        []*WTS_SESSION_INFO = make([]*WTS_SESSION_INFO, 0)
	)

	if returnCode, _, err := procWTSEnumerateSessionsW.Call(WTS_CURRENT_SERVER_HANDLE, 0, 1, uintptr(unsafe.Pointer(&sessionInformation)), uintptr(unsafe.Pointer(&sessionCount))); returnCode == 0 {
		return nil, fmt.Errorf("call native WTSEnumerateSessionsW: %s", err)
	}

	structSize := unsafe.Sizeof(WTS_SESSION_INFO{})
	current := uintptr(sessionInformation)
	for i := 0; i < sessionCount; i++ {
		sessionList = append(sessionList, (*WTS_SESSION_INFO)(unsafe.Pointer(current)))
		current += structSize
	}

	return sessionList, nil
}

// DuplicateUserTokenFromSessionID will attempt
// to duplicate the user token for the user logged
// into the provided session ID
func DuplicateUserTokenFromSessionID(sessionId windows.Handle) (windows.Token, error) {
	var (
		impersonationToken windows.Handle = 0
		userToken          windows.Token  = 0
	)

	if returnCode, _, err := procWTSQueryUserToken.Call(uintptr(sessionId), uintptr(unsafe.Pointer(&impersonationToken))); returnCode == 0 {
		return 0xFFFFFFFF, fmt.Errorf("call native WTSQueryUserToken: %s", err)
	}

	if returnCode, _, err := procDuplicateTokenEx.Call(uintptr(impersonationToken), 0, 0, uintptr(SecurityImpersonation), uintptr(TokenPrimary), uintptr(unsafe.Pointer(&userToken))); returnCode == 0 {
		return 0xFFFFFFFF, fmt.Errorf("call native DuplicateTokenEx: %s", err)
	}

	if err := windows.CloseHandle(impersonationToken); err != nil {
		return 0xFFFFFFFF, fmt.Errorf("close windows handle used for token duplication: %s", err)
	}

	return userToken, nil
}

func StartProcessAsCurrentUser(appPath, cmdLine, workDir string) (*windows.ProcessInformation, error) {
	var (
		sessionId windows.Handle
		userToken windows.Token
		envInfo   windows.Handle

		startupInfo windows.StartupInfo
		processInfo windows.ProcessInformation

		commandLine uintptr = 0
		workingDir  uintptr = 0

		err error
	)

	if sessionId, err = GetCurrentUserSessionId(); err != nil {
		return nil, err
	}

	if userToken, err = DuplicateUserTokenFromSessionID(sessionId); err != nil {
		return nil, fmt.Errorf("get duplicate user token for current user session: %s", err)
	}

	if returnCode, _, err := procCreateEnvironmentBlock.Call(uintptr(unsafe.Pointer(&envInfo)), uintptr(userToken), 0); returnCode == 0 {
		return nil, fmt.Errorf("create environment details for process: %s", err)
	}

	creationFlags := CREATE_UNICODE_ENVIRONMENT | CREATE_NO_WINDOW

	if len(cmdLine) > 0 {
		commandLine = uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(cmdLine)))
	}
	if len(workDir) > 0 {
		workingDir = uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(workDir)))
	}

	var stdin_rd windows.Handle
	var stdin_wr windows.Handle
	var stdout_rd windows.Handle
	var stdout_wr windows.Handle
	var sa windows.SecurityAttributes
	sa.InheritHandle = 1
	sa.SecurityDescriptor = nil
	err = windows.CreatePipe(&stdout_rd, &stdout_wr, &sa, 0)
	if err != nil {
		return nil, err
	}
	err = windows.SetHandleInformation(stdout_rd, windows.HANDLE_FLAG_INHERIT, 0)
	if err != nil {
		return nil, err
	}
	err = windows.CreatePipe(&stdin_rd, &stdin_wr, &sa, 0)
	if err != nil {
		return nil, err
	}
	err = windows.SetHandleInformation(stdin_wr, windows.HANDLE_FLAG_INHERIT, 0)
	if err != nil {
		return nil, err
	}

	proc_stdin_wr = stdin_wr
	startupInfo.ShowWindow = uint16(SW_HIDE)
	startupInfo.Desktop = windows.StringToUTF16Ptr("winsta0\\default")
	startupInfo.StdErr = stdout_wr
	startupInfo.StdOutput = stdout_wr
	startupInfo.StdInput = stdin_rd
	startupInfo.Flags = windows.STARTF_USESHOWWINDOW | windows.STARTF_USESTDHANDLES

	if returnCode, _, err := procCreateProcessAsUser.Call(
		uintptr(userToken), uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(appPath))), commandLine, 0, 0, 1,
		uintptr(creationFlags), 0, workingDir, uintptr(unsafe.Pointer(&startupInfo)), uintptr(unsafe.Pointer(&processInfo)),
	); returnCode == 0 {
		return nil, fmt.Errorf("create process as user: %s", err)
	}
	go func() {
		log.Println("Pipe routine")
		for {
			out := make([]byte, 512)
			n, err := windows.Read(stdout_rd, out)
			if err != nil {
				log.Println("Pipe read error", err.Error())
				return
			}
			log.Println("Read ", string(out[:n]), "bytes")
		}
	}()

	go func() {
		_, err = windows.WaitForSingleObject(processInfo.Process, windows.INFINITE)
		if err != nil {
			log.Println("WaitProcess error:" + err.Error())
			return
		}
		var exitCode uint32
		err = windows.GetExitCodeProcess(processInfo.Process, &exitCode)
		if err != nil {
			log.Println("WaitProcess error:" + err.Error())
			return
		}
		log.Println("Proc exited:" + strconv.Itoa(int(exitCode)))

	}()

	return &processInfo, nil
}

func (s *RecordingService) rec_start() error {
	mu.Lock()
	defer mu.Unlock()
	if reader != nil {
		return errors.New("not available")
	}
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	listen, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}
	reader = &myReader{
		sig: make(chan []byte),
	}
	// log.Println("createing log filesss")
	// eo, err := os.OpenFile("c:/ff.log", os.O_RDWR|os.O_CREATE, 0644)
	// if err != nil {
	// 	log.Println("stdout err", err.Error())
	// 	return err
	// }

	// ffs := ff.Input("desktop", ff.KwArgs{"format": "gdigrab", "framerate": 30}).
	// 	Output("c:/rec.mkv").
	// 	GlobalArgs("-progress", "tcp://"+listen.Addr().String()).
	// 	WithOutput(eo).
	// 	WithErrorOutput(eo).
	// 	WithInput(r).
	// 	OverWriteOutput()
	ffs, err := StartProcessAsCurrentUser("C:/windows/system32/ffmpeg.exe", "ffmpeg -f gdigrab -framerate 30 -i desktop c:/rec.mkv -progress tcp://"+listen.Addr().String()+" -y", "")
	if err != nil {
		return err
	}

	log.Println("Create ffmpeg proc:", ffs.ProcessId)
	pid = uint(ffs.ProcessId)

	go func() {
		conn, err := listen.Accept()
		if err != nil {
			return
		}
		go func() {
			log.Println("Cli connected")
			breader := bufio.NewReader(conn)
			for {
				line, _, err := breader.ReadLine()
				if err == nil {
					// log.Println("Read line")
					lines := string(line)
					a := strings.Split(lines, "=")
					if a[0] == "out_time" {
						recirod_time = a[1]
						if status != "recording" {
							// mu.Lock()
							// defer mu.Unlock()
							status = "recording"

						}
					}
				} else {
					log.Println("Read failed", err)
					mu.Lock()
					defer mu.Unlock()
					status = "finish"
					reader = nil
					log.Println("reader nulled", reader)
					return
				}
			}
		}()
	}()
	status = "starting"
	return nil
}
func (s *RecordingService) rec_stop() error {
	log.Println("stop routinrrrrr")
	mu.Lock()
	defer mu.Unlock()
	log.Println("stop start")
	if status == "stopping" {
		return errors.New("stop in queue")
	}
	status = "stopping"
	rout := []byte("q")
	var writed uint32 = 0
	err := windows.WriteFile(proc_stdin_wr, rout, &writed, nil)
	// _, err := StartProcessAsCurrentUser("C:/windows/system32/taskkill.exe", "taskkill /pid "+strconv.FormatUint(uint64(pid), 10), "")
	log.Println(err)
	return err
}

type myReader struct {
	io.Reader
	io.Writer
	sig chan []byte
}

func (rw *myReader) Read(p []byte) (int, error) {
	log.Println("Call read")
	t := <-rw.sig
	log.Println("Read", t)
	if len(t) == 0 {
		log.Println("EOFFF")
		return 0, io.EOF
	}
	n := copy(p, t)
	return n, nil
}
func (rw *myReader) Write(p []byte) (int, error) {
	log.Println("Writing to", p)
	rw.sig <- p
	return len(p), nil
}
