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

	defaultRegistry = "registry.ollama.ai"
	defaultLibrary  = "library"
	defaultTag      = "latest"

	dirPerm  = 0755
	filePerm = 0644
)

var (
	modelName string
	modelPath string
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
		"  ollama-export llama3:latest\n" +
		"  ollama-export llama3:latest -o /path/to/file\n",
	Run: func(cmd *cobra.Command, args []string) {
		if err := execute(); err != nil {
			fmt.Println(err.Error())
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

	rootCmd.Flags().StringVarP(&modelPath, "output", "o", "", "write to file")
}

// nolint:funlen,gocritic,gocyclo,mnd
func execute() error {
	usr, err := user.Current()
	if err != nil {
		return errors.Wrap(err, "failed to get current user")
	}

	ollamaHome := filepath.Join(usr.HomeDir, ".ollama")
	blobsPath := filepath.Join(ollamaHome, "models", "blobs")
	manifestsPath := filepath.Join(ollamaHome, "models", "manifests")

	nameArgs := strings.Split(strings.ReplaceAll(modelName, ":", "/"), "/")

	var registryName, libraryName, tagName string

	switch len(nameArgs) {
	case 4:
		registryName = nameArgs[0]
		libraryName = nameArgs[1]
		modelName = nameArgs[2]
		tagName = nameArgs[3]
	case 3:
		libraryName = nameArgs[0]
		modelName = nameArgs[1]
		tagName = nameArgs[2]
	case 2:
		modelName = nameArgs[0]
		tagName = nameArgs[1]
	case 1:
		modelName = nameArgs[0]
	}

	if registryName == "" {
		registryName = defaultRegistry
	}

	if libraryName == "" {
		libraryName = defaultLibrary
	}

	if modelName == "" {
		return errors.New("invalid model name")
	}

	if tagName == "" {
		tagName = defaultTag
	}

	fullName := modelName + ":" + tagName

	if modelPath == "" {
		root, _ := os.Getwd()
		modelPath = filepath.Join(root, modelName+"-"+tagName)
	}

	fmt.Printf("Export model %s to %s\n", fullName, modelPath)

	manifestsFilePath := filepath.Join(manifestsPath, registryName, libraryName, modelName, tagName)
	if _, err := os.Stat(manifestsFilePath); os.IsNotExist(err) {
		return errors.Wrap(err, "manifest not found")
	}

	if _, err := os.Stat(modelPath); err == nil {
		return errors.Wrap(err, "path already exists")
	}

	if err := os.MkdirAll(modelPath, os.ModePerm); err != nil {
		return errors.Wrap(err, "failed to make directory")
	}

	sourceFilePath := filepath.Join(modelPath, "source.txt")
	if err := os.WriteFile(sourceFilePath, []byte(fmt.Sprintf("%s/%s/%s:%s", registryName, libraryName, modelName, tagName)),
		filePerm); err != nil {
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

	exportModelFilePath := filepath.Join(modelPath, "Modelfile")
	exportModelBinPath := filepath.Join(modelPath, "model.bin")

	for _, layer := range manifest.Layers {
		blobFileName := strings.ReplaceAll(layer.Digest, ":", "-")
		blobFilePath := filepath.Join(blobsPath, blobFileName)
		blobData, err := os.ReadFile(blobFilePath)
		if err != nil {
			return errors.Wrap(err, "failed to read file")
		}
		blobTypeName := strings.Split(layer.MediaType, ".")[len(strings.Split(layer.MediaType, "."))-1]
		switch blobTypeName {
		case "model":
			if err := os.WriteFile(exportModelBinPath, blobData, filePerm); err != nil {
				return errors.Wrap(err, "failed to write file")
			}
			if err := appendFile(exportModelFilePath, "FROM ./model.bin\n"); err != nil {
				return errors.Wrap(err, "failed to append file")
			}
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
						if err := appendFile(exportModelFilePath, fmt.Sprintf("PARAMETER %s \"%v\"\n", key, val)); err != nil {
							return errors.Wrap(err, "failed to append file")
						}
					}
				}
			}
		default:
			typeName := strings.ToUpper(blobTypeName)
			if err := appendFile(exportModelFilePath, fmt.Sprintf("%s \"\"\"%q\"\"\"\n", typeName, string(blobData))); err != nil {
				return errors.Wrap(err, "failed to append file")
			}
		}
	}

	fmt.Printf("Model %s has been exported to %s\n", fullName, modelPath)

	return nil
}

func appendFile(filePath, text string) error {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, filePerm)
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
