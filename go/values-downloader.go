package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"dario.cat/mergo"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// func getHelmCmd() (ProcessInfo, error) {
// 	switch runtime.GOOS {
// 	case "windows":
// 		return GetHelmCmdWindows()
// 	default:
// 		return GetHelmCmdUnix()
// 	}
// }

func getChart(helmCmd string) (chart, chartVersion, chartRepo string) {
	args := parseHelmCmdArgs(helmCmd)
	var sawChartVersion, sawChartRepo bool

	for i := 0; i < len(args); i++ {
		opt := args[i]
		switch {
		case opt == "--version":
			sawChartVersion = true
		case len(opt) > 9 && opt[:9] == "--version":
			// --version=XXX
			if eqIdx := indexOf(opt, '='); eqIdx != -1 {
				chartVersion = opt[eqIdx+1:]
			}
		case opt == "--repo":
			sawChartRepo = true
		case len(opt) > 6 && opt[:6] == "--repo=":
			// --repo=XXX
			if eqIdx := indexOf(opt, '='); eqIdx != -1 {
				chartRepo = opt[eqIdx+1:]
			}
		case len(opt) > 1 && opt[:2] == "--":
			// skip other flags
		default:
			if sawChartVersion {
				chartVersion = opt
				sawChartVersion = false
			} else if sawChartRepo {
				chartRepo = opt
				sawChartRepo = false
			} else {
				chart = opt
			}
		}
	}
	return
}

// Helper: splits a command line into args, handling quotes (simple version)
func parseHelmCmdArgs(cmd string) []string {
	var args []string
	var current string
	inQuote := false
	quoteChar := byte(0)
	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		switch c {
		case ' ', '\t':
			if inQuote {
				current += string(c)
			} else if len(current) > 0 {
				args = append(args, current)
				current = ""
			}
		case '\'', '"':
			if inQuote && c == quoteChar {
				inQuote = false
			} else if !inQuote {
				inQuote = true
				quoteChar = c
			} else {
				current += string(c)
			}
		default:
			current += string(c)
		}
	}
	if len(current) > 0 {
		args = append(args, current)
	}
	return args
}

// Helper: returns index of first '=' or -1
func indexOf(s string, ch byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ch {
			return i
		}
	}
	return -1
}

func getLocal(chart string, valueFile string, tmpDir string) {
	Fdebug("Called getLocal with chart=%s, valueFile=%s", chart, valueFile)
	PrintValues(chart, valueFile, chart, tmpDir)
}

func getRemote(chart string, valueFile string, chartVersion string, chartRepo string, tmpDir string) {
	Fdebug("Called getRemote with chart=%s, valueFile=%s, chartVersion=%s, chartRepo=%s", chart, valueFile, chartVersion, chartRepo)

	helm := os.Getenv("HELM_BIN")
	if helm == "" {
		helm = "helm"
	}

	var args []string
	args = append(args, "pull")
	args = append(args, chart)
	if chartRepo != "" {
		args = append(args, "--repo", chartRepo)
	}
	if chartVersion != "" {
		args = append(args, "--version", chartVersion)
	}

	id := uuid.New().String()

	args = append(args, "--debug", "--untar", "--destination", tmpDir, "--untardir", id)

	cmd := exec.Command(helm, args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		Ferror("Erreur: %v", err)
		return
	}

	// If chart is repo/chartname we need to extract the chart name
	if strings.Contains(chart, "/") {
		parts := strings.Split(chart, "/")
		if len(parts) > 0 {
			chart = parts[len(parts)-1]
		}
	}

	extractedFolder := tmpDir + string(os.PathSeparator) + id + string(os.PathSeparator) + chart
	PrintValues(extractedFolder, valueFile, extractedFolder, tmpDir)
}

// Helper function to merge two maps, with src overriding dest
func mergeMaps(dest, src map[string]interface{}) {
	mergo.Merge(&dest, src, mergo.WithOverride)
}

/**
 * Recursive function to search for values files in a chart directory.
 * It traverses the directory structure and merges global values from found files into globalMap,
 * and the other values in localMap
 */
func searchInChart(chartDir, prefix string, valueFile string,
	globalMap map[string]interface{},
	localMap map[string]interface{},
	tagMap map[string]interface{},
	tmpDir string) {

	Fdebug("Searching in chart directory: %s with prefix: %s", chartDir, prefix)

	chartsDir := chartDir + string(os.PathSeparator) + "charts"
	info, err := os.Stat(chartsDir)

	// No chart directory found, we are at the deepest level
	if err == nil && info.IsDir() {
		entries, err := os.ReadDir(chartsDir)
		if err != nil {
			LogError(fmt.Sprintf("Failed to read directory %s: %v", chartsDir, err))
		}
		for _, entry := range entries {
			if entry.IsDir() {
				// The entry is a directory, we assume it is a sub-chart
				// Recursively search in subdirectories
				_, exists := localMap[entry.Name()]
				if !exists {
					localMap[entry.Name()] = make(map[string]interface{})
				}
				searchInChart(chartsDir+string(os.PathSeparator)+entry.Name(), entry.Name(), valueFile,
					globalMap,
					localMap[entry.Name()].(map[string]interface{}),
					tagMap,
					tmpDir)
			} else {
				tgzPath := chartsDir + string(os.PathSeparator) + entry.Name()
				tgz, err := IsTgzFile(tgzPath)
				if err != nil {
					Fwarn("Error checking if %s is a tgz file: %v", entry.Name(), err)
					continue
				}
				if !tgz {
					Fdebug("Skipping non-tgz file: %s", entry.Name())
					continue
				}
				Fdebug("Found tgz file: %s", entry.Name())

				id := uuid.New().String()
				tmpDirTgz := tmpDir + string(os.PathSeparator) + id

				// We found a tgz file, we extract it to a temporary directory
				err = ExtractTgz(tgzPath, tmpDirTgz)
				if err != nil {
					Fwarn("Failed to extract tgz file %s: %v", tgzPath, err)
					continue
				}
				Fdebug("Extracted tgz file %s to %s", tgzPath, tmpDirTgz)

				// The extracted directory should containt 1 single directory with the subchart name
				subentries, err := os.ReadDir(tmpDirTgz)

				if err != nil {
					Ferror("Error reading extracted folder: %v", err)
					continue
				}

				// Vérifier qu'il y a exactement un élément
				if len(subentries) != 1 {
					Fwarn("Subchart %s does not contain 1 element. Incorrect helm structure for helm chart", entry.Name())
					continue
				}

				// Vérifier que c'est un répertoire
				if !subentries[0].IsDir() {
					Fwarn("Subchart %s does not contain 1 folder. Incorrect helm structure for helm chart", entry.Name())
					continue
				}

				// Récupérer le nom du répertoire
				dirName := subentries[0].Name()
				Fdebug("Folder found: %s\n", dirName)

				_, exists := localMap[dirName]
				if !exists {
					localMap[dirName] = make(map[string]interface{})
				}

				searchInChart(tmpDir+string(os.PathSeparator)+id+string(os.PathSeparator)+dirName,
					dirName,
					valueFile,
					globalMap,
					localMap[dirName].(map[string]interface{}),
					tagMap,
					tmpDir)
			}
		}
	}

	filePath := chartDir + string(os.PathSeparator) + valueFile

	_, err = os.Stat(filePath)
	if err == nil {

		Fdebug("File %s found in %s", valueFile, filePath)

		content, err := os.ReadFile(filePath)
		if err != nil {
			Fdebug("Failed to read file %s: %v", filePath, err)
		}

		var valuesMap map[string]interface{}
		if err := yaml.Unmarshal(content, &valuesMap); err != nil {
			Fdebug("Failed to unmarshal YAML: %v", err)
		}

		// Merge the global values into the globalMap
		globalValue, exists := valuesMap["global"]
		if exists {
			mergeMaps(globalMap, globalValue.(map[string]interface{}))
			delete(valuesMap, "global")
		}

		// Merge the tags into the tagMap
		tagValue, exists := valuesMap["tags"]
		if exists {
			mergeMaps(tagMap, tagValue.(map[string]interface{}))
			delete(valuesMap, "tags")
		}

		// Merge the valuesMap into the localMap
		mergeMaps(localMap, valuesMap)

	} else if os.IsNotExist(err) {
		Fdebug("File %s does not exist in %s, skipping", valueFile, filePath)
	} else {
		// some other error
		Fwarn("Error checking file %s: %v", filePath, err)
	}

}

// Helper function to remove empty submaps from a map
func removeEmptyMaps(m map[string]interface{}) {
	for key, value := range m {
		if subMap, ok := value.(map[string]interface{}); ok {
			// Nettoyer récursivement la sous-map
			removeEmptyMaps(subMap)

			// Si la sous-map est maintenant vide, la supprimer
			if len(subMap) == 0 {
				delete(m, key)
			}
		}
	}
}

// Core function to print values from a local chart
// It reads the values file and prints its content to stdout.
func PrintValues(chartPath string, valueFile string, chart string, tmpDir string) {

	localMap := make(map[string]interface{})
	globalMap := make(map[string]interface{})
	tagMap := make(map[string]interface{})

	searchInChart(chartPath, "", valueFile, globalMap, localMap, tagMap, tmpDir)

	localMap["global"] = globalMap
	localMap["tags"] = tagMap

	removeEmptyMaps(localMap)

	yamlBytes, err := yaml.Marshal(localMap)
	if err != nil {
		Fdebug("Failed to marshal YAML: %v", err)
	} else {
		fmt.Print(string(yamlBytes))
	}

}

func main() {
	Ftrace("Système d'exploitation: %s", runtime.GOOS)
	Ftrace("PID actuel: %d", os.Getpid())

	p, _ := GetHelmCmd()
	if p.PID != 0 {
		Ftrace("Helm command line: %s", p.CmdLine)
	} else {
		LogError("helm not found in parent processes")
		os.Exit(1)
	}

	chart, chartVersion, chartRepo := getChart(p.CmdLine)

	if len(os.Args) < 5 {
		LogError("Wrong command line calling Values plugin.")
		os.Exit(1)
	}

	valueFile := strings.Replace(os.Args[4], "chart://", "", 1)

	Fdebug("Fetching %s from chart %s, version \"%s\", repo \"%s\"", valueFile, chart, chartVersion, chartRepo)

	tmpDir, errret := os.MkdirTemp("", "values-downloader-*")
	if errret != nil {
		LogError(fmt.Sprintf("Failed to create temporary directory: %v", errret))
		os.Exit(1)
	}

	defer os.RemoveAll(tmpDir)

	Fdebug("Created temporary directory: %s", tmpDir)

	// Check if we have chart://values.yaml@repo/remotechart
	if strings.Contains(valueFile, "@") {
		parts := strings.Split(valueFile, "@")
		if len(parts) != 2 {
			LogError("Invalid value file format. Expected chart://values.yaml@remotechart")
			os.Exit(1)
		}
		chart = parts[1]
		valueFile = parts[0]
		chartVersion := ""

		// now split chart into chart name and version
		if strings.Contains(chart, ":") {
			chartParts := strings.Split(chart, ":")
			if len(chartParts) == 2 {
				chartVersion = chartParts[1]
				chart = chartParts[0]
			} else {
				LogError("Invalid chart format. Expected chart:version")
				os.Exit(1)
			}
		}

		Fdebug("Using remote chart: %s", chart)
		getRemote(chart, valueFile, chartVersion, "", tmpDir)
	} else {
		// Check if it is a local chart or a remote chart. Guess it is a local chart
		// if we find a Chart.yaml in the path
		chartYamlPath := chart + string(os.PathSeparator) + "Chart.yaml"
		if _, statErr := os.Stat(chartYamlPath); os.IsNotExist(statErr) {
			getRemote(chart, valueFile, chartVersion, chartRepo, tmpDir)
		} else {
			getLocal(chart, valueFile, tmpDir)
		}
	}
}
