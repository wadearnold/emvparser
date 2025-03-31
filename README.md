# EMV Data Parser and Marshal Library

This library provides functionality to parse and marshal EMV (Europay, Mastercard, and Visa) data, specifically focusing on data element 55 (DE55) of ISO 8583 messages. It allows users to parse raw EMV TLV (Tag-Length-Value) data into a structured format (`EMVData`) and export it back into a TLV format suitable for inclusion in DE55.

## Features

- **Parse EMV Data**: The `Parse` function processes raw EMV TLV data and populates the `EMVData` struct with parsed fields.
- **Marshal EMV Data**: The `Marshal` function exports the `EMVData` struct into a TLV format, including only the fields required for DE55.
- **Support for DE55 Filtering**: Tags that are not part of DE55 (e.g., composite tags like `77`, `6F`, `BF0C`, `A5`) are excluded during marshaling.
- **Customizable Tag Formats**: The library uses the `EMVTagFormats` map to define the expected format, description, and DE55 inclusion for each tag.

## Installation

To use this library, you can clone the repository or include it in your Go project:

```bash
git clone https://github.com/wadearnold/emvparser.git
```

## Usage

### Parsing EMV Data

The `Parse` function takes raw EMV TLV data as input and populates the `EMVData` struct with the parsed fields.

```go
parser := NewEMVParser()
rawData := []byte{...} // Raw EMV TLV data
parsedData, err := parser.Parse(rawData)
if err != nil {
    log.Fatalf("Error parsing EMV data: %v", err)
}

// Access parsed fields
fmt.Printf("Issuer Application Data: %X\n", parsedData.IssuerAppData)
```

### Example Workflow

1. Parse raw EMV TLV data:

```go
rawData := "6F30840E325041592E5359532E4444463031A51EBF0C1B61194F07A0000000031010500B56495341204352454449548701019000"
parsedData, err := parser.Parse([]byte(rawData))
if err != nil {
    log.Fatalf("Error parsing EMV data: %v", err)
}
```

2. Access parsed fields:

```go
fmt.Printf("Application Identifier (AID): %X\n", parsedData.ApplicationIdentifier)
```

3. Marshal the data for DE55:

```go
marshaledData, err := parser.Marshal(parsedData)
if err != nil {
    log.Fatalf("Error marshaling EMV data: %v", err)
}
fmt.Printf("DE55 Data: %X\n", marshaledData)
```

4. Access parsed fields by tag name: 

```go 
tag := "9F10"
value, err := parser.GetEMVPropertyByTag(tag)
if err != nil {
	fmt.Printf("Error retrieving tag %s: %v", tag, err)
}
