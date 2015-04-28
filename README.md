# fester

A command-line tool to generate a JSON manifest that summarizes information about a list of
docker images.

# Usage

    fester --images <images-file> --registry <registry> --tag <tag> --output <output-file>

--images gives the path to a file containing image names, one per line.

--registry gives the name of the registry to pull the images from.

--tag gives the tag to pull for the images.

--output gives the path to the file that the JSON should be written out to.

Here's a more concrete example:

    fester --images images.txt --registry discoenv --tag dev --output manifest.json
