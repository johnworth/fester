package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

var (
	reg   = flag.String("registry", "", "The registry to pull from")
	imgs  = flag.String("images", "", "Path to a new-line delimited list of image names")
	tag   = flag.String("tag", "", "The tag to pull")
	outf  = flag.String("output", "", "The file to write the JSON to.")
	files = flag.String("files", "", "A list of files that need to be included in the manifest.")
)

func init() {
	flag.Parse()
}

// VersionInfo encapsulates version info extracted from a Docker image.
type VersionInfo struct {
	AppVersion string `json:"app_version"`
	GitRef     string `json:"git_ref"`
	BuiltBy    string `json:"built_by"`
	ImageID    string `json:"image_id"`
}

// NewVersionInfo creates a new VersionInfo instance from info parsed out of a
// []string.
func NewVersionInfo(parsefrom []string, imageID string) *VersionInfo {
	var appver, gitref, builtby string
	for _, p := range parsefrom {
		if strings.HasPrefix(p, "App-Version: ") {
			appver = strings.TrimSpace(strings.TrimLeft(p, "App-Version: "))
		}
		if strings.HasPrefix(p, "Git-Ref: ") {
			gitref = strings.TrimSpace(strings.TrimLeft(p, "Git-Ref: "))
		}
		if strings.HasPrefix(p, "Built-By: ") {
			builtby = strings.TrimSpace(strings.TrimLeft(p, "Built-By: "))
		}
	}
	v := &VersionInfo{
		AppVersion: appver,
		GitRef:     gitref,
		BuiltBy:    builtby,
		ImageID:    imageID,
	}
	return v
}

// ImageSpecifier encapsulates a Docker image string.
type ImageSpecifier struct {
	Registry string
	Image    string
	Tag      string
}

// New returns a pointer to a new ImageSpecifier.
func New(reg, img, tag string) *ImageSpecifier {
	i := &ImageSpecifier{
		Registry: reg,
		Image:    img,
		Tag:      tag,
	}
	return i
}

func (i *ImageSpecifier) String() string {
	return fmt.Sprintf("%s/%s:%s", i.Registry, i.Image, i.Tag)
}

// Pull pulls a Docker image.
func (i *ImageSpecifier) Pull() error {
	cmd := exec.Command(
		"docker",
		"pull",
		i.String(),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ReadLines parses a []byte into a []string based on newlines.
func ReadLines(content []byte) []string {
	var lines []string
	reader := bytes.NewReader(content)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

// ImageID returns the image identifier for the docker image.
func (i *ImageSpecifier) ImageID() (string, error) {
	var outbuf bytes.Buffer
	outwriter := bufio.NewWriter(&outbuf)
	cmd := exec.Command(
		"docker",
		"inspect",
		"--format",
		"{{.Id}}",
		i.String(),
	)
	cmd.Stdout = outwriter
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	err = outwriter.Flush()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(outbuf.String()), nil
}

// Version returns the output of calling --version on the image.
func (i *ImageSpecifier) Version() (*VersionInfo, error) {
	var outbuf bytes.Buffer
	outwriter := bufio.NewWriter(&outbuf)
	cmd := exec.Command(
		"docker",
		"run",
		"--rm",
		i.String(),
		"--version",
	)
	cmd.Stdout = outwriter
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	err = outwriter.Flush()
	if err != nil {
		return nil, err
	}
	imageID, err := i.ImageID()
	if err != nil {
		return nil, err
	}
	return NewVersionInfo(ReadLines(outbuf.Bytes()), imageID), nil
}

// ReadImages reads in file and returns a []string of image names.
func ReadImages(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	filebytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return ReadLines(filebytes), nil
}

// ReadFilesList reads in the list of files that need to be included in the drop
// and returns the map.
func ReadFilesList(filename string) (map[string]string, error) {
	filebytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	retval := make(map[string]string)
	err = json.Unmarshal(filebytes, &retval)
	if err != nil {
		return nil, err
	}
	return retval, nil
}

// OutputMap contains the info that is written out to a file.
type OutputMap struct {
	DropFiles    map[string]string         `json:"drop_files"`
	DockerImages map[string][]*VersionInfo `json:"docker_images"`
}

func main() {
	if *imgs == "" {
		log.Fatal("--images must be set")
	}
	if *reg == "" {
		log.Fatal("--registry must be set")
	}
	if *tag == "" {
		log.Fatalf("--tag must be set")
	}
	if *outf == "" {
		log.Fatalf("--output must be set")
	}
	images, err := ReadImages(*imgs)
	if err != nil {
		log.Fatalf("Error reading images: %s", err)
	}
	dropFiles, err := ReadFilesList(*files)
	if err != nil {
		log.Fatalf("Error reading files list: %s", err)
	}
	imageVersions := make(map[string][]*VersionInfo)
	for _, image := range images {
		fmt.Println(image)
		spec := New(*reg, image, *tag)
		err = spec.Pull()
		if err != nil {
			log.Fatalf("Error pulling image: %s", err)
		}
		v, err := spec.Version()
		if err != nil {
			log.Fatalf("Error getting version: %s", err)
		}
		imageVersions[spec.String()] = append(imageVersions[spec.String()], v)
	}
	output := &OutputMap{
		DropFiles:    dropFiles,
		DockerImages: imageVersions,
	}
	imgJSON, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatalf("Error marshalling JSON: %s", err)
	}
	err = ioutil.WriteFile(*outf, imgJSON, 0644)
	if err != nil {
		log.Fatalf("Error writing JSON file: %s", err)
	}
}
