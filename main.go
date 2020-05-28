package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/jlaffaye/ftp"
)

type request struct {
	Source     *source     `json:"source"`
	Parameters *parameters `json:"params"`
}

type source struct {
	Address  string `json:"address"`
	Username string `json:"username"`
	Password string `json:"password"`
	Filename string `json:"filename"`
}

type parameters struct {
	Path string `json:"path"`
}

func main() {
	log.SetFlags(0)

	if err := run(); err != nil {
		log.Fatalf("Error: %s", err)
	}
}

func (r *request) verify() {
	if r.Source == nil {
		panic("Source not specified")
	}
	if r.Parameters == nil {
		r.Parameters = &parameters{}
		log.Println("WARNING: Parameters not specified!")
	}

	if r.Source.Address == "" {
		panic("'address' field in the source cannot be empty")
	}
	if r.Source.Username == "" {
		panic("'username' field in the source cannot be empty")
	}
	if r.Source.Password == "" {
		panic("'password' field in the source cannot be empty")
	}
	if r.Source.Filename == "" {
		panic("'filename' field in the source cannot be empty")
	}

	if r.Parameters.Path == "" {
		log.Println("WARNING: empty path field in the params!")
		r.Parameters.Path = r.Source.Filename
	}
}

func run() error {
	req := &request{}

	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		return fmt.Errorf("error decoding input: %s", err)
	}

	req.verify()

	basePath := ""
	if len(os.Args) > 1 {
		basePath = os.Args[1]
	}

	progName := filepath.Base(os.Args[0])
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

func connect(src *source) (*ftp.ServerConn, error) {
	c, err := ftp.Dial(src.Address)
	if err != nil {
		return nil, fmt.Errorf("error dialing %q: %s", src.Address, err)
	}

	if err := c.Login(src.Username, src.Password); err != nil {
		return nil, fmt.Errorf("error logging in as %q: %s", src.Username, err)
	}

	return c, nil
}

func runCheck(req *request) error {
	log.Println("Connecting to the server...")
	c, err := connect(req.Source)
	if err != nil {
		return fmt.Errorf("error during connect: %s", err)
	}
	defer c.Quit()

	log.Println("Reading the file...")
	file, err := c.Retr(req.Source.Filename)
	if err != nil {
		log.Println("File does not exist...")
		fmt.Println("[]")
		return nil
	}
	defer file.Close()

	log.Println("Computing file hash...")
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("error reading file: %s", err)
	}

	fmt.Println("[{ \"ref\": \"" + hex.EncodeToString(hasher.Sum(nil)) + "\" }]")
	return nil
}

func runIn(req *request, basePath string) error {
	log.Println("Connecting to the server...")
	c, err := connect(req.Source)
	if err != nil {
		return fmt.Errorf("error during connect: %s", err)
	}
	defer c.Quit()

	log.Println("Opening remote file '" + req.Source.Filename + "'...")
	file, err := c.Retr(req.Source.Filename)
	if err != nil {
		return fmt.Errorf("error opening remote file: %s", err)
	}
	defer file.Close()

	log.Println("Reading remote file...")
	filedata, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("error reading remote file: %s", err)
	}

	log.Println("Computing hash for downloaded content...")
	hasher := sha256.New()
	_, err = hasher.Write(filedata)
	if err != nil {
		return fmt.Errorf("error computing file hash: %s", err)
	}

	log.Println("Writting local file'" + basePath + "/" + req.Parameters.Path + "'...")
	err = ioutil.WriteFile(basePath+"/"+req.Parameters.Path, filedata, 0644)
	if err != nil {
		return fmt.Errorf("error writing file: %s", err)
	}

	fmt.Println("{\"version\": { \"ref\": \"" + hex.EncodeToString(hasher.Sum(nil)) + "\" },\"metadata\":[]}")
	return nil
}

func runOut(req *request, basePath string) error {
	log.Println("Connecting to the server...")
	c, err := connect(req.Source)
	if err != nil {
		return fmt.Errorf("error during connect: %s", err)
	}
	defer c.Quit()

	log.Println("Opening local file...")
	file, err := os.Open(basePath + "/" + req.Parameters.Path)
	if err != nil {
		return fmt.Errorf("error opening local file: %s", err)
	}
	defer file.Close()

	log.Println("Uploading file to the server...")
	err = c.Stor(req.Source.Filename, file)
	if err != nil {
		return fmt.Errorf("error uploading remote file: %s", err)
	}
	file.Close()

	log.Println("Reading local file...")
	filedata, err := ioutil.ReadFile(basePath + "/" + req.Parameters.Path)
	if err != nil {
		return fmt.Errorf("error reading local file: %s", err)
	}

	log.Println("Computing file hash...")
	hasher := sha256.New()
	_, err = hasher.Write(filedata)
	if err != nil {
		return fmt.Errorf("error computing file hash: %s", err)
	}

	fmt.Println("{\"version\":{\"ref\":\"" + hex.EncodeToString(hasher.Sum(nil)) + "\"},\"metadata\":[]}")
	return nil
}
