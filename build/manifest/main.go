package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
)

const PluginIdGoFileTemplate = `package main

const PluginId = "%s"
`

const PluginIdJsFileTemplate = `export default '%s';
`

func main() {
	if len(os.Args) <= 1 {
		panic("no cmd specified")
	}

	manifest, err := findManifest()
	if err != nil {
		panic("failed to find manifest: " + err.Error())
	}

	cmd := os.Args[1]
	switch cmd {
	case "plugin_id":
		dumpPluginId(manifest)

	case "has_server":
		if manifest.HasServer() {
			fmt.Printf("true")
		}

	case "has_webapp":
		if manifest.HasWebapp() {
			fmt.Printf("true")
		}

	case "apply":
		if err := applyManifest(manifest); err != nil {
			panic("failed to apply manifest: " + err.Error())
		}

	default:
		panic("unrecognized command: " + cmd)
	}
}

func findManifest() (*model.Manifest, error) {
	_, manifestFilePath, err := model.FindManifest(".")
	if err != nil {
		return nil, errors.Wrap(err, "failed to find manifest in current working directory")
	}
	manifestFile, err := os.Open(manifestFilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open %s", manifestFilePath)
	}
	defer manifestFile.Close()

	// Re-decode the manifest, disallowing unknown fields. When we write the manifest back out,
	// we don't want to accidentally clobber anything we won't preserve.
	var manifest model.Manifest
	decoder := json.NewDecoder(manifestFile)
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&manifest); err != nil {
		return nil, errors.Wrap(err, "failed to parse manifest")
	}

	return &manifest, nil
}

// dumpPluginId writes the plugin id from the given manifest to standard out
func dumpPluginId(manifest *model.Manifest) {
	fmt.Printf("%s", manifest.Id)
}

// applyManifest propagates the plugin_id into the server and webapp folders, as necessary
func applyManifest(manifest *model.Manifest) error {
	if manifest.HasServer() {
		if err := ioutil.WriteFile(
			"server/plugin_id.go",
			[]byte(fmt.Sprintf(PluginIdGoFileTemplate, manifest.Id)),
			0644,
		); err != nil {
			return errors.Wrap(err, "failed to write server/plugin_id.go")
		}
	}

	if manifest.HasWebapp() {
		if err := ioutil.WriteFile(
			"webapp/src/plugin_id.js",
			[]byte(fmt.Sprintf(PluginIdJsFileTemplate, manifest.Id)),
			0644,
		); err != nil {
			return errors.Wrap(err, "failed to open webapp/src/plugin_id.js")
		}
	}

	return nil
}
