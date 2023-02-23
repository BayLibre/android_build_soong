package bazel

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	os_lib "os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Logs fatal events of BazelProxyServer.
type ServerLogger interface {
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
}

type BazelCmdRequest struct {
	Flags []string
	Env   []string
}

type BazelCmdResponse struct {
	Stdout string
	Stderr string
	Err    error
}

type BazelProxyClient struct {
	outDir string
}

type BazelProxyServer struct {
	logger       ServerLogger
	outDir       string
	workspaceDir string
	done         chan bool
}

func NewBazelProxyClient(outDir string) *BazelProxyClient {
	return &BazelProxyClient{
		outDir: outDir,
	}
}

func unixSocketPath(outDir string) string {
	return filepath.Join(outDir, "bazelsocket.sock")
}

func (b *BazelProxyClient) IssueCommand(req BazelCmdRequest) (resp BazelCmdResponse, err error) {
	d := net.Dialer{Timeout: 1 * time.Second}
	var conn net.Conn
	conn, err = d.Dial("unix", unixSocketPath(b.outDir))
	if err != nil {
		return
	}
	defer conn.Close()

	enc := gob.NewEncoder(conn)
	if err = enc.Encode(req); err != nil {
		return
	}
	dec := gob.NewDecoder(conn)
	err = dec.Decode(&resp)
	return
}

func NewBazelProxyServer(logger ServerLogger, outDir string, workspaceDir string) *BazelProxyServer {
	return &BazelProxyServer{
		logger:       logger,
		outDir:       outDir,
		workspaceDir: workspaceDir,
		done:         make(chan bool),
	}
}

// Start initializes the server unix socket and (in a separate goroutine)
// handles requests on the socket until the server is closed. Returns an error
// if a failure occurs during  initialization, otherwise will log any errors to
// the server's logger.
func (b *BazelProxyServer) Start() error {
	unixSocketAddr := unixSocketPath(b.outDir)
	if err := os_lib.RemoveAll(unixSocketAddr); err != nil {
		return fmt.Errorf("couldn't remove socket '%s': %s", unixSocketAddr, err)
	}
	listener, err := net.Listen("unix", unixSocketAddr)

	if err != nil {
		return fmt.Errorf("error listening on socket '%s': %s", unixSocketAddr, err)
	}

	handleConnection := func(conn net.Conn) error {
		defer conn.Close()

		dec := gob.NewDecoder(conn)
		var req BazelCmdRequest
		if err := dec.Decode(&req); err != nil {
			return fmt.Errorf("Error decoding request: %s", err)
		}

		bazelCmd := exec.Command("./build/bazel/bin/bazel", req.Flags...)
		bazelCmd.Dir = b.workspaceDir
		bazelCmd.Env = req.Env

		stderr := &bytes.Buffer{}
		bazelCmd.Stderr = stderr
		var stdout string
		var bazelErr error

		if output, err := bazelCmd.Output(); err != nil {
			bazelErr = fmt.Errorf("bazel command failed: %s\n---command---\n%s\n---env---\n%s\n---stderr---\n%s---",
				err, bazelCmd, strings.Join(bazelCmd.Env, "\n"), stderr)
		} else {
			stdout = string(output)
		}

		resp := BazelCmdResponse{stdout, string(stderr.Bytes()), bazelErr}
		enc := gob.NewEncoder(conn)
		if err := enc.Encode(&resp); err != nil {
			return fmt.Errorf("Error encoding response: %s", err)
		}
		return nil
	}

	handlerRoutine := func() error {
		for {
			listener.(*net.UnixListener).SetDeadline(time.Now().Add(time.Second))
			conn, err := listener.Accept()

			select {
			case <-b.done:
				return nil
			default:
			}

			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					// Timeout is normal and expected while waiting for client to establish
					// a connection.
					continue
				} else {
					b.logger.Fatalf("Listener error: %s", err)
				}
			}

			err = handleConnection(conn)
			if err != nil {
				b.logger.Fatal(err)
			}
		}
	}

	go handlerRoutine()
	return nil
}

// Closes the server. This will stop the server from listening for additional requests.
func (b *BazelProxyServer) Close() {
	b.done <- true
}
