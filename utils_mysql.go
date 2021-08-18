package db2struct

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func GetTablesFromMysqlDatabase(mariadbUser string, mariadbPassword string, mariadbHost string, mariadbPort int,
	mariadbDatabase string) ([]*string, error) {

	var err error
	var db *sql.DB
	if mariadbPassword != "" {
		db, err = sql.Open("mysql", mariadbUser+":"+mariadbPassword+"@tcp("+mariadbHost+":"+strconv.Itoa(mariadbPort)+")/"+mariadbDatabase+"?&parseTime=True")
	} else {
		db, err = sql.Open("mysql", mariadbUser+"@tcp("+mariadbHost+":"+strconv.Itoa(mariadbPort)+")/"+mariadbDatabase+"?&parseTime=True")
	}
	defer db.Close()

	if err != nil {
		fmt.Println("Error opening mysql db: " + err.Error())
		return nil, err
	}

	tables := []*string{}
	sqlQuery := "SELECT table_name from INFORMATION_SCHEMA.tables where table_schema=?"

	// fmt.Printf("sql:%s,%+v\n",sqlQuery,mariadbDatabase)
	rows, err := db.Query(sqlQuery, mariadbDatabase)
	if err != nil {
		fmt.Println("Error selecting from db: " + err.Error())
		return nil, err
	}
	if rows != nil {
		defer rows.Close()
	} else {
		fmt.Println("no results returned for database:" + mariadbDatabase)
		return nil, errors.New("No results returned for table")
	}

	for rows.Next() {
		var table string
		rows.Scan(&table)
		tables = append(tables, &table)
	}

	return tables, nil
}

// GetColumnsFromMysqlTable Select column details from information schema and return map of map
func GetColumnsFromMysqlTable(mariadbUser string, mariadbPassword string, mariadbHost string, mariadbPort int,
	mariadbDatabase string, mariadbTable string) (*map[string]map[string]string, []string, error) {

	var err error
	var db *sql.DB
	if mariadbPassword != "" {
		db, err = sql.Open("mysql", mariadbUser+":"+mariadbPassword+"@tcp("+mariadbHost+":"+strconv.Itoa(mariadbPort)+")/"+mariadbDatabase+"?&parseTime=True")
	} else {
		db, err = sql.Open("mysql", mariadbUser+"@tcp("+mariadbHost+":"+strconv.Itoa(mariadbPort)+")/"+mariadbDatabase+"?&parseTime=True")
	}
	defer db.Close()

	// Check for error in db, note this does not check connectivity but does check uri
	if err != nil {
		fmt.Println("Error opening mysql db: " + err.Error())
		return nil, nil, err
	}

	columnNamesSorted := []string{}

	// Store colum as map of maps
	columnDataTypes := make(map[string]map[string]string)
	// Select columnd data from INFORMATION_SCHEMA
	columnDataTypeQuery := "SELECT COLUMN_NAME, COLUMN_KEY, DATA_TYPE, IS_NULLABLE, COLUMN_COMMENT FROM " +
		"INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = ? AND table_name = ? order by ordinal_position asc"

	if Debug {
		fmt.Println("running: " + columnDataTypeQuery)
	}

	rows, err := db.Query(columnDataTypeQuery, mariadbDatabase, mariadbTable)

	if err != nil {
		fmt.Println("Error selecting from db: " + err.Error())
		return nil, nil, err
	}
	if rows != nil {
		defer rows.Close()
	} else {
		return nil, nil, errors.New("No results returned for table")
	}

	for rows.Next() {
		var column string
		var columnKey string
		var dataType string
		var nullable string
		var comment string
		rows.Scan(&column, &columnKey, &dataType, &nullable, &comment)

		columnDataTypes[column] = map[string]string{"value": dataType, "nullable": nullable, "primary": columnKey, "comment": comment}
		columnNamesSorted = append(columnNamesSorted, column)
	}

	return &columnDataTypes, columnNamesSorted, err
}

// Generate go struct entries for a map[string]interface{} structure
func generateMysqlTypes(obj map[string]map[string]string, columnsSorted []string, depth int, jsonAnnotation bool,
	gormAnnotation bool, gureguTypes bool, tagAnnotation string, sqlType bool) string {
	structure := "struct {"

	for _, key := range columnsSorted {
		mysqlType := obj[key]
		nullable := false
		if sqlType && mysqlType["nullable"] == "YES" {
			nullable = true
		}

		primary := ""
		if mysqlType["primary"] == "PRI" {
			primary = ";primary_key"
		}

		// Get the corresponding go value type for this mysql type
		var valueType string
		// If the guregu (https://github.com/guregu/null) CLI option is passed use its types, otherwise use go's sql.NullX

		valueType = mysqlTypeToGoType(mysqlType["value"], nullable, gureguTypes)

		fieldName := fmtFieldName(stringifyFirstChar(key))
		var annotations []string
		if gormAnnotation == true {
			annotations = append(annotations, fmt.Sprintf("gorm:\"column:%s%s\"", key, primary))
		}
		if jsonAnnotation == true {
			annotations = append(annotations, fmt.Sprintf("json:\"%s\"", key))
		}

		if tagAnnotation != "" {
			annotations = append(annotations, fmt.Sprintf("%s:\"%s\"", tagAnnotation, key))
		}

		if len(annotations) > 0 {
			// add colulmn comment
			comment := mysqlType["comment"]
			structure += fmt.Sprintf("\n%s %s `%s`  //%s", fieldName, valueType, strings.Join(annotations, " "), comment)
			// structure += fmt.Sprintf("\n%s %s `%s`", fieldName, valueType, strings.Join(annotations, " "))
		} else {
			structure += fmt.Sprintf("\n%s %s", fieldName, valueType)
		}
	}
	return structure
}

// mysqlTypeToGoType converts the mysql types to go compatible sql.Nullable (https://golang.org/pkg/database/sql/) types
func mysqlTypeToGoType(mysqlType string, nullable bool, gureguTypes bool) string {
	switch mysqlType {
	case "tinyint", "int", "smallint", "mediumint":
		if nullable {
			if gureguTypes {
				return gureguNullInt
			}
			return sqlNullInt
		}
		return golangInt
	case "bigint":
		if nullable {
			if gureguTypes {
				return gureguNullInt
			}
			return sqlNullInt
		}
		return golangInt64
	case "char", "enum", "varchar", "longtext", "mediumtext", "text", "tinytext", "json":
		if nullable {
			if gureguTypes {
				return gureguNullString
			}
			return sqlNullString
		}
		return "string"
	case "date", "datetime", "time", "timestamp":
		if nullable && gureguTypes {
			return gureguNullTime
		}
		return golangTime
	case "decimal", "double":
		if nullable {
			if gureguTypes {
				return gureguNullFloat
			}
			return sqlNullFloat
		}
		return golangFloat64
	case "float":
		if nullable {
			if gureguTypes {
				return gureguNullFloat
			}
			return sqlNullFloat
		}
		return golangFloat32
	case "binary", "blob", "longblob", "mediumblob", "varbinary":
		return golangByteArray
	}
	return ""
}
