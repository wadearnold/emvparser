package kernel

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"
)

// TestMarshalEMVData tests the EMV data marshalling
func TestMarshalEMVData(t *testing.T) {
	// Your GPO data
	gpodata, err := hex.DecodeString("77598202200057134147202500716749D26072011010041301051F5F200F43415244484F4C4445522F564953415F3401019F100706021203A000009F2608D0C669EEB70C58DD9F2701809F360200699F6C0200009F6E04207000009000")
	if err != nil {
		fmt.Printf("Error decoding hex: %v\n", err)
		return
	}

	// Extract the status word (last 2 bytes)
	dataLen := len(gpodata)
	statusWord := gpodata[dataLen-2:]
	emvData := gpodata[:dataLen-2] // Remove status word from EMV data

	fmt.Printf("Original data length: %d bytes\n", len(gpodata))
	fmt.Printf("Status Word: %X\n", statusWord)
	fmt.Printf("Original EMV data: %X\n", emvData)

	// Create a new EMVParser
	parser := NewEMVParser()

	// Parse the data using the parser
	parsedData, err := parser.Parse(emvData)
	if err != nil {
		fmt.Printf("Error parsing EMV data: %v\n", err)
		return
	}

	// Print parsed data with descriptions
	fmt.Println("\n=== Parsed EMV Data ===")
	printEMVDataWithDescriptions(parsedData)

	// Re-encode the data using the parser
	reEncodedData, err := parser.Marshal(parsedData)
	if err != nil {
		fmt.Printf("Error re-encoding EMV data: %v\n", err)
		return
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
		fmt.Printf("Error re-parsing EMV data: %v\n", err)
		return
	}

	// Compare the original struct and re-parsed struct
	compareStructs(parsedData, reparsedData)
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
