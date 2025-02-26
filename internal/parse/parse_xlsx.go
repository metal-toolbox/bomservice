package parse

import (
	"github.com/pkg/errors"
	"github.com/tealeg/xlsx/v3"

	fleetdbapi "github.com/metal-toolbox/fleetdb/pkg/api/v1"
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
	//nolint:gosec // it's not a credential!
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
//
//nolint:gocyclo,revive // this is inherently cyclomatic and yes, the name stutters
func ParseXlsxFile(fileBytes []byte) ([]fleetdbapi.Bom, error) {
	file, err := xlsx.OpenBinary(fileBytes)
	if err != nil {
		return nil, errors.New("failed to open the file")
	}

	bomsMap := make(map[string]*fleetdbapi.Bom)

	for _, sheet := range file.Sheets {
		var categoryCol *categoryColNum

		rowProcessor := func(row *xlsx.Row) error {
			if categoryCol == nil {
				categoryCol = newCategoryColNum()

				cellProcessor := func(cell *xlsx.Cell) error {
					i, _ := cell.GetCoordinates()
					switch cell.Value {
					case serialNumColName:
						categoryCol.serialNumCol = i
					case subItemColName:
						categoryCol.subItemCol = i
					case subSerialColName:
						categoryCol.subSerialCol = i
					}
					return nil
				}
				_ = row.ForEachCell(cellProcessor)

				if categoryCol.serialNumCol == -1 || categoryCol.subItemCol == -1 || categoryCol.subSerialCol == -1 {
					return errors.Errorf("missing colomn, serial num %v, sub-item %v, sub-serial %v", categoryCol.serialNumCol, categoryCol.subItemCol, categoryCol.subSerialCol)
				}

				return nil
			}

			// There won't be any out of idex issue since any non-existing value will default to empty string.
			serialNum := row.GetCell(categoryCol.serialNumCol).Value

			if serialNum == "" {
				return errors.New("empty serial number")
			}

			bom, ok := bomsMap[serialNum]
			if !ok {
				bom = &fleetdbapi.Bom{SerialNum: serialNum}
				bomsMap[serialNum] = bom
			}

			v := row.GetCell(categoryCol.subSerialCol).Value
			switch row.GetCell(categoryCol.subItemCol).Value {
			case aocFieldName:
				aocMacAddress := v
				if aocMacAddress == "" {
					return errors.New("empty aoc mac address")
				}

				if bom.AocMacAddress != "" {
					bom.AocMacAddress += ","
				}

				bom.AocMacAddress += aocMacAddress
			case bmcFieldName:
				bmcMacAddress := v
				if bmcMacAddress == "" {
					return errors.New("empty bmc mac address")
				}

				if bom.BmcMacAddress != "" {
					bom.BmcMacAddress += ","
				}

				bom.BmcMacAddress += bmcMacAddress
			case ipmiFieldName:
				bom.NumDefiPmi = v
			case ipwdFieldName:
				bom.NumDefPWD = v
			}
			return nil
		}
		err := sheet.ForEachRow(rowProcessor)
		if err != nil {
			return nil, err
		}
	}

	retBoms := make([]fleetdbapi.Bom, 0, len(bomsMap))
	for _, bom := range bomsMap {
		retBoms = append(retBoms, *bom)
	}
	return retBoms, nil
}
