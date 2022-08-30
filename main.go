package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	zlog "github.com/rs/zerolog/log"
)

/**
 * Auxiliary types.
 */

type LogLine struct {
	Date     string `json:"date"`
	Message  string `json:"message"`
	PrNumber string `json:"prNumber"`
}

type Tag struct {
	Date  string    `json:"date"`
	Logs  []LogLine `json:"logs"`
	Value string    `json:"value"`
}

/**
 * Main method.
 */

func main() {
	handleArgs(os.Args[1:])

	fh, err := os.OpenFile("CHANGELOG.md", os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)

	if err != nil {
		handleError("Could not open changelog file", nil)
	}

	defer fh.Close()

	fh.WriteString("# Changelog\n\n")

	tagList, logData := getContent()
	generate(fh, tagList, logData)
}

/**
 * Generate changelog file.
 */

func generate(fh *os.File, tagList []string, logData map[string]*Tag) {
	var tags []string
	tags = append(tags, tagList...)
	firstCommit := getFirstCommit()
	baseUrl := getGithubRepositoryUrl()

	for len(tags) > 0 {
		current := tags[0]
		tags = tags[1:]
		tagContent := logData[current]

		fh.WriteString("## [" + current + "](" + baseUrl + "/tree/" + current + ") (" + tagContent.Date + ")\n\n")

		if len(tags) > 0 {
			fh.WriteString("[Full Changelog](" + baseUrl + "/compare/" + tags[0] + "..." + current + ")\n\n")
		} else {
			fh.WriteString("[Full Changelog](" + baseUrl + "/compare/" + firstCommit + "..." + current + ")\n\n")
		}

		for _, line := range tagContent.Logs {
			fh.WriteString("- " + line.Message)
			fh.WriteString(" [\\#" + line.PrNumber + "](" + baseUrl + "/pull/" + line.PrNumber + ")\n")
		}

		fh.WriteString("\n\n")
	}
}

/**
 * Organize and get git log content.
 */

func getContent() ([]string, map[string]*Tag) {
	logCommand := runCommand("git log --tags --merges --pretty=format:\"%b*%s*%d*%cs\" --abbrev-commit")
	logLines := strings.Split(logCommand, "\n")
	splitRegex := regexp.MustCompile(`\*`)
	prRegex := regexp.MustCompile(`Merge pull request #([0-9]+)`)
	tagRegex := regexp.MustCompile(`tag: (v?[0-9\.]+)`)
	currentTag := ""

	tagList := []string{}
	logData := make(map[string]*Tag)

	for _, line := range logLines {
		fmt.Println(line)
		splitLine := splitRegex.Split(line, 6)
		newTag := tagRegex.FindStringSubmatch(splitLine[2])

		if newTag == nil && currentTag == "" {
			handleError("Commit not after a tag", nil)
		}

		if newTag != nil {
			currentTag = strings.TrimSpace(newTag[1])
			tagList = append(tagList, currentTag)
		}

		prMessage := prRegex.FindStringSubmatch(splitLine[1])

		if prMessage == nil {
			continue
		}

		message := strings.TrimSpace(splitLine[0])
		prNumber := strings.TrimSpace(prMessage[1])
		date := strings.TrimSpace(splitLine[3])
		logLine := LogLine{
			Date:     date,
			Message:  message,
			PrNumber: prNumber,
		}

		if logData[currentTag] == nil {
			logData[currentTag] = &Tag{Date: date, Logs: []LogLine{}}
		}

		logData[currentTag].Logs = append(logData[currentTag].Logs, logLine)
	}

	return tagList, logData
}

/**
 * Get GitHub repository url.
 */

func getGithubRepositoryUrl() string {
	fetchUrlCommand := runCommand("git remote show origin | grep 'Fetch URL'")
	urlRegex := regexp.MustCompile(`.*(https://github\.com/|git@github\.com:)(.*)\.git`)

	return "https://github.com/" + urlRegex.FindStringSubmatch(fetchUrlCommand)[2]
}

/**
 * Get first commit hash.
 */

func getFirstCommit() string {
	return strings.TrimSpace(runCommand("git rev-list HEAD | tail -n 1"))
}

/**
 * Handle command arguments.
 */

func handleArgs(args []string) {
	if len(args) == 0 {
		handleError("Missing mandatory parameters", nil)
	}

	if args[0] == "help" || args[0] == "-h" {
		help()
	}

	if args[0] != "generate" && args[0] != "-g" {
		handleError("Unsupported command", nil)
	}

	_, err := os.ReadDir(".git")

	if err != nil {
		handleError("Unable to find git folder", nil)
	}
}

/**
 * Run generic command.
 */

func runCommand(command string) string {
	out, err := exec.Command("bash", "-c", command).Output()
	result := string(out)

	if err != nil {
		handleError("Error running command", err)
	}

	return result
}

/**
 * Handle errors.
 */

func handleError(message string, err error) {
	zlog.Error().Err(err).Msg(message)
	os.Exit(1)
}

/**
 * Help print method.
 */

func help() {
	fmt.Println("> changelog-generator generate")
	os.Exit(0)
}

/**
 * Print object auxiliary method.
 */

func Print(elem interface{}) {
	content, _ := json.Marshal(elem)
	fmt.Println(string(content))
}
