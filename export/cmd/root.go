package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	minArgs = 1
)

var (
	modelName   string
	modelOutput string
)

var rootCmd = &cobra.Command{
	Use: "ollama-export <name>",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < minArgs {
			return errors.New("invalid argument\n")
		}
		if len(args) == minArgs && args[0] == "help" {
			return errors.New("invalid argument\n")
		}
		modelName = args[0]
		return nil
	},
	Example: "\n" +
		"  ollama-export llama3.2\n" +
		"  ollama-export llama3.2 -o /path/to/llama3.2\n",
	Run: func(cmd *cobra.Command, args []string) {
		if err := execute(); err != nil {
			os.Exit(1)
		}
	},
}

type Layer struct {
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
}

type Manifest struct {
	Layers []Layer `json:"layers"`
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// nolint: gochecknoinits
func init() {
	rootCmd.Root().CompletionOptions.DisableDefaultCmd = true

	rootCmd.Flags().StringVarP(&modelOutput, "output", "o", "", "write to files")
}

func execute() error {
	usr, err := user.Current()
	if err != nil {
		return errors.Wrap(err, "failed to get current user")
	}

	ollamaHome := filepath.Join(usr.HomeDir, ".ollama")
	blobsFileBasePath := filepath.Join(ollamaHome, "models", "blobs")
	manifestsFileBasePath := filepath.Join(ollamaHome, "models", "manifests")

	nameArgs := strings.Split(strings.ReplaceAll(modelName, ":", "/"), "/")

	var manifestsRegistryName, manifestsLibraryName string
	var manifestsModelName, manifestsParamsName string

	switch len(nameArgs) {
	case 4:
		manifestsRegistryName = nameArgs[0]
		manifestsLibraryName = nameArgs[1]
		manifestsModelName = nameArgs[2]
		manifestsParamsName = nameArgs[3]
	case 3:
		manifestsLibraryName = nameArgs[0]
		manifestsModelName = nameArgs[1]
		manifestsParamsName = nameArgs[2]
	case 2:
		manifestsModelName = nameArgs[0]
		manifestsParamsName = nameArgs[1]
	case 1:
		manifestsModelName = nameArgs[0]
	}

	if manifestsRegistryName == "" {
		manifestsRegistryName = "registry.ollama.ai"
	}
	if manifestsLibraryName == "" {
		manifestsLibraryName = "library"
	}
	if manifestsModelName == "" {
		manifestsModelName = "vicuna"
	}
	if manifestsParamsName == "" {
		manifestsParamsName = "latest"
	}

	modelFullName := manifestsModelName + ":" + manifestsParamsName
	fmt.Printf("Exporting model \"%s\" to \"%s\"...\n\n", modelFullName, targetPath)

	manifestsFilePath := filepath.Join(manifestsFileBasePath, manifestsRegistryName, manifestsLibraryName, manifestsModelName, manifestsParamsName)
	if _, err := os.Stat(manifestsFilePath); os.IsNotExist(err) {
		return errors.Wrap(err, "manifest not found")
	}

	if _, err := os.Stat(targetPath); err == nil {
		return errors.Wrap(err, "target already exists")
	}

	if err := os.MkdirAll(targetPath, os.ModePerm); err != nil {
		return errors.Wrap(err, "failed to make directory")
	}

	sourceFilePath := filepath.Join(targetPath, "source.txt")
	if err := os.WriteFile(sourceFilePath, []byte(fmt.Sprintf("%s/%s/%s:%s", manifestsRegistryName, manifestsLibraryName, manifestsModelName, manifestsParamsName)), os.ModePerm); err != nil {
		return errors.Wrap(err, "failed to write file")
	}

	manifestData, err := os.ReadFile(manifestsFilePath)
	if err != nil {
		return errors.Wrap(err, "failed to read file")
	}

	var manifest Manifest

	if err = json.Unmarshal(manifestData, &manifest); err != nil {
		return errors.Wrap(err, "failed to unmarshal data")
	}

	exportModelFilePath := filepath.Join(targetPath, "Modelfile")
	exportModelBinPath := filepath.Join(targetPath, "model.bin")

	for _, layer := range manifest.Layers {
		blobFileName := strings.ReplaceAll(layer.Digest, ":", "-")
		blobFilePath := filepath.Join(blobsFileBasePath, blobFileName)
		blobData, err := os.ReadFile(blobFilePath)
		if err != nil {
			return errors.Wrap(err, "failed to read file")
		}
		blobTypeName := strings.Split(layer.MediaType, ".")[len(strings.Split(layer.MediaType, "."))-1]
		switch blobTypeName {
		case "model":
			if err := os.WriteFile(exportModelBinPath, blobData, os.ModePerm); err != nil {
				return errors.Wrap(err, "failed to write file")
			}
			appendToFile(exportModelFilePath, "FROM ./model.bin\n")
		case "params":
			paramsJson := string(blobData)
			paramsMap := make(map[string]interface{})
			if err := json.Unmarshal([]byte(paramsJson), &paramsMap); err != nil {
				return errors.Wrap(err, "failed to unmarshal data")
			}
			for key, value := range paramsMap {
				switch v := value.(type) {
				case []interface{}:
					for _, val := range v {
						appendToFile(exportModelFilePath, fmt.Sprintf("PARAMETER %s \"%v\"\n", key, val))
					}
				}
			}
		default:
			typeName := strings.ToUpper(blobTypeName)
			appendToFile(exportModelFilePath, fmt.Sprintf("%s \"\"\"%s\"\"\"\n", typeName, string(blobData)))
		}
	}

	fmt.Printf("%s Model \"%s\" has been exported to \"%s\"!\n", successPrefix, modelFullName, targetPath)

	return nil
}

func appendToFile(filePath, text string) error {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}

	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	if _, err = f.WriteString(text); err != nil {
		return errors.Wrap(err, "failed to write file")
	}

	return nil
}
