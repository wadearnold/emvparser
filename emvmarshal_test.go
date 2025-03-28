package kernel

import (
	"encoding/hex"
	"fmt"
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

	// Print parsed data
	fmt.Println("\n=== Parsed EMV Data ===")
	printEMVData(parsedData)

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
