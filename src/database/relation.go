package database

const QUERY_NAME = "relations"

type Relation struct {
	ConstraintName    string `db:"constraint_name"`
	TableName         string `db:"table_name"`
	ColumnName        string `db:"column_name"`
	ForeignTableName  string `db:"foreign_table_name"`
	ForeignColumnName string `db:"foreign_column_name"`
}

func GetRelations(table string) ([]Relation, error) {
	var rels []Relation
	err := QuerySelect(QUERY_NAME, &rels, table)
	return rels, err
}

func LoadRelations(model Model) error {

	return nil
}
