package parse

import (
	"bufio"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	sservice "go.hollow.sh/serverservice/pkg/api/v1"
)

var testSerialNumBomInfo1 = sservice.Bom{
	SerialNum:     "test-serial-1",
	AocMacAddress: "FakeAOC1,FakeAOC2",
	BmcMacAddress: "FakeMac1,FakeMac2",
	NumDefiPmi:    "FakeDEFI1",
	NumDefPWD:     "FakeDEFPWD1",
}

var testSerialNumBomInfo2 = sservice.Bom{
	SerialNum:     "test-serial-2",
	AocMacAddress: "FakeAOC3,FakeAOC4",
	BmcMacAddress: "FakeMac3,FakeMac4",
	NumDefiPmi:    "FakeDEFI2",
	NumDefPWD:     "FakeDEFPWD2",
}

func sortSerialNumBomInfos(boms []sservice.Bom) {
	sort.Slice(boms, func(i, j int) bool {
		return boms[i].SerialNum < boms[j].SerialNum
	})
}

func TestParseXlsxFile(t *testing.T) {
	var testCases = []struct {
		testName                 string
		filePath                 string
		expectedErr              bool
		expectedErrMsg           string
		expectedSerialNumBomInfo []sservice.Bom
	}{
		{
			testName:                 "file missing serial number",
			filePath:                 "./testdata/test_empty_serial.xlsx",
			expectedErr:              true,
			expectedErrMsg:           "empty serial number",
			expectedSerialNumBomInfo: nil,
		},
		{
			testName:                 "file missing aocMacAddress ",
			filePath:                 "./testdata/test_empty_aocMacAddress.xlsx",
			expectedErr:              true,
			expectedErrMsg:           "empty aoc mac address",
			expectedSerialNumBomInfo: nil,
		},
		{
			testName:                 "file missing bmcMacAddress",
			filePath:                 "./testdata/test_empty_bmcMacAddress.xlsx",
			expectedErr:              true,
			expectedErrMsg:           "empty bmc mac address",
			expectedSerialNumBomInfo: nil,
		},
		{
			testName:                 "valid file for single bom",
			filePath:                 "./testdata/test_valid_one_bom.xlsx",
			expectedErr:              false,
			expectedErrMsg:           "",
			expectedSerialNumBomInfo: []sservice.Bom{testSerialNumBomInfo1},
		},
		{
			testName:                 "valid file for multiple bom",
			filePath:                 "./testdata/test_valid_multiple_boms.xlsx",
			expectedErr:              false,
			expectedErrMsg:           "",
			expectedSerialNumBomInfo: []sservice.Bom{testSerialNumBomInfo1, testSerialNumBomInfo2},
		},
		{
			testName:                 "file missing SERIALNUM col",
			filePath:                 "./testdata/test_empty_serial_col.xlsx",
			expectedErr:              true,
			expectedErrMsg:           "missing colomn, serial num -1, sub-item 4, sub-serial 5",
			expectedSerialNumBomInfo: nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.testName, func(t *testing.T) {
			file, err := os.Open(tt.filePath)
			if err != nil {
				t.Fatalf("os.Open(%v) failed to open file %v\n", tt.filePath, err)
			}

			// Translate file to bytes since ParseXlsxFile accepts bytes of file as argument,
			// which is the format of a file reading from the HTTP request.
			stat, err := file.Stat()
			if err != nil {
				t.Fatalf("file.Stat() failed %v", err)
			}
			bs := make([]byte, stat.Size())
			_, err = bufio.NewReader(file).Read(bs)
			if err != nil && err != io.EOF {
				t.Fatalf("bufio.NewReader(file).Read(bs) failed %v", err)
			}

			serialNumInfos, err := ParseXlsxFile(bs)
			if tt.expectedErr {
				if !strings.Contains(err.Error(), tt.expectedErrMsg) {
					t.Fatalf("test %v failed, got %v, expect %v", tt.testName, err, tt.expectedErrMsg)
				}
				if serialNumInfos != nil {
					t.Fatalf("test %v expect nil serialNumInfos, got %v", tt.testName, len(serialNumInfos))
				}
				return
			}
			if err != nil {
				t.Fatalf("test %v failed to parse Xlsx file: %v", tt.testName, err)
			}

			if len(serialNumInfos) != len(tt.expectedSerialNumBomInfo) {
				t.Fatalf("test %v parsed incorrect numbers of serialNumInfos, got %v, expect %v", tt.testName, len(serialNumInfos), tt.expectedSerialNumBomInfo)
			}

			// Sort the serialNumInfos to avoid unexpected orders of the maps.Values
			sortSerialNumBomInfos(serialNumInfos)
			sortSerialNumBomInfos(tt.expectedSerialNumBomInfo)
			for i := range serialNumInfos {
				info := serialNumInfos[i]
				expectedInfo := tt.expectedSerialNumBomInfo[i]
				if !reflect.DeepEqual(info, expectedInfo) {
					t.Fatalf("test %v parsed incorrect bom info, got %v, expect %v", tt.testName, info, expectedInfo)
				}
			}
		})
	}
}
