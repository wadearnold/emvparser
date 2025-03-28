package kernel

import (
	"encoding/hex"
	"fmt"
	"reflect"
)

// EMVData represents a parsed EMV record with fields mapped to EMV tags
type EMVData struct {
	ResponseMessageTemplate1 []byte `emv:"77" json:"responseMessageTemplate1"`
	AIP                      []byte `emv:"82" json:"applicationInterchangeProfile"`
	TrackData                []byte `emv:"57" json:"track2EquivalentData"`
	CardholderName           string `emv:"5F20" json:"cardholderName"`
	ApplicationExpDate       []byte `emv:"5F24" json:"applicationExpirationDate"`
	IssuerAppData            []byte `emv:"9F10" json:"issuerApplicationData"`
	ATC                      []byte `emv:"9F36" json:"applicationTransactionCounter"`
	PinTryCounter            []byte `emv:"9F17" json:"pinTryCounter"`
	TransactionStatusInfo    []byte `emv:"9F6E" json:"transactionStatusInformation"`
	CardTransactionQualifier []byte `emv:"9F6C" json:"cardTransactionQualifier"`
	UnpredictableNumber      []byte `emv:"9F37" json:"unpredictableNumber"`
	ApplicationCryptogram    []byte `emv:"9F26" json:"applicationCryptogram"`
	IssuerAuthData           []byte `emv:"91" json:"issuerAuthenticationData"`
}

// EMVTagFormat defines the expected format for a specific EMV tag
type EMVTagFormat struct {
	// MinLength is the minimum length in bytes
	MinLength int

	// PadLeft indicates whether to pad on the left with zeros (true) or right (false)
	PadLeft bool

	// Format is an optional format spec used for special cases
	Format string
}

// EMVTagFormats maps EMV tags to their expected format
var EMVTagFormats = map[string]EMVTagFormat{
	// Application Interchange Profile
	"82": {MinLength: 2, PadLeft: true},

	// Track 2 Equivalent Data
	"57": {MinLength: 0, PadLeft: false},

	// Cardholder Name
	"5F20": {MinLength: 0, PadLeft: false},

	// Application Expiration Date
	"5F24": {MinLength: 3, PadLeft: true},

	// Issuer Application Data
	"9F10": {MinLength: 0, PadLeft: false},

	// Application Cryptogram
	"9F26": {MinLength: 8, PadLeft: true},

	// Cryptogram Information Data
	"9F27": {MinLength: 1, PadLeft: true},

	// Application Transaction Counter
	"9F36": {MinLength: 2, PadLeft: true},

	// Unpredictable Number
	"9F37": {MinLength: 4, PadLeft: true},

	// Card Transaction Qualifier
	"9F6C": {MinLength: 2, PadLeft: true},

	// Transaction Status Information
	"9F6E": {MinLength: 4, PadLeft: true},

	// Response Message Template
	"77": {MinLength: 0, PadLeft: false},

	// Default for other tags
	"DEFAULT": {MinLength: 0, PadLeft: false},
}

// Simplified EMV TLV parser for this test
func parseEMVData(data []byte) (*EMVData, error) {
	// Parse TLV data (simplified for test)
	tagValues := make(map[string][]byte)

	// Start with position 0
	pos := 0
	for pos < len(data) {
		// Check if we have at least 1 byte for tag
		if pos >= len(data) {
			break
		}

		// Determine tag length (1 or 2 bytes)
		tagLen := 1
		if (data[pos] & 0x1F) == 0x1F {
			tagLen = 2
			// Ensure we have enough bytes
			if pos+1 >= len(data) {
				return nil, fmt.Errorf("unexpected end of data when reading tag")
			}
		}

		// Extract tag
		tag := data[pos : pos+tagLen]
		pos += tagLen

		// Ensure we have at least 1 byte for length
		if pos >= len(data) {
			return nil, fmt.Errorf("unexpected end of data when reading length")
		}

		// Determine length bytes
		lenByte := data[pos]
		pos++

		var valueLen int
		if (lenByte & 0x80) != 0 {
			// Length is in next N bytes where N is (lenByte & 0x7F)
			lenBytes := int(lenByte & 0x7F)
			if pos+lenBytes > len(data) {
				return nil, fmt.Errorf("unexpected end of data when reading extended length")
			}

			// Calculate length from multiple bytes
			valueLen = 0
			for i := 0; i < lenBytes; i++ {
				valueLen = (valueLen << 8) | int(data[pos])
				pos++
			}
		} else {
			// Length is in this byte
			valueLen = int(lenByte)
		}

		// Ensure we have enough bytes for value
		if pos+valueLen > len(data) {
			return nil, fmt.Errorf("unexpected end of data when reading value")
		}

		// Extract value
		value := data[pos : pos+valueLen]
		pos += valueLen

		// Store in map
		tagHex := hex.EncodeToString(tag)
		tagHex = fmt.Sprintf("%X", tag) // Uppercase
		tagValues[tagHex] = value

		// Process constructed tags (recursively)
		if (tag[0] & 0x20) != 0 {
			// This is a constructed tag, parse its value
			subTags, err := parseConstructedValue(value)
			if err != nil {
				continue // Skip if error parsing constructed value
			}

			// Add sub-tags to main map
			for subTag, subValue := range subTags {
				tagValues[subTag] = subValue
			}
		}
	}

	// Create EMVData struct and populate
	result := &EMVData{}

	// Use reflection to set struct fields
	v := reflect.ValueOf(result).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		tag := t.Field(i).Tag.Get("emv")

		if tag == "" {
			continue
		}

		value, ok := tagValues[tag]
		if !ok {
			continue
		}

		// Set field value
		if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Uint8 {
			field.SetBytes(value)
		} else if field.Kind() == reflect.String {
			field.SetString(string(value))
		}
	}

	return result, nil
}

// Parse constructed TLV value
func parseConstructedValue(data []byte) (map[string][]byte, error) {
	result := make(map[string][]byte)

	// Simplified parsing logic for constructed tags
	pos := 0
	for pos < len(data) {
		// Similar to main parsing but simplified
		if pos+1 >= len(data) {
			break
		}

		// Determine tag length
		tagLen := 1
		if (data[pos] & 0x1F) == 0x1F {
			tagLen = 2
			if pos+1 >= len(data) {
				return result, nil
			}
		}

		tag := data[pos : pos+tagLen]
		pos += tagLen

		if pos >= len(data) {
			return result, nil
		}

		lenByte := data[pos]
		pos++

		var valueLen int
		if (lenByte & 0x80) != 0 {
			lenBytes := int(lenByte & 0x7F)
			if pos+lenBytes > len(data) {
				return result, nil
			}

			valueLen = 0
			for i := 0; i < lenBytes; i++ {
				valueLen = (valueLen << 8) | int(data[pos])
				pos++
			}
		} else {
			valueLen = int(lenByte)
		}

		if pos+valueLen > len(data) {
			return result, nil
		}

		value := data[pos : pos+valueLen]
		pos += valueLen

		tagHex := fmt.Sprintf("%X", tag) // Uppercase
		result[tagHex] = value
	}

	return result, nil
}

// Custom marshaler for EMV data
func marshalEMVData(data *EMVData) ([]byte, error) {
	// Start with empty result
	result := []byte{}

	// Use reflection to get struct fields
	v := reflect.ValueOf(data).Elem()
	t := v.Type()

	// Map to temporarily store tag-value pairs
	tlvMap := make(map[string][]byte)

	// First collect all non-empty fields
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		tag := t.Field(i).Tag.Get("emv")

		if tag == "" || isZeroValue(field) {
			continue
		}

		// Get value as bytes
		var value []byte
		if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Uint8 {
			value = field.Bytes()
		} else if field.Kind() == reflect.String {
			value = []byte(field.String())
		} else {
			continue
		}

		// Apply formatting based on tag
		value = formatValueForTag(value, tag)

		// Store in map
		tlvMap[tag] = value
	}

	// Special handling for template tag 77 (Response Message Template Format 1)
	if templateData, exists := tlvMap["77"]; exists {
		// Template should be the outer tag
		result = encodeTLV("77", templateData)
		return result, nil
	}

	// Build all other tags
	var innerTLVs []byte
	for tag, value := range tlvMap {
		if tag == "77" {
			continue // Skip template tag, handled separately
		}

		// Encode this TLV
		tlv := encodeTLV(tag, value)
		innerTLVs = append(innerTLVs, tlv...)
	}

	// For the GPO response, everything should be inside template 77
	if len(innerTLVs) > 0 {
		result = encodeTLV("77", innerTLVs)
	}

	return result, nil
}

// Format a value according to the EMV tag format
func formatValueForTag(value []byte, tag string) []byte {
	// Get format for this tag
	format, ok := EMVTagFormats[tag]
	if !ok {
		format = EMVTagFormats["DEFAULT"]
	}

	// If value is already longer than minimum length, return as is
	if len(value) >= format.MinLength {
		return value
	}

	// Apply padding
	paddedValue := make([]byte, format.MinLength)

	if format.PadLeft {
		// Pad on the left with zeros
		copy(paddedValue[format.MinLength-len(value):], value)
	} else {
		// Pad on the right with zeros
		copy(paddedValue, value)
	}

	return paddedValue
}

// Encode a single TLV
func encodeTLV(tag string, value []byte) []byte {
	// Decode tag from hex string
	tagBytes, _ := hex.DecodeString(tag)

	// Create TLV
	result := tagBytes

	// Encode length
	if len(value) < 128 {
		// Short form
		result = append(result, byte(len(value)))
	} else {
		// Long form
		// Determine how many bytes needed for length
		lenBytes := 0
		temp := len(value)
		for temp > 0 {
			lenBytes++
			temp >>= 8
		}

		// Add length byte with MSB set and number of length bytes
		result = append(result, byte(0x80|lenBytes))

		// Add length bytes
		for i := lenBytes - 1; i >= 0; i-- {
			result = append(result, byte((len(value)>>(i*8))&0xFF))
		}
	}

	// Add value
	result = append(result, value...)

	return result
}

// Check if a reflection value is zero
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Slice, reflect.Map, reflect.Interface, reflect.Ptr:
		return v.IsNil() || v.Len() == 0
	case reflect.String:
		return v.Len() == 0
	default:
		return v.IsZero()
	}
}

// Print EMV data in a readable format
func printEMVData(data *EMVData) {
	v := reflect.ValueOf(data).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := t.Field(i).Name
		tag := t.Field(i).Tag.Get("emv")

		if isZeroValue(field) {
			continue
		}

		fmt.Printf("Field: %s (Tag: %s)\n", fieldName, tag)
		if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Uint8 {
			fmt.Printf("  Value (hex): %X\n", field.Bytes())
		} else if field.Kind() == reflect.String {
			fmt.Printf("  Value: %s\n", field.String())
		}
	}
}

// Compare original and re-encoded EMV data
func compareEMVData(original, reEncoded []byte) {
	if len(original) != len(reEncoded) {
		fmt.Printf("Length mismatch: original=%d, re-encoded=%d\n", len(original), len(reEncoded))
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

// Extract TLVs from data
func extractTLVs(data []byte) map[string][]byte {
	result := make(map[string][]byte)
	pos := 0

	for pos < len(data) {
		// Check if we have enough data
		if pos >= len(data) {
			break
		}

		// Determine tag length
		tagLen := 1
		if (data[pos] & 0x1F) == 0x1F {
			tagLen = 2
			if pos+1 >= len(data) {
				break
			}
		}

		// Extract tag
		tag := fmt.Sprintf("%X", data[pos:pos+tagLen])
		pos += tagLen

		// Get length
		if pos >= len(data) {
			break
		}

		lenByte := data[pos]
		pos++

		var valueLen int
		if (lenByte & 0x80) != 0 {
			lenBytes := int(lenByte & 0x7F)
			if pos+lenBytes > len(data) {
				break
			}

			valueLen = 0
			for i := 0; i < lenBytes; i++ {
				valueLen = (valueLen << 8) | int(data[pos])
				pos++
			}
		} else {
			valueLen = int(lenByte)
		}

		// Extract value
		if pos+valueLen > len(data) {
			break
		}

		value := data[pos : pos+valueLen]
		pos += valueLen

		// Store in result
		result[tag] = value

		// If this is a constructed tag, also extract its inner TLVs
		if (data[pos-valueLen-tagLen-1] & 0x20) != 0 {
			innerTLVs := extractTLVs(value)
			for innerTag, innerValue := range innerTLVs {
				result[innerTag] = innerValue
			}
		}
	}

	return result
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
