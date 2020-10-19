package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"

	"github.com/pkg/sftp"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

func newForegroundCmd(out io.Writer) *cobra.Command {
	server := server{}

	cmd := &cobra.Command{
		Use:   "foreground",
		Short: "run in the foreground",
		RunE: func(cmd *cobra.Command, args []string) error {
			return server.start()
		},
	}

	cmd.Flags().IntVarP(&server.port, "port", "p", 22, "the port to bind to")
	return cmd
}

type server struct {
	port int
}

func (s *server) start() error {
	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", s.port))
	if err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not retrieve home: %v", err)
	}
	privateBytes, err := ioutil.ReadFile(fmt.Sprintf("%s/.ssh/id_rsa", home))
	if err != nil {
		return fmt.Errorf("failed to load private key: %v", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %v", err)
	}
	config.AddHostKey(private)

	nConn, err := listener.Accept()
	if err != nil {
		return fmt.Errorf("failed to accept incoming connection: %v", err)
	}

	_, chans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		return fmt.Errorf("failed to handshake: %v", err)
	}

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			return fmt.Errorf("could not accept new channel: %v", err)
		}

		go func(in <-chan *ssh.Request) {
			for req := range in {
				ok := false
				switch req.Type {
				case "subsystem":
					if string(req.Payload[4:]) == "sftp" {
						ok = true
					}
				}

				req.Reply(ok, nil)
			}
		}(requests)

		server, err := sftp.NewServer(
			channel,
		)
		if err != nil {
			return fmt.Errorf("could not start newServer: %v", err)
		}
		if err := server.Serve(); err == io.EOF {
			server.Close()
		} else if err != nil {
			return fmt.Errorf("sftp server completed with errors: %v", err)
		}
	}

	return nil
}
