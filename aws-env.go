package main

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func main() {

	keys := strings.Split(os.Getenv("SSM_PATH"), "/")
	params := make(map[string]string)

	// Remove the empty string created by the split
	if keys[0] == "" {
		keys = keys[1:]
	}

	path := ""
	// Loop through the sub paths and retrieve parameters
	for i := range keys {
		path = path + "/" + keys[i]
		log.Printf("Retriving parameters in path %s", path)
		ExportVariables(path, "", params)
	}

	ParametersToFile(params)
}

func CreateClient() *ssm.SSM {
	session := session.Must(session.NewSession())
	return ssm.New(session)
}

func ExportVariables(path string, nextToken string, params map[string]string ) {
	client := CreateClient()

	input := &ssm.GetParametersByPathInput{
		Path:           &path,
		WithDecryption: aws.Bool(true),
	}

	if nextToken != "" {
		input.SetNextToken(nextToken)
	}

	output, err := client.GetParametersByPath(input)

	if err != nil {
		log.Panic(err)
	}

	for _, element := range output.Parameters {
		env, value := TrimParameter(path, element)
		params[env] = value
	}

	if output.NextToken != nil {
		ExportVariables(path, *output.NextToken, params)
	}
}

func TrimParameter(path string, parameter *ssm.Parameter) (string, string) {
	name := *parameter.Name
	value := *parameter.Value

	env := strings.Trim(name[len(path):], "/")
	value = strings.Replace(value, "\n", "\\n", -1)

	return env, value
}

func ParametersToFile(params map[string]string) {
	var buffer bytes.Buffer
	format := os.Getenv("FORMAT")

	for key, value := range params {
		buffer.WriteString(FormatParameter(key,value,format))
	}

	dir := os.Getenv("DIRECTORY")
	if dir == "" {
		dir = "/ssm"
	}
	// Create /ssm directory if it doesn't exist
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Printf("Creating directory %s", dir)
		os.MkdirAll(dir, 0755)
	}

	// Write evironment variables to .env file
	err := ioutil.WriteFile(dir + "/.env", buffer.Bytes(), 0744)
	if err != nil {
		log.Panic(err)
	}
}

func FormatParameter(key string, value string, format string) string {
	switch format {
	case "shell":
		return fmt.Sprintf("%s='%s'\n", key, value)
	default:
		return fmt.Sprintf("export %s=$'%s'\n", key, value)
	}
}
