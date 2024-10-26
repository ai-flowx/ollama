# ollama-export



## Introduction

*ollama-export* is module export of [ai-shflow](https://github.com/ai-shflow) written in Go.



## Prerequisites

- Go >= 1.23.0



## Build

```bash
make build
```



## Usage

```
Usage:
  ollama-export <name> [flags]

Examples:

  ollama-export llama3:latest
  ollama-export llama3:latest -o /path/to/file


Flags:
  -h, --help            help for ollama-export
  -o, --output string   write to file
```



## License

Project License can be found [here](LICENSE).



## Reference

- [ollama-export.go](https://gist.github.com/JerrettDavis/7bc86098e705e3a7b4efcd60a2b413d7)
