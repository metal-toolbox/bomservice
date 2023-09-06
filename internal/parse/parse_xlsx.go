package parse

import (
	"github.com/pkg/errors"
	"github.com/tealeg/xlsx"

	sservice "go.hollow.sh/serverservice/pkg/api/v1"
)

const (
	// column name
	serialNumColName string = "SERIALNUM"
	subItemColName   string = "SUB-ITEM"
	subSerialColName string = "SUB-SERIAL" // value of the sub-item
	// interested catogories in sub-item
	aocFieldName  string = "MAC-AOC-ADDRESS"
	bmcFieldName  string = "MAC-ADDRESS"
	ipmiFieldName string = "NUM-DEFIPMI"
	ipwdFieldName string = "NUM-DEFPWD"
)

type categoryColNum struct {
	serialNumCol int // column number of the serial number, -1 means no such column
	subItemCol   int // column number of the sub-item, -1 means no such column
	subSerialCol int // column number of the sub-serial, -1 means no such column
}

func newCategoryColNum() *categoryColNum {
	return &categoryColNum{
		serialNumCol: -1,
		subItemCol:   -1,
		subSerialCol: -1,
	}
}

// ParseXlsxFile is the helper function to parse xlsx to boms.
func ParseXlsxFile(fileBytes []byte) ([]sservice.Bom, error) {
	file, err := xlsx.OpenBinary(fileBytes)
	if err != nil {
		return nil, errors.New("failed to open the file")
	}

	bomsMap := make(map[string]*sservice.Bom)

	for _, sheet := range file.Sheets {
		var categoryCol *categoryColNum
		for _, row := range sheet.Rows {
			if categoryCol == nil {
				categoryCol = newCategoryColNum()

				for i, cell := range row.Cells {
					switch cell.Value {
					case serialNumColName:
						categoryCol.serialNumCol = i
					case subItemColName:
						categoryCol.subItemCol = i
					case subSerialColName:
						categoryCol.subSerialCol = i
					}
				}

				if categoryCol.serialNumCol == -1 || categoryCol.subItemCol == -1 || categoryCol.subSerialCol == -1 {
					return nil, errors.Errorf("missing colomn, serial num %v, sub-item %v, sub-serial %v", categoryCol.serialNumCol, categoryCol.subItemCol, categoryCol.subSerialCol)
				}

				continue
			}

			// There won't be any out of idex issue since any non-existing value will default to empty string.
			cells := row.Cells
			serialNum := cells[categoryCol.serialNumCol].Value

			if len(serialNum) == 0 {
				return nil, errors.New("empty serial number")
			}

			bom, ok := bomsMap[serialNum]
			if !ok {
				bom = &sservice.Bom{SerialNum: serialNum}
				bomsMap[serialNum] = bom
			}

			switch cells[categoryCol.subItemCol].Value {
			case aocFieldName:
				aocMacAddress := cells[categoryCol.subSerialCol].Value
				if len(aocMacAddress) == 0 {
					return nil, errors.New("empty aoc mac address")
				}

				if len(bom.AocMacAddress) > 0 {
					bom.AocMacAddress += ","
				}

				bom.AocMacAddress += aocMacAddress
			case bmcFieldName:
				bmcMacAddress := cells[categoryCol.subSerialCol].Value
				if len(bmcMacAddress) == 0 {
					return nil, errors.New("empty bmc mac address")
				}

				if len(bom.BmcMacAddress) > 0 {
					bom.BmcMacAddress += ","
				}

				bom.BmcMacAddress += bmcMacAddress
			case ipmiFieldName:
				bom.NumDefiPmi = cells[categoryCol.subSerialCol].Value
			case ipwdFieldName:
				bom.NumDefPWD = cells[categoryCol.subSerialCol].Value
			}
		}
	}

	var retBoms []sservice.Bom
	for _, bom := range bomsMap {
		retBoms = append(retBoms, *bom)
	}
	return retBoms, nil
}
