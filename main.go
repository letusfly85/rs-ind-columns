package main

import (
	"database/sql"
	"flag"
	"github.com/joho/godotenv"
	"github.com/olekukonko/tablewriter"
	"log"
	"os"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

type IndexMeta struct {
	TableName string  `sql:"table_name"`
	IndexName string  `sql:"index_name"`
	IndexKey  []uint8 `sql:"ind_key"`
}

type IndexColumnMeta struct {
	SchemaName string `sql:"schema_name"`
	TableName  string `sql:"table_name"`
	IndexName  string `sql:"index_name"`
	AttrNumber int    `sql:"attr_number"`
	ColumnName string `sql:"column_name"`
}

var indexQuery = `
select
    t.relname as table_name,
    i.relname as index_name,
    ix.indkey
from
    pg_class t,
    pg_class i,
    pg_index ix
where
    t.oid = ix.indrelid
    and i.oid = ix.indexrelid
    and t.relkind = 'r'
    and t.relname like $1 
order by
    t.relname,
    i.relname
`

func getIndexColumnQuery(str string) string {
	var query = `
select
    ns.nspname as schema_name,
    t.relname as table_name,
    i.relname as index_name,
    a.attnum as attr_number,
    a.attname as column_name
from
    pg_class t,
    pg_class i,
    pg_index ix,
    pg_attribute a,
    pg_namespace ns
where
    t.oid = ix.indrelid
    and i.oid = ix.indexrelid
    and a.attrelid = t.oid
    and a.attnum in (` + str + `)
    and t.relkind = 'r'
    and t.relnamespace = ns.oid
    and t.relname like $1
order by
    a.attnum,
    t.relname,
    i.relname
`
	return query
}

func main() {
	var tableName string
	flag.StringVar(&tableName, "t", "blank", "tableName")
	flag.Parse()

	err := godotenv.Load("development.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	connStr := os.Getenv("DB_PARAM")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalln("failed connect:", err)
	}
	defer db.Close()

	row, err := db.Query(indexQuery, tableName+"%")
	if err != nil {
		log.Fatalln("failed query:", err)
	}
	indexMeta := IndexMeta{}
	for row.Next() {
		err := row.Scan(&indexMeta.TableName, &indexMeta.IndexName, &indexMeta.IndexKey)
		if err != nil {
			log.Fatalln("failed scan:", err)
		}
	}
	if err := row.Err(); err != nil {
		log.Fatalln("failed iterate:", err)
	}

	var aryInt []int
	indexKeys := strings.Split(string(indexMeta.IndexKey), " ")
	for i := 0; i < len(indexKeys); i++ {
		key, err := strconv.Atoi(indexKeys[i])
		if err != nil {
			log.Fatalln(err)
		}
		aryInt = append(aryInt, key)
	}

	var str string
	for _, value := range aryInt {
		str += strconv.Itoa(value) + ","
	}
	str = str[:len(str)-1]

	stmt, err := db.Prepare(getIndexColumnQuery(str))
	row, err = stmt.Query(tableName + "%")
	if err != nil {
		log.Fatalln("failed query:", err)
	}

	var indexColumnMetaList = []IndexColumnMeta{}
	for row.Next() {
		indexColumnMeta := IndexColumnMeta{}
		err := row.Scan(
			&indexColumnMeta.SchemaName,
			&indexColumnMeta.TableName,
			&indexColumnMeta.IndexName,
			&indexColumnMeta.AttrNumber,
			&indexColumnMeta.ColumnName)
		if err != nil {
			log.Fatalln("failed scan:", err)
		}
		indexColumnMetaList = append(indexColumnMetaList, indexColumnMeta)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"schema_name", "table_name", "index_name", "attrnum", "column_name"})
	for _, v := range indexColumnMetaList {
		strAry := []string{v.SchemaName, v.TableName, v.IndexName, strconv.Itoa(v.AttrNumber), v.ColumnName}
		table.Append(strAry)
	}
	table.Render()
	if err := row.Err(); err != nil {
		log.Fatalln("failed iterate:", err)
	}
}
