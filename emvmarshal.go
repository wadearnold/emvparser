package kernel

import (
	"encoding/hex"
	"fmt"
	"log"
	"reflect"
)

// EMVData represents a parsed EMV record with fields mapped to EMV tags
type EMVData struct {
	ResponseMessageTemplate     []byte `emv:"77" json:"responseMessageTemplate1"`
	AIP                         []byte `emv:"82" json:"applicationInterchangeProfile"`
	TrackData                   []byte `emv:"57" json:"track2EquivalentData"`
	CardholderName              string `emv:"5F20" json:"cardholderName"`
	ApplicationExpDate          []byte `emv:"5F24" json:"applicationExpirationDate"`
	IssuerAppData               []byte `emv:"9F10" json:"issuerApplicationData"`
	ATC                         []byte `emv:"9F36" json:"applicationTransactionCounter"`
	PinTryCounter               []byte `emv:"9F17" json:"pinTryCounter"`
	TransactionStatusInfo       []byte `emv:"9F6E" json:"transactionStatusInformation"`
	CardTransactionQualifier    []byte `emv:"9F6C" json:"cardTransactionQualifier"`
	UnpredictableNumber         []byte `emv:"9F37" json:"unpredictableNumber"`
	ApplicationCryptogram       []byte `emv:"9F26" json:"applicationCryptogram"`
	IssuerAuthData              []byte `emv:"91" json:"issuerAuthenticationData"`
	PanSequenceNumber           []byte `emv:"5F34" json:"panSequenceNumber"`
	CryptogramInformationData   []byte `emv:"9F47" json:"cryptogramInformationData"`
	IntegredCircuitLevelResults []byte `emv:"9F27" json:"integratedCircuitLevelResults"`
}

// EMVTagFormat defines the expected format for a specific EMV tag
type EMVTagFormat struct {
	// MinLength is the minimum length in bytes
	MinLength int

	// MaxLength is the maximum length in bytes (0 means no maximum)
	MaxLength int

	// PadLeft indicates whether to pad on the left with zeros (true) or right (false)
	PadLeft bool

	// Format is an optional format spec used for special cases
	Format string

	// Description provides a human-readable description of the tag
	Description string
}

// EMVTagFormats maps EMV tags to their expected format
var EMVTagFormats = map[string]EMVTagFormat{
	"82":      {MinLength: 2, MaxLength: 2, PadLeft: true, Description: "Application Interchange Profile"},
	"57":      {MinLength: 0, MaxLength: 37, PadLeft: false, Description: "Track 2 Equivalent Data"},
	"5F20":    {MinLength: 0, MaxLength: 26, PadLeft: false, Description: "Cardholder Name"},
	"5F24":    {MinLength: 3, MaxLength: 3, PadLeft: true, Description: "Application Expiration Date"},
	"9F10":    {MinLength: 0, MaxLength: 32, PadLeft: false, Description: "Issuer Application Data"},
	"9F26":    {MinLength: 8, MaxLength: 8, PadLeft: true, Description: "Application Cryptogram"},
	"9F36":    {MinLength: 2, MaxLength: 2, PadLeft: true, Description: "Application Transaction Counter"},
	"9F37":    {MinLength: 4, MaxLength: 4, PadLeft: true, Description: "Unpredictable Number"},
	"9F6C":    {MinLength: 2, MaxLength: 8, PadLeft: true, Description: "Card Transaction Qualifier"},
	"9F6E":    {MinLength: 4, MaxLength: 4, PadLeft: true, Description: "Transaction Status Information"},
	"77":      {MinLength: 0, MaxLength: 0, PadLeft: false, Description: "Response Message Template"},
	"5F34":    {MinLength: 1, MaxLength: 1, PadLeft: true, Description: "PAN Sequence Number"},
	"9F47":    {MinLength: 1, MaxLength: 1, PadLeft: false, Description: "Cryptogram Information Data (CID)"},
	"9F27":    {MinLength: 1, MaxLength: 1, PadLeft: true, Description: "Integrated Circuit Level Results"},
	"DEFAULT": {MinLength: 0, MaxLength: 0, PadLeft: false, Description: "Default Tag Format"},
}

// EMVTagMap provides a mapping from EMV tag to struct field
type EMVTagMap map[string]fieldInfo

type fieldInfo struct {
	Index int
	Field reflect.StructField
}

// BuildEMVTagMap creates a mapping from EMV tags to struct fields
func BuildEMVTagMap(structType reflect.Type) EMVTagMap {
	tagMap := make(EMVTagMap)

	for i := range structType.NumField() {
		field := structType.Field(i)

		// Get the emv tag value from the struct tag
		tagValue := field.Tag.Get("emv")
		if tagValue != "" {
			// Store field info in the map with the EMV tag as key
			tagMap[tagValue] = fieldInfo{
				Index: i,
				Field: field,
			}
		}
	}

	return tagMap
}

// EMVParser handles parsing and mapping of EMV data
type EMVParser struct {
	tagMap EMVTagMap
}

// NewEMVParser creates a new EMV parser for the given struct type
func NewEMVParser() *EMVParser {
	// Build tag map from the EMVData struct
	tagMap := BuildEMVTagMap(reflect.TypeOf(EMVData{}))

	return &EMVParser{
		tagMap: tagMap,
	}
}

// Parse EMV data using the parser
func (parser *EMVParser) Parse(data []byte) (*EMVData, error) {
	tagValues := make(map[string][]byte)

	// Start parsing at position 0
	pos := 0
	for pos < len(data) {
		// Ensure we have at least 1 byte for the tag
		if pos >= len(data) {
			break
		}

		// Determine tag length (1 or 2 bytes)
		tagLen := 1
		if (data[pos] & 0x1F) == 0x1F {
			tagLen = 2
			// Ensure we have enough bytes for a 2-byte tag
			if pos+1 >= len(data) {
				return nil, fmt.Errorf("unexpected end of data when reading tag")
			}
		}

		// Extract the tag
		tag := data[pos : pos+tagLen]
		pos += tagLen

		// Ensure we have at least 1 byte for the length
		if pos >= len(data) {
			return nil, fmt.Errorf("unexpected end of data when reading length")
		}

		// Determine the length of the value
		lenByte := data[pos]
		pos++

		valueLen := 0
		if (lenByte & 0x80) != 0 {
			// Length is in the next N bytes where N is (lenByte & 0x7F)
			lenBytes := int(lenByte & 0x7F)
			if pos+lenBytes > len(data) {
				return nil, fmt.Errorf("unexpected end of data when reading extended length")
			}

			// Calculate length from multiple bytes
			for i := 0; i < lenBytes; i++ {
				valueLen = (valueLen << 8) | int(data[pos])
				pos++
			}
		} else {
			// Length is in this byte
			valueLen = int(lenByte)
		}

		// Ensure we have enough bytes for the value
		if pos+valueLen > len(data) {
			return nil, fmt.Errorf("unexpected end of data when reading value")
		}

		// Extract the value
		value := data[pos : pos+valueLen]
		pos += valueLen

		// Check if the tag is a constructed tag (6th bit of the first byte is set)
		if (tag[0] & 0x20) != 0 {
			// This is a constructed tag, recursively parse its value
			subTags, err := parser.Parse(value)
			if err != nil {
				return nil, fmt.Errorf("error parsing constructed tag %X: %v", tag, err)
			}

			// Add sub-tags to the main map
			for subTag, subValue := range subTags.toMap() {
				tagValues[subTag] = subValue
			}
		} else {
			// Store the tag and value in the map
			tagHex := fmt.Sprintf("%X", tag) // Convert tag to uppercase hex string
			tagValues[tagHex] = value
		}
	}

	// Create an EMVData struct and populate its fields
	result := &EMVData{}
	v := reflect.ValueOf(result).Elem()

	for tag, value := range tagValues {
		fieldInfo, ok := parser.tagMap[tag]
		if !ok {
			// Log unknown tag
			log.Fatalf("Warning: Tag %s found in data but not defined in EMVData\n", tag)
			continue // Skip unknown tags
		}

		field := v.Field(fieldInfo.Index)
		if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Uint8 {
			field.SetBytes(value)
		} else if field.Kind() == reflect.String {
			field.SetString(string(value))
		}
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

// extractTLVs parses a byte slice containing TLV (Tag-Length-Value) encoded data
// and extracts the individual TLVs into a map. The map's keys are the tags (as
// uppercase hex strings), and the values are the corresponding byte slices for
// each tag's value.
//
// This function supports both primitive and constructed tags. For constructed
// tags (indicated by the 6th bit of the first byte of the tag being set), it
// recursively parses the inner TLVs and adds them to the map.
//
// Parameters:
// - data: A byte slice containing the TLV-encoded data.
//
// Returns:
//   - A map where the keys are tags (as strings) and the values are the corresponding
//     byte slices for each tag's value.
//   - If the data is malformed or incomplete, the function stops parsing and returns
//     the map with the successfully parsed TLVs up to that point.
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

// Helper method to convert EMVData to a map for nested tag handling
func (data *EMVData) toMap() map[string][]byte {
	result := make(map[string][]byte)
	v := reflect.ValueOf(data).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		tag := t.Field(i).Tag.Get("emv")

		if isZeroValue(field) {
			continue
		}

		if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Uint8 {
			result[tag] = field.Bytes()
		} else if field.Kind() == reflect.String {
			result[tag] = []byte(field.String())
		}
	}

	return result
}

// Marshal EMV data using the parser
func (parser *EMVParser) Marshal(data *EMVData) ([]byte, error) {
	result := []byte{}
	v := reflect.ValueOf(data).Elem()

	// Map to temporarily store tag-value pairs
	tlvMap := make(map[string][]byte)

	// Collect all non-empty fields
	for tag, fieldInfo := range parser.tagMap {
		field := v.Field(fieldInfo.Index)
		if isZeroValue(field) {
			continue
		}

		var value []byte
		if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Uint8 {
			value = field.Bytes()
		} else if field.Kind() == reflect.String {
			value = []byte(field.String())
		}

		// Apply formatting
		value = formatValueForTag(value, tag)

		// Store in map
		tlvMap[tag] = value
	}

	// Special handling for template tag 77 (Response Message Template Format 1)
	if _, exists := tlvMap["77"]; exists {
		// Template should be the outer tag
		innerTLVs := []byte{}

		// Build nested TLVs for all other tags
		for tag, value := range tlvMap {
			if tag == "77" {
				continue // Skip the outer tag itself
			}

			// Encode this TLV
			tlv := encodeTLV(tag, value)
			innerTLVs = append(innerTLVs, tlv...)
		}

		// Combine nested TLVs into the outer tag
		result = encodeTLV("77", innerTLVs)
		return result, nil
	}

	// If no template tag 77, build all tags normally
	for tag, value := range tlvMap {
		tlv := encodeTLV(tag, value)
		result = append(result, tlv...)
	}

	return result, nil
}
