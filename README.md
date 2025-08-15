# railgen

A CLI tool for automatically generating Go test files from OpenAPI specifications.

## Install

```shell
go install github.com/otakakot/railgen@latest
```

## Features

- Generate test files from OpenAPI 3.0 specifications by specifying operationId
- Organize test files into directories by tag
- Automatically generate test cases for each response code
- Include HTTP method and API path in comments
- Include summary and description as comments
- **Custom Comment File Support** - Insert custom TODO comments into tests
- **Existing File Protection** - Prevent accidental overwriting of existing test files
- **Backup Functionality** - Automatically create backup files when overwriting
- Delete generated test files
- List operationIds and their implementation status

## Usage

### Commands

railgen supports the following commands:

- `generate`: Generate test files from OpenAPI operationId
- `delete`: Delete test files for specified operationId
- `list`: List operationIds and their implementation status
- `help`: Show help information

### Generate Test Files

```bash
railgen generate -o <operationId>
```

#### Using Custom Comment Files

```bash
railgen generate -o <operationId> -c comments.txt
```

#### Overwrite Existing Files (with Backup)

```bash
railgen generate -o <operationId> --overwrite
```

### Delete Test Files

```bash
railgen delete -o <operationId>
```

### List OperationIds

```bash
# Show all operationIds with implementation status
railgen list

# Show only unimplemented operationIds
railgen list --unimplemented
```

### Options

#### generate/delete Commands

- `-f, -file`: OpenAPI specification file path (default: openapi.yaml)
- `-o, -operation`: Target operationId (required)
- `-d, -output`: Output directory (default: test)
- `-c, -comment`: Custom comment file path (optional)
- `--overwrite`: Overwrite existing test file (creates backup)
- `-h, --help`: Show help message

#### list Command

- `-f, -file`: OpenAPI specification file path (default: openapi.yaml)
- `-d, -output`: Output directory (default: test)
- `--unimplemented`: Show only unimplemented operationIds
- `-h, --help`: Show help message

### Examples

```bash
# Generate test for addPet operation
railgen generate -o addPet

# Generate test using custom comment file
railgen generate -o addPet -c comments.txt

# Use specified OpenAPI file
railgen generate -f my-api.yaml -o createUser

# Specify output directory
railgen generate -o placeOrder -d generated-tests

# Overwrite existing file (with backup)
railgen generate -o addPet --overwrite

# Combine custom comments and overwrite
railgen generate -o addPet -c comments.txt --overwrite

# Delete test file
railgen delete -o addPet

# Show all operationIds and implementation status
railgen list

# Show only unimplemented operationIds
railgen list --unimplemented

# Show help
railgen help
railgen generate -h
railgen generate --help
railgen delete -h
railgen delete --help
railgen list -h
railgen list --help
```

## Custom Comment Files

railgen allows you to use custom comment files to insert your own TODO comments into generated test files.

### Creating Comment Files

Create a `comments.txt` file and write the comment content you want to insert:

```txt
# Common instructions for all tests
# The content of this file applies to all operationId tests

Fix tests until they pass.
Do not modify other files.
Focus on testing validation logic
Verify response format
- Don't forget error handling
- Consider performance testing
```

### Using Comment Files

```bash
railgen generate -o addPet -c comments.txt
```

### Notes

- Comment files are added to `.gitignore`, allowing each developer to have their own settings
- Comment file content is automatically inserted after `// TODO:`
- Each line automatically gets comment symbols (`//`) prepended

## Existing File Protection and Backup

railgen provides the following features to prevent accidental overwriting of existing test files:

### Existing File Check

When an existing test file is found, generation is stopped by default:

```bash
$ railgen generate -o addPet
Found operation POST /pet with operationId: addPet
test file already exists: test/pet/add_pet_test.go
Use --overwrite to overwrite the existing file
```

### Force Overwrite and Backup

You can overwrite existing files using the `--overwrite` option. When doing so, a backup file is automatically created:

```bash
$ railgen generate -o addPet --overwrite
Found operation POST /pet with operationId: addPet
Created backup: test/pet/add_pet_test.go.backup.20250803-191047
Generated test file: test/pet/add_pet_test.go
```

### Backup File Format

Backup files are created in the following format:
- `{original_filename}.backup.{YYYYMMDD-HHMMSS}`
- Example: `add_pet_test.go.backup.20250803-191047`

## Generated File Structure

```
test/
├── pet/
│   ├── add_pet_test.go
│   └── find_pets_by_status_test.go
├── store/
│   └── place_order_test.go
└── user/
    └── create_user_test.go
```

## Generated Test File Examples

### Basic Test File (without custom comments)

```go
package pet_test

import "testing"

// POST /pet
// Summary: Add a new pet to the store.
// Description: Add a new pet to the store.
func TestAddPet(t *testing.T) {
	t.Parallel()

	// TODO
	t.Skip("not implemented")

	t.Run("200_POST_/pet", func(t *testing.T) {
		t.Parallel()

		// TODO - Successful operation
		t.Skip("not implemented")
	})

	t.Run("400_POST_/pet", func(t *testing.T) {
		t.Parallel()

		// TODO - Invalid input
		t.Skip("not implemented")
	})

	t.Run("422_POST_/pet", func(t *testing.T) {
		t.Parallel()

		// TODO - Validation exception
		t.Skip("not implemented")
	})
}
```

### With Custom Comment File

```go
package pet_test

import "testing"

// POST /pet
// Summary: Add a new pet to the store.
// Description: Add a new pet to the store.
func TestAddPet(t *testing.T) {
	t.Parallel()

	// TODO:
	// # Common instructions for all tests
	// # The content of this file applies to all operationId tests

	// Fix tests until they pass.
	// Do not modify other files.
	// Focus on testing validation logic
	// Verify response format
	// - Don't forget error handling
	// - Consider performance testing

	t.Skip("not implemented")

	t.Run("200_POST_/pet", func(t *testing.T) {
		t.Parallel()

		// TODO - Successful operation
		t.Skip("not implemented")
	})

	t.Run("400_POST_/pet", func(t *testing.T) {
		t.Parallel()

		// TODO - Invalid input
		t.Skip("not implemented")
	})

	t.Run("422_POST_/pet", func(t *testing.T) {
		t.Parallel()

		// TODO - Validation exception
		t.Skip("not implemented")
	})
}
```

## List Command Output Examples

### Show All OperationIds

```bash
$ railgen list
All Operation IDs:
==================

[pet]
-----
[x] findPetsByStatus
    GET /pet/findByStatus

[ ] addPet
    POST /pet

[ ] updatePet
    PUT /pet

[store]
-------
[x] getInventory
    GET /store/inventory

[ ] placeOrder
    POST /store/order

Implementation Status: 2/5 (40.0%)
```

### Show Only Unimplemented OperationIds

```bash
$ railgen list --unimplemented
Unimplemented Operation IDs:
============================

[pet]
-----
* addPet
  POST /pet

* updatePet
  PUT /pet

[store]
-------
* placeOrder
  POST /store/order

Total unimplemented: 3
```

---
