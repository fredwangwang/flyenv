package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/concourse/concourse/fly/rc"
	"github.com/jessevdk/go-flags"
)

type Command struct {
	Target  rc.TargetName `short:"t" long:"target" description:"Concourse target name"`
	Version func()        `short:"v" long:"version" description:"Print the version of Fly and exit"`
}

var command Command
var fly string
var versionFlag bool

func main() {
	fly = flyCliName()
	folder := filepath.Join(userHomeDir(), ".flyenv")
	bundledFolder := filepath.Join(folder, "bundled")
	flyPath := filepath.Join(bundledFolder, fly)

	createFolder(bundledFolder)
	if !flyInstalled(bundledFolder) {
		getFlyCliFromGithub(bundledFolder, getLatestCliVersion())
	}

	parser := flags.NewParser(&command, flags.IgnoreUnknown)
	_, err := parser.Parse()
	if err != nil {
		log.Fatalf("error parsing args: %s", err)
	}

	if command.Target != "" {
		target, version, ok, err := getTargetApiVersion(command.Target)
		if err != nil {
			log.Fatalf("error fetching target: %s", err)
		}

		if ok {
			versionFolder := filepath.Join(folder, version)
			createFolder(versionFolder)
			getFlyCliFromTargetIfNotInstalled(target.URL(), versionFolder)
			flyPath = filepath.Join(versionFolder, fly)
		}
	}

	var stdoutBuffer strings.Builder
	cmd := &exec.Cmd{
		Path:   flyPath,
		Args:   os.Args,
		Env:    os.Environ(),
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if versionFlag {
		cmd.Stdout = &stdoutBuffer
	}

	if err = cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		log.Fatal(err)
	}

	if versionFlag {
		fmt.Printf("%s-flyenv", strings.TrimRight(stdoutBuffer.String(), "\n\r"))
	}
}

func getTargetApiVersion(name rc.TargetName) (rc.Target, string, bool, error) {
	target, err := rc.LoadTarget(name, false)
	if err != nil {
		if _, ok := err.(rc.UnknownTargetError); ok {
			return target, "", false, nil
		}
		return target, "", false, err
	}

	target, err = rc.LoadUnauthenticatedTarget(
		name,
		"",
		target.TLSConfig().InsecureSkipVerify,
		target.CACert(),
		false,
	)
	if err != nil {
		return target, "", false, err
	}

	version, err := target.Version()
	return target, version, true, err
}

func flyInstalled(folder string) bool {
	return fileExists(filepath.Join(folder, fly))
}

func getFlyCliFromTargetIfNotInstalled(target, targetFolder string) {
	if !flyInstalled(targetFolder) {
		getFlyCliFromTarget(target, targetFolder)
	}
}

func flyCliName() string {
	if runtime.GOOS == "windows" {
		return "fly.exe"
	}
	return "fly"
}

func getFlyCliFromTarget(target string, targetFolder string) {
	downloadUrl := fmt.Sprintf("%s/api/v1/cli?arch=amd64&platform=", target)
	switch runtime.GOOS {
	case "windows":
		fallthrough
	case "darwin":
		fallthrough
	case "linux":
		downloadUrl = downloadUrl + runtime.GOOS
	default:
		log.Fatal(runtime.GOOS, " is not supported")
	}

	_, skipSSL := os.LookupEnv("FLYENV_SKIP_SSL")
	if skipSSL {
		log.Println("FLYENV_SKIP_SSL variable set: skippling SSL validation")
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	log.Println("download cli from:", downloadUrl)
	resp, err := http.Get(downloadUrl)
	if err != nil {
		log.Fatal("error downloading fly cli", err)
	}
	if resp.StatusCode != 200 {
		log.Fatal("get ", resp.StatusCode, " when downloading fly cli")
	}
	defer resp.Body.Close()

	outputPath := filepath.Join(targetFolder, fly)
	f, err := os.Create(outputPath)
	if err != nil {
		log.Fatal("error creating fly cli file", err)
	}
	io.Copy(f, resp.Body)
	f.Close()
	os.Chmod(outputPath, 0777)
}

func getFlyCliFromGithub(targetFolder string, version string) {
	downloadUrl := "https://github.com/concourse/concourse/releases/download/v%s/fly-%s-%s"
	fileType := "tgz"

	if runtime.GOOS == "linux" {
		downloadUrl = fmt.Sprintf(downloadUrl, version, version, "linux-amd64.tgz")
	} else if runtime.GOOS == "darwin" {
		downloadUrl = fmt.Sprintf(downloadUrl, version, version, "darwin-amd64.tgz")
	} else if runtime.GOOS == "windows" {
		downloadUrl = fmt.Sprintf(downloadUrl, version, version, "windows-amd64.zip")
		fileType = "zip"
	} else {
		log.Fatal(runtime.GOOS, " is not supported")
	}

	log.Println("download cli from:", downloadUrl)
	resp, err := http.Get(downloadUrl)
	if err != nil {
		log.Fatal("error downloading fly cli", err)
	}
	if resp.StatusCode != 200 {
		log.Fatal("get ", resp.StatusCode, " when downloading fly cli")
	}
	defer resp.Body.Close()

	if fileType == "tgz" {
		untgz(resp.Body, fly, targetFolder)
	} else {
		unzip(resp.Body, fly, targetFolder)
	}
}

func getLatestCliVersion() string {
	resp, err := http.Get("https://api.github.com/repos/concourse/concourse/releases/latest")
	if err != nil {
		log.Fatal("error fetching concourse github api", err)
	}
	defer resp.Body.Close()

	payload := make(map[string]interface{})
	json.NewDecoder(resp.Body).Decode(&payload)
	return strings.TrimLeft(payload["tag_name"].(string), "v")
}

func createFolder(folder string) {
	err := os.MkdirAll(folder, 0755)
	if err != nil {
		log.Fatal("failed to setup folder", err)
	}
}

func userHomeDir() string {
	home := os.Getenv("HOME")
	if home != "" {
		return home
	}

	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
		if home != "" {
			return home
		}

		home = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home != "" {
			return home
		}
	}

	panic("could not detect home directory for .flyrc")
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func untgz(srcFile io.Reader, name, folder string) {
	gzf, err := gzip.NewReader(srcFile)
	if err != nil {
		log.Fatal("error precessing file", err)
	}

	tarReader := tar.NewReader(gzf)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		hname := header.Name

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			if hname == name {
				outputPath := filepath.Join(folder, hname)
				f, err := os.Create(outputPath)
				if err != nil {
					log.Fatal("error creating fly cli file", err)
				}
				io.Copy(f, tarReader)
				f.Close()
				os.Chmod(outputPath, 0777)
			}
		default:
			fmt.Printf("%s : %c %s %s\n",
				"Yikes! Unable to figure out type",
				header.Typeflag,
				"in file",
				hname,
			)
		}
	}
}

func unzip(srcFile io.Reader, name, folder string) {
	srcContent, err := ioutil.ReadAll(srcFile)
	if err != nil {
		log.Fatalf("error reading the content: %s", err)
	}

	r, err := zip.NewReader(bytes.NewReader(srcContent), int64(len(srcContent)))
	if err != nil {
		log.Fatalf("error reading the content: %s", err)
	}

	for _, f := range r.File {
		if f.Name != name {
			log.Println("ignoring", f.Name)
			continue
		}

		fpath := filepath.Join(folder, name)

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			log.Fatal("error creating fly cli file", err)
		}

		rc, err := f.Open()
		if err != nil {
			log.Fatal("error reading fly cli content", err)
		}

		io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()
		os.Chmod(fpath, 0777)

	}
}

func init() {
	command.Version = func() { versionFlag = true }
}
