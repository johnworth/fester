package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

var (
	dockerURI = flag.String("docker-uri", "unix:///var/run/docker.sock", "The docker URI.")
	outf      = flag.String("output", "", "The file to write the JSON to.")
	files     = flag.String("files", "", "A list of files that need to be included in the manifest.")
)

// OutputMap contains the info that is written out to a file.
type OutputMap struct {
	Hostname   string            `json:"hostname"`
	Date       string            `json:"date"`
	Images     []types.Image     `json:"images"`
	Containers []types.Container `json:"containers"`
}

func main() {
	flag.Parse()

	//Create the docker client. Config will be blank because this apps doesn't
	//use it and isn't creating any porklock containers.
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	d, err := client.NewClient(*dockerURI, "v1.22", nil, defaultHeaders)
	if err != nil {
		log.Fatalf("Error creating docker client: %s", err)
	}

	ctx := context.Background()

	images, err := d.ImageList(ctx, types.ImageListOptions{All: true})
	if err != nil {
		log.Fatal(err)
	}

	containers, err := d.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		log.Fatal(err)
	}

	var hostname string
	if hostname, err = os.Hostname(); err != nil {
		hostname = ""
	}

	date := time.Now().Format("2006-01-02T15:04:05-07:00")

	output := &OutputMap{
		Hostname:   hostname,
		Date:       date,
		Images:     images,
		Containers: containers,
	}

	imgJSON, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatalf("Error marshalling JSON: %s", err)
	}
	if _, err = os.Stdout.Write(imgJSON); err != nil {
		log.Fatal(err)
	}
}
