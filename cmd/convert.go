package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func handleFile(file string) (err error) {
	fm, rowsFmComment, rows, err := parseFile(file)
	if err != nil {
		return errors.Wrapf(err, "Failed to parseFile %s", file)
	}

	log.Debug(fm)
	log.Debug(rows)

	outputsFm, err := convertFrontMatter(fm)
	if err != nil {
		return errors.Wrap(err, "Failed to convertFrontMatter")
	}

	outputs := []string{}
	outputs = append(outputs, "---")
	outputs = append(outputs, outputsFm...)
	outputs = append(outputs, rowsFmComment...)
	outputs = append(outputs, "---")
	outputs = append(outputs, rows...)
	dst := fmt.Sprintf("%s%s", file, ".dst.md")
	return WriteFile(dst, strings.Join(outputs, "\n"))
}

func parseFile(file string) (fm map[string]any, rowsFmComment []string, rows []string, err error) {
	rowsFmComment = []string{}
	rows = []string{}
	fm = make(map[string]any)
	fp, err := os.Open(filepath.Clean(file))
	if err != nil {
		return fm, rowsFmComment, rows, err
	}
	defer func() {
		err = fp.Close()
	}()

	scanner := bufio.NewScanner(fp)
	start := false
	frontMatter := false
	rowsFm := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		r := regexp.MustCompile(`^---`)
		if r.MatchString(line) {
			if start {
				frontMatter = false
				// break
				continue
			}
			start = true
			frontMatter = true
			continue
		}
		if frontMatter {
			regComment := regexp.MustCompile(`^#.*`)
			if regComment.MatchString(line) {
				rowsFmComment = append(rowsFmComment, line)
				continue
			}
			rowsFm = append(rowsFm, line)
			continue
		}
		rows = append(rows, line)
		// fmt.Println(line)
	}

	err = yaml.Unmarshal([]byte(strings.Join(rowsFm, "\n")), &fm)
	return fm, rowsFmComment, rows, err
}

// draft: false
// date: 2023-08-16T15:36:12+09:00
// lastmod: 2023-09-07T08:51:04+09:00
// cover:
//     image: img.png
// title: "SQLAlchemy で Flake8 error: E712 comparison to True should be 'if cond is True:' or 'if cond:' の lint エラーに対応する"
// categories:
//   - tech
// tags:
//   - python
//   - SQLAlchemy
// showToc: false
func convertFrontMatter(fm map[string]any) ([]string, error) {
	order := []string{"draft", "date", "lastmod", "cover", "title", "categories", "tags"}
	outputs := []string{}
	for _, key := range order {
		val, ok := fm[key]
		if !ok {
			switch key {
			case "draft":
				val = false
			// case "date":
			// 	val = ""
			case "lastmod":
				val = time.Now().Format("2006-01-02T15:04:05+09:00")
			case "cover":
				// TODO image move to directory?
				valImage, ok := fm["image"]
				if !ok || valImage == nil {
					continue
				}
				val = map[string]string{"image": valImage.(string)}
			// case "title":
			// 	val = ""
			// case "categories":
			// 	val = []string{}
			// case "tags":
			// 	val = []string{}
			default:
				// val = ""
				continue
			}
		} else {
			if key == "cover" {
				if reflect.TypeOf(val).Kind() == reflect.String {
					val = map[string]string{"image": val.(string)}
				}
			}
		}
		outputs = append(outputs, getAppendOutputs(key, val, false)...)
	}

	outputs = append(outputs, "####################")

	for key, val := range fm {
		if slices.Contains(order, key) {
			continue
		}
		outputs = append(outputs, getAppendOutputs(key, val, true)...)
	}
	return outputs, nil
}

func getAppendOutputs(key string, val any, asComment bool) []string {
	outputs := []string{}
	comment := ""
	if asComment {
		comment = "# "
	}
	formatVal := "%v"
	if val == nil {
		val = ""
	}
	// log.Infof("key: %s, val: %v", key, val)
	if key == "title" {
		formatVal = "%q"
	} else if reflect.TypeOf(val).Kind() == reflect.Bool {
		formatVal = "%t"
	} else if reflect.TypeOf(val).Kind() == reflect.Map {
		// TODO map for cover. cover must be img.png?
		outputs = append(outputs, fmt.Sprintf(comment+"%s:", key))
		for k, v := range val.(map[string]any) {
			outputs = append(outputs, fmt.Sprintf(comment+"  %s: %s", k, v))
		}
		return outputs
	}
	outputs = append(outputs, fmt.Sprintf(comment+"%s: "+formatVal, key, val))
	return outputs
}

// func appendOutputs(outputs []string, key string, def any, fm map[any]any) ([]string, error) {
// 	var output any
// 	if _, ok := fm[key]; ok {
// 		output = fm[key]
// 	} else {
// 		output = def
// 	}
// 	t := reflect.TypeOf(output)
// 	switch t.Kind() {
// 	case reflect.Bool:
// 		outputs = append(outputs, fmt.Sprintf("%s: %t", key, output))
// 	case reflect.Map:
// 		outputs = append(outputs, fmt.Sprintf("%s:", key))
// 		for k, v := range output.(map[any]any) {
// 			outputs = append(outputs, fmt.Sprintf("  %s: %v", k, v))
// 		}
// 	case reflect.Struct:
// 		outputs = append(outputs, fmt.Sprintf("%s: %v", key, output))
// 	// case reflect.String:
// 	// 	outputs = append(outputs, fmt.Sprintf("%s: %s", key, output))
// 	// case reflect.Slice:
// 	// 	outputs = append(outputs, fmt.Sprintf("%s: %s", key, output))
// 	default:
// 		outputs = append(outputs, fmt.Sprintf("%s: %v", key, output))
// 	}
// 	// switch output.(type) {
// 	// case string:
// 	// 	outputs = append(outputs, fmt.Sprintf("%s: %s", key, output))
// 	// 	return outputs, nil
// 	//        case
// 	// }
//
// 	return outputs, nil
// }

func WriteFile(filePath, data string) error {
	return os.WriteFile(filePath, []byte(data), 0664) //#nosec G306
}

func handleInner() error {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return err
	}
	if (fi.Mode() & os.ModeCharDevice) != 0 {
		// not pipe
		return errors.New("No stdin")
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		file := scanner.Text()
		log.Infof("==> File:%s", file)
		err = handleFile(file)
		if err != nil {
			return err
		}
	}
	return nil
}

func handle() {
	// initConfigInner
	level, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err == nil {
		log.SetLevel(level)
	}
	err = handleInner()
	if err != nil {
		log.Error("Failed to handle. Error: ", err)
	}
}
