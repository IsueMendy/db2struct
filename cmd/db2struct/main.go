package main

import (
	"fmt"
	"github.com/Shelnutt2/db2struct"
	goopt "github.com/droundy/goopt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/howeyc/gopass"
	"os"
	"strconv"
)

var mariadbHost = os.Getenv("MYSQL_HOST")
var mariadbHostPassed = goopt.String([]string{"-H", "--host"}, "", "Host to check mariadb status of")
var mariadbPort = goopt.Int([]string{"--mysql_port"}, 3306, "Specify a port to connect to")
var mariadbTable = goopt.String([]string{"-t", "--table"}, "", "Table to build struct from, if nil, "+
	"all tables from databases")
var mariadbDatabase = goopt.String([]string{"-d", "--database"}, "nil", "Database to for connection")
var mariadbPassword *string
var mariadbUser = goopt.String([]string{"-u", "--user"}, "user", "user to connect to database")
var verbose = goopt.Flag([]string{"-v", "--verbose"}, []string{}, "Enable verbose output", "")
var packageName = goopt.String([]string{"--package"}, "", "name to set for package")
var structName = goopt.String([]string{"--struct"}, "", "name to set for struct")
var tagAnnotation = goopt.String([]string{"--tag"}, "", "inject tag for struct")

var sqlType = goopt.Flag([]string{"--sqlNULL"}, []string{}, "use sql.NULL type", "use raw golang type")
var jsonAnnotation = goopt.Flag([]string{"--json"}, []string{"--no-json"}, "Add json annotations (default)", "Disable json annotations")
var gormAnnotation = goopt.Flag([]string{"--gorm"}, []string{}, "Add gorm annotations (tags)", "")
var gureguTypes = goopt.Flag([]string{"--guregu"}, []string{}, "Add guregu null types", "")
var targetFile = goopt.String([]string{"--target"}, "", "Save file path")

func init() {
	goopt.OptArg([]string{"-p", "--password"}, "", "Mysql password", getMariadbPassword)
	// goopt.ReqArg([]string{"-u", "--user"}, "user", "user to connect to database", setUser)

	// Setup goopts
	goopt.Description = func() string {
		return "Mariadb http Check"
	}
	goopt.Version = "0.0.2"
	goopt.Summary = "db2struct [-H] [-p] [-v] --package pkgName --struct structName --database databaseName --table tableName"

	// Parse options
	goopt.Parse(nil)

}

func main() {

	// Username is required
	if mariadbUser == nil || *mariadbUser == "user" {
		fmt.Println("Username is required! Add it with --user=name")
		return
	}

	// If a mariadb host is passed use it
	if mariadbHostPassed != nil && *mariadbHostPassed != "" {
		mariadbHost = *mariadbHostPassed
	}

	if mariadbPassword != nil && *mariadbPassword == "" {
		fmt.Print("Password: ")
		pass, err := gopass.GetPasswd()
		stringPass := string(pass)
		mariadbPassword = &stringPass
		if err != nil {
			fmt.Println("Error reading password: " + err.Error())
			return
		}
	} else if mariadbPassword == nil {
		p := ""
		mariadbPassword = &p
	}

	if *verbose {
		fmt.Println("Connecting to mysql server " + mariadbHost + ":" + strconv.Itoa(*mariadbPort))
	}

	if mariadbDatabase == nil || *mariadbDatabase == "" {
		fmt.Println("Database can not be null")
		return
	}

	if mariadbTable == nil || *mariadbTable == "" {
		if targetFile != nil && *targetFile != "" {
			os.MkdirAll(*targetFile, os.ModePerm)
		}
		tables, _ := db2struct.GetTablesFromMysqlDatabase(*mariadbUser, *mariadbPassword,
			mariadbHost, *mariadbPort, *mariadbDatabase)
		// fmt.Printf("get tables from db %+v, tables:%+v\n",*mariadbDatabase,tables)
		for _, dbTable := range tables {
			fmt.Printf("genStruct from table:%+v\n", *dbTable)
			var outputFile string
			if targetFile != nil && *targetFile != "" {
				outputFile = *targetFile + "/" + *dbTable + ".go"
			}
			genStructsByTables(dbTable, *structName, *packageName, outputFile)
		}
		return
	}

	genStructsByTables(mariadbTable, *structName, *packageName, *targetFile)
}

func genStructsByTables(dbTable *string, localStruct string, localPackage string, localTarget string) {
	columnDataTypes, columnsSorted, err := db2struct.GetColumnsFromMysqlTable(*mariadbUser, *mariadbPassword,
		mariadbHost, *mariadbPort, *mariadbDatabase, *dbTable)

	if err != nil {
		fmt.Println("Error in selecting column data information from mysql information schema")
		return
	}

	// If structName is not set we need to default it
	if localStruct == "" {
		localStruct = *dbTable
		// *structName = "newStruct"
	}
	// If packageName is not set we need to default it
	if localPackage == "" {
		localPackage = *mariadbDatabase
	}
	// Generate struct string based on columnDataTypes
	struc, err := db2struct.Generate(*columnDataTypes, columnsSorted, *dbTable, localStruct, localPackage,
		*jsonAnnotation, *gormAnnotation, *gureguTypes, *tagAnnotation, *sqlType)

	if err != nil {
		fmt.Println("Error in creating struct from json: " + err.Error())
		return
	}
	if localTarget != "" {
		file, err := os.OpenFile(localTarget, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("Open File fail: " + err.Error())
			return
		}
		length, err := file.WriteString(string(struc))
		if err != nil {
			fmt.Println("Save File fail: " + err.Error())
			return
		}
		fmt.Printf("wrote %d bytes\n", length)
	} else {
		fmt.Println(string(struc))
	}
}

func getMariadbPassword(password string) error {
	mariadbPassword = new(string)
	*mariadbPassword = password
	return nil
}
