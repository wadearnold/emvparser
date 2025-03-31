package emvparser

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"
)

// Helper function to test parsing and marshaling of EMV data
func testEMVData(t *testing.T, rawData string) {
	// Decode the raw EMV data from hex string
	gpodata, err := hex.DecodeString(rawData)
	if err != nil {
		t.Fatalf("Error decoding hex: %v", err)
	}

	// Extract the status word (last 2 bytes)
	dataLen := len(gpodata)
	statusWord := gpodata[dataLen-2:]
	emvData := gpodata[:dataLen-2] // Remove status word from EMV data

	fmt.Printf("\n=== Testing EMV Data ===\n")
	fmt.Printf("Original data length: %d bytes\n", len(gpodata))
	fmt.Printf("Status Word: %X\n", statusWord)
	fmt.Printf("Original EMV data: %X\n", emvData)

	// Create a new EMVParser
	parser := NewEMVParser()

	// Parse the data using the parser
	parsedData, err := parser.Parse(emvData)
	if err != nil {
		t.Fatalf("Error parsing EMV data: %v", err)
	}

	// Print parsed data with descriptions
	fmt.Println("\n=== Parsed EMV Data ===")
	printEMVDataWithDescriptions(parsedData)

	// Re-encode the data using the parser
	reEncodedData, err := parser.Marshal(parsedData)
	if err != nil {
		t.Fatalf("Error re-encoding EMV data: %v", err)
	}

	// Print re-encoded data
	fmt.Printf("\nRe-encoded data length: %d bytes\n", len(reEncodedData))
	fmt.Printf("Re-encoded data: %X\n", reEncodedData)

	// Compare original and re-encoded data
	fmt.Println("\n=== Comparison ===")
	compareEMVData(emvData, reEncodedData)

	// Test a round trip with both format-aware parsing and encoding
	fmt.Println("\n=== Round Trip Test ===")
	// Parse the re-encoded data using the parser
	reparsedData, err := parser.Parse(reEncodedData)
	if err != nil {
		t.Fatalf("Error re-parsing EMV data: %v", err)
	}

	// Compare the original struct and re-parsed struct
	compareStructs(parsedData, reparsedData)
}

// Test case for multiple EMV data inputs
func TestMultipleEMVData(t *testing.T) {
	// Test case 1: Original GPO data
	testEMVData(t, "77598202200057134147202500716749D26072011010041301051F5F200F43415244484F4C4445522F564953415F3401019F100706021203A000009F2608D0C669EEB70C58DD9F2701809F360200699F6C0200009F6E04207000009000")

	// Test case 2: New EMV data
	testEMVData(t, "6F30840E325041592E5359532E4444463031A51EBF0C1B61194F07A0000000031010500B56495341204352454449548701019000")
}

// printEMVDataWithDescriptions prints the parsed EMV data with tag descriptions
func printEMVDataWithDescriptions(data *EMVData) {
	v := reflect.ValueOf(data).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := t.Field(i).Name
		tag := t.Field(i).Tag.Get("emv")

		if isZeroValue(field) {
			continue
		}

		// Get the description from EMVTagFormats
		description := "Unknown"
		if format, ok := EMVTagFormats[tag]; ok {
			description = format.Description
		}

		fmt.Printf("Field: %s (Tag: %s - %s)\n", fieldName, tag, description)
		if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Uint8 {
			fmt.Printf("  Value (hex): %X\n", field.Bytes())
		} else if field.Kind() == reflect.String {
			fmt.Printf("  Value: %s\n", field.String())
		}
	}
}

// Compare two EMVData structs
func compareStructs(original, reparsed *EMVData) {
	// Use reflection to compare fields
	v1 := reflect.ValueOf(original).Elem()
	v2 := reflect.ValueOf(reparsed).Elem()
	t := v1.Type()

	diff := false
	fmt.Println("Comparing original struct with re-parsed struct:")

	for i := 0; i < v1.NumField(); i++ {
		field1 := v1.Field(i)
		field2 := v2.Field(i)
		fieldName := t.Field(i).Name
		tag := t.Field(i).Tag.Get("emv")

		// Skip if both are zero value
		if isZeroValue(field1) && isZeroValue(field2) {
			continue
		}

		// Compare based on type
		if field1.Kind() == reflect.Slice && field1.Type().Elem().Kind() == reflect.Uint8 {
			// For byte slices
			bytes1 := field1.Bytes()
			bytes2 := field2.Bytes()

			if !bytesEqual(bytes1, bytes2) {
				if !diff {
					diff = true
				}
				fmt.Printf("  Field '%s' (Tag %s) differs:\n", fieldName, tag)
				fmt.Printf("    Original: %X\n", bytes1)
				fmt.Printf("    Re-parsed: %X\n", bytes2)
			}
		} else if field1.Kind() == reflect.String {
			// For strings
			str1 := field1.String()
			str2 := field2.String()

			if str1 != str2 {
				if !diff {
					diff = true
				}
				fmt.Printf("  Field '%s' (Tag %s) differs:\n", fieldName, tag)
				fmt.Printf("    Original: %s\n", str1)
				fmt.Printf("    Re-parsed: %s\n", str2)
			}
		}
	}

	if !diff {
		fmt.Println("  All fields match between original and re-parsed structs!")
	}
}

// Helper to compare byte slices
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Compare original and re-encoded EMV data
func compareEMVData(original, reEncoded []byte) {
	if len(original) != len(reEncoded) {
		fmt.Printf("Length mismatch: original=%d, re-encoded=%d\n", len(original), reEncoded)
	}

	// Compare each byte
	minLen := len(original)
	if len(reEncoded) < minLen {
		minLen = len(reEncoded)
	}

	diff := false
	for i := 0; i < minLen; i++ {
		if original[i] != reEncoded[i] {
			if !diff {
				fmt.Println("Differences found:")
				diff = true
			}
			fmt.Printf("  Position %d: original=0x%02X, re-encoded=0x%02X\n", i, original[i], reEncoded[i])
		}
	}

	if !diff {
		if len(original) == len(reEncoded) {
			fmt.Println("The re-encoded data exactly matches the original data!")
		} else {
			fmt.Println("The re-encoded data matches the original data up to the minimum length.")
		}
	}

	// Parse both for comparison at TLV level
	fmt.Println("\n=== TLV Comparison ===")
	originalTLVs := extractTLVs(original)
	reEncodedTLVs := extractTLVs(reEncoded)

	// Compare TLVs
	fmt.Println("Original TLVs:")
	for tag, value := range originalTLVs {
		fmt.Printf("  %s: %X\n", tag, value)
	}

	fmt.Println("Re-encoded TLVs:")
	for tag, value := range reEncodedTLVs {
		fmt.Printf("  %s: %X\n", tag, value)
	}

	// Check for missing or different tags
	fmt.Println("\nTLV Differences:")
	diffFound := false

	// Check tags in original but not in re-encoded
	for tag, origValue := range originalTLVs {
		reValue, ok := reEncodedTLVs[tag]
		if !ok {
			fmt.Printf("  Tag %s missing in re-encoded data\n", tag)
			diffFound = true
			continue
		}

		// Compare values
		if !bytesEqual(origValue, reValue) {
			fmt.Printf("  Tag %s value differs:\n", tag)
			fmt.Printf("    Original: %X\n", origValue)
			fmt.Printf("    Re-encoded: %X\n", reValue)
			diffFound = true
		}
	}

	// Check tags in re-encoded but not in original
	for tag := range reEncodedTLVs {
		if _, ok := originalTLVs[tag]; !ok {
			fmt.Printf("  Extra tag %s in re-encoded data\n", tag)
			diffFound = true
		}
	}

	if !diffFound {
		fmt.Println("  No TLV differences found!")
	}
}

func TestMarshalExcludesNonDE55Tags(t *testing.T) {
	// Create an EMVData instance with values for both DE55 and non-DE55 tags
	data := &EMVData{
		ResponseMessageTemplate:       []byte{0x01, 0x02, 0x03},                   // Tag 77 (non-DE55)
		ApplicationIdentifier:         []byte{0xA0, 0x00, 0x00, 0x03, 0x10, 0x10}, // Tag 4F (non-DE55)
		IssuerAppData:                 []byte{0x12, 0x34, 0x56},                   // Tag 9F10 (DE55)
		ApplicationCryptogram:         []byte{0xAB, 0xCD, 0xEF},                   // Tag 9F26 (DE55)
		ApplicationTransactionCounter: []byte{0x00, 0x01},                         // Tag 9F36 (DE55)
		FileControlInformation:        []byte{0x6F, 0x01},                         // Tag 6F (non-DE55)
	}

	// Create a new EMVParser
	parser := NewEMVParser()

	// Marshal the data
	marshaledData, err := parser.Marshal(data)
	if err != nil {
		t.Fatalf("Error marshaling EMV data: %v", err)
	}

	// Decode the marshaled data into a map of tags
	decodedTags := extractTLVs(marshaledData)

	// Verify that non-DE55 tags are not present
	nonDE55Tags := []string{"77", "6F", "BF0C", "A5"}
	for _, tag := range nonDE55Tags {
		if _, exists := decodedTags[tag]; exists {
			t.Errorf("Tag %s should not appear in the output", tag)
		}
	}

	// Verify that DE55 tags are present
	de55Tags := []string{"9F10", "9F26", "9F36"}
	for _, tag := range de55Tags {
		if _, exists := decodedTags[tag]; !exists {
			t.Errorf("Tag %s should appear in the output", tag)
		}
	}
}
func TestGetEMVPropertyByTag(t *testing.T) {
	// Create an EMVParser instance
	parser := NewEMVParser()

	// Populate the EMVData instance with some test data
	parser.data = &EMVData{
		IssuerAppData:                 []byte{0x12, 0x34, 0x56}, // Tag 9F10
		ApplicationTransactionCounter: []byte{0x00, 0x01},       // Tag 9F36
		ApplicationLabel:              "VISA",                   // Tag 50
	}

	// Test case 1: Retrieve an existing tag (9F10 - Issuer Application Data)
	tag := "9F10"
	value, err := parser.GetEMVPropertyByTag(tag)
	if err != nil {
		t.Fatalf("Error retrieving tag %s: %v", tag, err)
	}
	expectedValue := []byte{0x12, 0x34, 0x56}
	if string(value) != string(expectedValue) {
		t.Errorf("Expected value for tag %s: %X, got: %X", tag, expectedValue, value)
	} else {
		fmt.Printf("Tag %s retrieved successfully: %X\n", tag, value)
	}

	// Test case 2: Retrieve another existing tag (50 - Application Label)
	tag = "50"
	value, err = parser.GetEMVPropertyByTag(tag)
	if err != nil {
		t.Fatalf("Error retrieving tag %s: %v", tag, err)
	}
	expectedLabel := "VISA"
	if string(value) != expectedLabel {
		t.Errorf("Expected value for tag %s: %s, got: %s", tag, expectedLabel, string(value))
	} else {
		fmt.Printf("Tag %s retrieved successfully: %s\n", tag, string(value))
	}
}
