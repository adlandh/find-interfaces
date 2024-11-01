package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var interfaceRegex = regexp.MustCompile(`type\s+(\w+)\s+interface\s*\{`)

func findInterfaces(folder string) ([]string, error) {
	var interfaces []string

	err := filepath.WalkDir(folder, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && path != folder {
			return filepath.SkipDir
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			src := string(data)
			matches := interfaceRegex.FindAllStringSubmatch(src, -1)
			for _, match := range matches {
				interfaces = append(interfaces, match[1])
			}
		}
		return nil
	})
	return interfaces, err
}

func main() {
	var flagPath string

	flag.StringVar(&flagPath, "path", ".", "path to search")
	flag.Parse()

	interfaces, err := findInterfaces(flagPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(strings.Join(interfaces, " "))
}
