package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

var version string

type TestData struct {
	PackageName    string
	TestName       string
	Method         string
	Path           string
	Summary        string
	Description    string
	Responses      []ResponseCode
	CustomComments []string
}

type ResponseCode struct {
	Code        string
	Description string
	Method      string
	Path        string
}

type OperationInfo struct {
	ID          string
	Method      string
	Path        string
	Tag         string
	Summary     string
	Description string
	Implemented bool
}

const testTemplate = `package {{.PackageName}}_test

import "testing"

// {{.Method}} {{.Path}}
{{if .Summary}}// Summary: {{.Summary}}{{end}}
{{if .Description}}// Description: {{.Description}}{{end}}
func {{.TestName}}(t *testing.T) {
	t.Parallel()

	// TODO:{{if .CustomComments}}{{range .CustomComments}}{{if .}}
	// {{.}}{{else}}
{{end}}{{end}}{{end}}
	t.Skip("not implemented")
{{range .Responses}}
	t.Run("{{.Code}}_{{.Method}}_{{.Path}}", func(t *testing.T) {
		t.Parallel()

		// TODO - {{.Description}}
		t.Skip("not implemented")
	})
{{end}}
}
`

var (
	openapiFile       *string
	operationID       *string
	outputDir         *string
	commentsFile      *string
	overwrite         *bool
	unimplementedOnly *bool
)

func usage() {
	fmt.Fprintf(os.Stderr, "railgen - Generate test rails from OpenAPI specification\n\n")
	fmt.Fprintf(os.Stderr, "A CLI tool to generate Go test files from OpenAPI operation IDs\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  %s <command> [flags]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  generate    Generate test files from operation ID\n")
	fmt.Fprintf(os.Stderr, "  delete      Delete test files for operation ID\n")
	fmt.Fprintf(os.Stderr, "  list        List operation IDs and their implementation status\n")
	fmt.Fprintf(os.Stderr, "  help        Show help for commands\n\n")
	fmt.Fprintf(os.Stderr, "Global Flags:\n")
	fmt.Fprintf(os.Stderr, "  -v, -version    Show version information\n\n")
	fmt.Fprintf(os.Stderr, "Use \"%s <command> -h\" for more information about a command.\n", os.Args[0])
}

func printCommonFlags() {
	fmt.Fprintf(os.Stderr, "  -h, -help\n")
	fmt.Fprintf(os.Stderr, "        Show this help message\n")
	fmt.Fprintf(os.Stderr, "  -f, -file string\n")
	fmt.Fprintf(os.Stderr, "        OpenAPI specification file (default \"openapi.yaml\")\n")
	fmt.Fprintf(os.Stderr, "  -d, -output string\n")
	fmt.Fprintf(os.Stderr, "        Output directory for generated tests (default \"test\")\n")
}

func generateUsage() {
	fmt.Fprintf(os.Stderr, "Generate test files from OpenAPI operation ID\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  %s generate [flags]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Flags:\n")
	printCommonFlags()
	fmt.Fprintf(os.Stderr, "  -o, -operation string\n")
	fmt.Fprintf(os.Stderr, "        Operation ID to generate test for\n")
	fmt.Fprintf(os.Stderr, "  -c, -comments string\n")
	fmt.Fprintf(os.Stderr, "        Comments file to include custom TODO comments\n")
	fmt.Fprintf(os.Stderr, "  --overwrite\n")
	fmt.Fprintf(os.Stderr, "        Overwrite existing test file (creates backup)\n")
}

func deleteUsage() {
	fmt.Fprintf(os.Stderr, "Delete test files for OpenAPI operation ID\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  %s delete [flags]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Flags:\n")
	printCommonFlags()
	fmt.Fprintf(os.Stderr, "  -o, -operation string\n")
	fmt.Fprintf(os.Stderr, "        Operation ID to delete test for\n")
}

func listUsage() {
	fmt.Fprintf(os.Stderr, "List operation IDs and their implementation status\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  %s list [flags]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Flags:\n")
	printCommonFlags()
	fmt.Fprintf(os.Stderr, "  --unimplemented\n")
	fmt.Fprintf(os.Stderr, "        Show only unimplemented operation IDs\n")
}

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "-v" || arg == "-version" {
			printVersion()
			os.Exit(0)
		}
	}

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "generate":
		handleGenerate()
	case "delete":
		handleDelete()
	case "list":
		handleList()
	case "help", "-h", "-help":
		usage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'\n\n", command)
		usage()
		os.Exit(1)
	}
}

func printVersion() {
	if version == "" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			version = bi.Main.Version
		}
	}

	fmt.Printf("railgen version %s\n", version)
}

func handleGenerate() {
	generateFlagSet := flag.NewFlagSet("generate", flag.ExitOnError)
	generateFlagSet.Usage = generateUsage

	initGenerateFlags(generateFlagSet)
	if err := generateFlagSet.Parse(os.Args[2:]); err != nil {
		os.Exit(1)
	}

	if *openapiFile == "" || *operationID == "" {
		fmt.Fprintf(os.Stderr, "Error: operation ID is required\n\n")
		generateUsage()
		os.Exit(1)
	}

	if err := generateTest(*overwrite); err != nil {
		log.Fatal(err)
	}
}

func handleDelete() {
	deleteFlagSet := flag.NewFlagSet("delete", flag.ExitOnError)
	deleteFlagSet.Usage = deleteUsage

	initDeleteFlags(deleteFlagSet)
	if err := deleteFlagSet.Parse(os.Args[2:]); err != nil {
		os.Exit(1)
	}

	if *openapiFile == "" || *operationID == "" {
		fmt.Fprintf(os.Stderr, "Error: operation ID is required\n\n")
		deleteUsage()
		os.Exit(1)
	}

	if err := deleteTest(); err != nil {
		log.Fatal(err)
	}
}

func handleList() {
	listFlagSet := flag.NewFlagSet("list", flag.ExitOnError)
	listFlagSet.Usage = listUsage

	initListFlags(listFlagSet)
	if err := listFlagSet.Parse(os.Args[2:]); err != nil {
		os.Exit(1)
	}

	if *openapiFile == "" {
		fmt.Fprintf(os.Stderr, "Error: OpenAPI file is required\n\n")
		listUsage()
		os.Exit(1)
	}

	if err := listOperations(*unimplementedOnly); err != nil {
		log.Fatal(err)
	}
}

func initCommonFlags(fs *flag.FlagSet) {
	openapiFile = fs.String("f", "openapi.yaml", "OpenAPI specification file")
	fs.StringVar(openapiFile, "file", "openapi.yaml", "OpenAPI specification file")

	outputDir = fs.String("d", "test", "Output directory for generated tests")
	fs.StringVar(outputDir, "output", "test", "Output directory for generated tests")
}

func initGenerateFlags(fs *flag.FlagSet) {
	initCommonFlags(fs)

	operationID = fs.String("o", "", "Operation ID to generate test for")
	fs.StringVar(operationID, "operation", "", "Operation ID to generate test for")

	commentsFile = fs.String("c", "", "Comments file to include custom TODO comments")
	fs.StringVar(commentsFile, "comments", "", "Comments file to include custom TODO comments")

	overwrite = fs.Bool("overwrite", false, "Overwrite existing test file (creates backup)")
}

func initDeleteFlags(fs *flag.FlagSet) {
	initCommonFlags(fs)

	operationID = fs.String("o", "", "Operation ID to delete test for")
	fs.StringVar(operationID, "operation", "", "Operation ID to delete test for")
}

func initListFlags(fs *flag.FlagSet) {
	initCommonFlags(fs)
	unimplementedOnly = fs.Bool("unimplemented", false, "Show only unimplemented operation IDs")
}

func generateTest(overwrite bool) error {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(*openapiFile)
	if err != nil {
		return fmt.Errorf("failed to load OpenAPI file: %w", err)
	}

	var operation *openapi3.Operation
	var tag string
	var method string
	var path string
	var found bool

	for apiPath, pathItem := range doc.Paths.Map() {
		for httpMethod, op := range pathItem.Operations() {
			if op.OperationID == *operationID {
				operation = op
				method = strings.ToUpper(httpMethod)
				path = apiPath
				found = true
				if len(op.Tags) > 0 {
					tag = op.Tags[0]
				}
				fmt.Printf("Found operation %s %s with operationId: %s\n", method, path, *operationID)
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		return fmt.Errorf("operation with ID '%s' not found", *operationID)
	}

	customComments, err := loadCustomComment(*commentsFile)
	if err != nil {
		return fmt.Errorf("failed to load custom comment: %w", err)
	}

	testData := TestData{
		PackageName:    sanitizePackageName(tag),
		TestName:       "Test" + toPascalCase(*operationID),
		Method:         method,
		Path:           path,
		Summary:        operation.Summary,
		Description:    operation.Description,
		Responses:      []ResponseCode{},
		CustomComments: customComments,
	}

	var codes []string
	for code := range operation.Responses.Map() {
		if code != "default" {
			codes = append(codes, code)
		}
	}
	sort.Strings(codes)

	for _, code := range codes {
		response := operation.Responses.Map()[code]
		description := ""
		if response.Value != nil && response.Value.Description != nil {
			description = *response.Value.Description
		}
		testData.Responses = append(testData.Responses, ResponseCode{
			Code:        code,
			Description: description,
			Method:      method,
			Path:        path,
		})
	}

	tagDir := filepath.Join(*outputDir, testData.PackageName)
	if err := os.MkdirAll(tagDir, fs.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	tmpl, err := template.New("test").Parse(testTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	fileName := fmt.Sprintf("%s_test.go", toSnakeCase(*operationID))
	filePath := filepath.Join(tagDir, fileName)

	if _, err := os.Stat(filePath); err == nil {
		if !overwrite {
			return fmt.Errorf("test file already exists: %s\nUse --overwrite to overwrite the existing file", filePath)
		}

		backupPath := fmt.Sprintf("%s.backup.%s", filePath, time.Now().Format("20060102-150405"))
		if err := copyFile(filePath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("Created backup: %s\n", backupPath)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create test file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file: %v\n", err)
		}
	}()

	if err := tmpl.Execute(file, testData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Printf("Generated test file: %s\n", filePath)
	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := sourceFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close source file: %v\n", err)
		}
	}()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if err := destFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close destination file: %v\n", err)
		}
	}()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return nil
}

func deleteTest() error {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(*openapiFile)
	if err != nil {
		return fmt.Errorf("failed to load OpenAPI file: %w", err)
	}

	var tag string
	var found bool

	for _, pathItem := range doc.Paths.Map() {
		for _, op := range pathItem.Operations() {
			if op.OperationID == *operationID {
				found = true
				if len(op.Tags) > 0 {
					tag = op.Tags[0]
				}
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		return fmt.Errorf("operation with ID '%s' not found", *operationID)
	}

	packageName := sanitizePackageName(tag)
	tagDir := filepath.Join(*outputDir, packageName)
	fileName := fmt.Sprintf("%s_test.go", toSnakeCase(*operationID))
	filePath := filepath.Join(tagDir, fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("test file does not exist: %s", filePath)
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete test file: %w", err)
	}

	fmt.Printf("Deleted test file: %s\n", filePath)

	if err := os.Remove(tagDir); err == nil {
		fmt.Printf("Removed empty directory: %s\n", tagDir)
	}

	return nil
}

func sanitizePackageName(tag string) string {
	if tag == "" {
		return "api"
	}
	result := strings.ToLower(tag)
	result = strings.ReplaceAll(result, "-", "_")
	result = strings.ReplaceAll(result, " ", "_")
	return result
}

func loadCustomComment(commentsFile string) ([]string, error) {
	if commentsFile == "" {
		return nil, nil
	}

	content, err := os.ReadFile(commentsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read comments file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	return lines, nil
}

func toPascalCase(s string) string {
	if s == "" {
		return ""
	}

	parts := strings.FieldsFunc(s, func(c rune) bool {
		return c == '_' || c == '-' || c == ' '
	})

	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}

	return strings.Join(parts, "")
}

func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}

	return strings.ToLower(result.String())
}

func listOperations(unimplementedOnly bool) error {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(*openapiFile)
	if err != nil {
		return fmt.Errorf("failed to load OpenAPI file: %w", err)
	}

	var operations []OperationInfo
	for apiPath, pathItem := range doc.Paths.Map() {
		for httpMethod, op := range pathItem.Operations() {
			if op.OperationID == "" {
				continue
			}

			tag := ""
			if len(op.Tags) > 0 {
				tag = op.Tags[0]
			}

			operations = append(operations, OperationInfo{
				ID:          op.OperationID,
				Method:      strings.ToUpper(httpMethod),
				Path:        apiPath,
				Tag:         tag,
				Summary:     op.Summary,
				Description: op.Description,
				Implemented: false,
			})
		}
	}

	for i := range operations {
		op := &operations[i]
		packageName := sanitizePackageName(op.Tag)
		fileName := fmt.Sprintf("%s_test.go", toSnakeCase(op.ID))
		filePath := filepath.Join(*outputDir, packageName, fileName)

		if _, err := os.Stat(filePath); err == nil {
			op.Implemented = true
		}
	}

	sort.Slice(operations, func(i, j int) bool {
		if operations[i].Tag != operations[j].Tag {
			return operations[i].Tag < operations[j].Tag
		}
		return operations[i].ID < operations[j].ID
	})

	if unimplementedOnly {
		fmt.Println("Unimplemented Operation IDs:")
		fmt.Println("============================")
		count := 0

		currentTag := ""
		for _, op := range operations {
			if !op.Implemented {
				if op.Tag != currentTag {
					if currentTag != "" {
						fmt.Println()
					}
					fmt.Printf("[%s]\n", op.Tag)
					fmt.Println(strings.Repeat("-", len(op.Tag)+2))
					currentTag = op.Tag
				}

				fmt.Printf("* %s\n", op.ID)
				fmt.Printf("  %s %s\n", op.Method, op.Path)
				fmt.Println()
				count++
			}
		}
		if count == 0 {
			fmt.Println("All operations have been implemented!")
		} else {
			fmt.Printf("Total unimplemented: %d\n", count)
		}
	} else {
		fmt.Println("All Operation IDs:")
		fmt.Println("==================")
		implementedCount := 0
		totalCount := len(operations)

		currentTag := ""
		for _, op := range operations {
			if op.Tag != currentTag {
				if currentTag != "" {
					fmt.Println()
				}
				fmt.Printf("[%s]\n", op.Tag)
				fmt.Println(strings.Repeat("-", len(op.Tag)+2))
				currentTag = op.Tag
			}

			status := "[ ]"
			if op.Implemented {
				status = "[x]"
				implementedCount++
			}

			fmt.Printf("%s %s\n", status, op.ID)
			fmt.Printf("    %s %s\n", op.Method, op.Path)
			fmt.Println()
		}

		fmt.Printf("Implementation Status: %d/%d (%.1f%%)\n",
			implementedCount, totalCount, float64(implementedCount)/float64(totalCount)*100)
	}

	return nil
}
