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

	// Parse the data
	parsedData, err := parseEMVData(emvData)
	if err != nil {
		fmt.Printf("Error parsing EMV data: %v\n", err)
		return
	}

	// Print parsed data with descriptions
	fmt.Println("\n=== Parsed EMV Data ===")
	printEMVDataWithDescriptions(parsedData)

	// Re-encode the data
	reEncodedData, err := marshalEMVData(parsedData)
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
	// Parse the re-encoded data
	reparsedData, err := parseEMVData(reEncodedData)
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
