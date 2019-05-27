package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

var matcher *regexp.Regexp

func init() {
	viper.SetConfigName("tmpl_config")

	viper.AddConfigPath("/etc/tmpl")
	viper.AddConfigPath("$HOME/.tmpl")
	viper.AddConfigPath("config")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("_", "."))

	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(err)
		}
	}
}

func getFileMatches(dirMatcher, fileMatcher string) ([]string, error) {
	matches, err := filepath.Glob(path.Join(dirMatcher, "**", fileMatcher))
	if err != nil {
		return nil, err
	}

	addMatches, err := filepath.Glob(path.Join(dirMatcher, fileMatcher))
	if err != nil {
		return nil, err
	}

	return append(matches, addMatches...), nil
}

func updateFile(filePath string) (err error) {
	fmt.Println(filePath)

	stats, err := os.Stat(filePath)
	if err != nil {
		return
	}

	rawData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return
	}
	fileContent := string(rawData)

	uniqueMatches := map[string]string{}
	matches := matcher.FindAllStringSubmatch(fileContent, -1)
	for _, match := range matches {
		uniqueMatches[match[0]] = strings.Replace(match[1], "_", ".", -1)
	}

	for replace, replaceVar := range uniqueMatches {
		newValue := viper.GetString(replaceVar)
		if newValue == "" && os.Getenv(replaceVar) != "" {
		    newValue = os.Getenv(replaceVar)
		}
		if newValue == "" {
			fmt.Printf("[WARN] Variable %s has not been set!\n", strings.Replace(replaceVar, ".", "_", -1))
		}
		fileContent = strings.Replace(fileContent, replace, newValue, -1)
	}

	dirname, filename := path.Split(filePath)
	filename = strings.Replace(filename, ".tmpl", "", 1)
	return ioutil.WriteFile(path.Join(dirname, filename), []byte(fileContent), stats.Mode())
}

func main() {
	var err error

	// TODO allow config overrides for these
	dirMatcher := "."
	fileMatcher := ".tmpl"
	matcher, err = regexp.Compile(`\$\{?([a-zA-Z_.]+)\}?`)
	if err != nil {
		panic(err)
	}
	// TODO allow config overrides for these

	matches, err := getFileMatches(dirMatcher, "*"+fileMatcher+"*")
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(len(matches))

	for _, filePath := range matches {
		go func(filePath string) {
			if err := updateFile(filePath); err != nil {
				fmt.Printf("Failed to process %s: %s\n", filePath, err)
			}

			wg.Done()
		}(filePath)
	}

	wg.Wait()
}
