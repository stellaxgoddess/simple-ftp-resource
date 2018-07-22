package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"

	"github.com/jlaffaye/ftp"
)

type request struct {
	Source     source     `json:"source"`
	Parameters parameters `json:"params"`
}

type source struct {
	Hostname string `json:"host"`
	Username string `json:"user"`
	Password string `json:"password"`
	StartTLS bool   `json:"tls"`
}

type parameters struct {
	LocalPath  string `json:"local"`
	RemotePath string `json:"remote"`
}

func main() {
	log.SetFlags(0)

	if err := run(); err != nil {
		log.Fatalf("Error: %s", err)
	}
}

func run() error {
	req := request{}

	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		return fmt.Errorf("error decoding input: %s", err)
	}

	progName := filepath.Base(os.Args[0])

	basePath := ""
	if len(os.Args) > 1 {
		basePath = os.Args[1]
	}

	switch progName {
	case "check":
		return runCheck(req)
	case "in":
		return runIn(req, basePath)
	case "out":
		return runOut(req, basePath)
	default:
		return fmt.Errorf("os.Args[0] must be one of 'check', 'in', 'out'; got %q", progName)
	}
}

func connect(src source) (*ftp.ServerConn, error) {
	host, _, err := net.SplitHostPort(src.Hostname)
	if err != nil {
		return nil, fmt.Errorf("can not split host and port: %s", err)
	}

	c, err := ftp.Dial(src.Hostname)
	if err != nil {
		return nil, fmt.Errorf("error dialing %q: %s", src.Hostname, err)
	}

	if src.StartTLS {
		config := &tls.Config{
			ServerName: host,
		}

		if err := c.StartTLS(config); err != nil {
			return nil, fmt.Errorf("error upgrading to TLS: %s", err)
		}
	}

	if err := c.Login(src.Username, src.Password); err != nil {
		return nil, fmt.Errorf("error logging in as %q: %s", src.Username, err)
	}

	return c, nil
}

func runCheck(req request) error {
	c, err := connect(req.Source)
	if err != nil {
		return fmt.Errorf("error during connect: %s", err)
	}
	defer c.Quit()

	log.Println("Login successful.")
	fmt.Println("[]")
	return nil
}

func runIn(req request, basePath string) error {
	c, err := connect(req.Source)
	if err != nil {
		return fmt.Errorf("error during connect: %s", err)
	}
	defer c.Quit()

	log.Println("in is not implemented")
	fmt.Println("{}")
	return nil
}

func runOut(req request, basePath string) error {
	params := req.Parameters
	if params.LocalPath == "" {
		return errors.New("local path can not be empty")
	}

	if !strings.HasPrefix(params.RemotePath, "/") {
		return errors.New("remote path must be absolute")
	}

	baseAbs, err := filepath.Abs(basePath)
	if err != nil {
		return fmt.Errorf("can not convert base path to absolute: %s", err)
	}
	log.Printf("Local path: %s", baseAbs)

	if filepath.IsAbs(params.LocalPath) {
		return fmt.Errorf("local path can not be absolute: %s", params.LocalPath)
	}

	localAbs := filepath.Join(baseAbs, params.LocalPath)
	_, err = os.Stat(localAbs)
	switch {
	case os.IsNotExist(err):
		return fmt.Errorf("local path does not exist: %s", localAbs)
	case err != nil:
		return fmt.Errorf("error accessing local path %q: %s", localAbs, err)
	default:
	}

	c, err := connect(req.Source)
	if err != nil {
		return fmt.Errorf("error during connect: %s", err)
	}
	defer c.Quit()

	err = filepath.Walk(localAbs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing %q: %s", path, err)
			return err
		}

		relative := strings.TrimPrefix(path, localAbs)
		remotePath := filepath.Join(params.RemotePath, relative)

		log.Printf("%s -> %s", path, remotePath)
		if remotePath == params.RemotePath {
			// skip base directory
			return nil
		}

		if info.IsDir() {
			err := c.MakeDir(remotePath)
			if protoErr, ok := err.(*textproto.Error); ok {
				if protoErr.Code == 550 && strings.Contains(protoErr.Msg, "exists") {
					return nil
				}
			}

			if err != nil {
				log.Printf("Error creating directory %q: %s", remotePath, err)
				return filepath.SkipDir
			}
			return nil
		}

		remoteSize, err := c.FileSize(remotePath)
		if err == nil && remoteSize == info.Size() {
			log.Printf("Skip file %q", path)
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			log.Printf("Error opening local file %q: %s", path, err)
			return err
		}
		defer file.Close()

		if err := c.Stor(remotePath, file); err != nil {
			log.Printf("Error uploading file %q: %s", remotePath, err)
			return err
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("upload failed: %s", err)
	}

	fmt.Println("{}")
	return nil
}
