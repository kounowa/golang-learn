package helper

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

/**
  将结构体转化为sql
*/
type SqlBuilder struct {
	source      interface{}
	tableName   string
	primaryKeys []string
	updatedKeys []string
	isUpdateAll bool
}

// 入参
func (sb *SqlBuilder) SetStruct(source interface{}) *SqlBuilder {
	sb.source = source
	return sb
}

func (sb *SqlBuilder) SetTable(table string) *SqlBuilder {
	sb.tableName = table
	return sb
}

// 设置主键（主要为了在更新的时候防止主键冲突)
func (sb *SqlBuilder) SetPrimaryKeys(keys ...string) *SqlBuilder {
	sb.primaryKeys = keys
	return sb
}

// 设置更新的字段
func (sb *SqlBuilder) SetUpdatedColumn(keys ...string) *SqlBuilder {
	sb.primaryKeys = keys
	sb.isUpdateAll = false
	return sb
}

// 是否更新所有
func (sb *SqlBuilder) SetIsUpdateAll(isUpdateAll bool) *SqlBuilder {
	sb.isUpdateAll = isUpdateAll
	return sb
}

// todo 检测包含参数是否完整
func (sb *SqlBuilder) isComplete() error {
	// 检测参数
	return nil
}

// build insert
// INSERT INTO `test` (`name`,`count`,`active`,`value`) VALUES( ? , ? , ? , ? )
func (sb *SqlBuilder) BuildInsert() (string, error) {
	if err := sb.isComplete(); err != nil {
		return "", err
	}
	return "", nil
}

// build update
// update {{table_name}} set
func (sb *SqlBuilder) BuildUpdate() (string, error) {
	if err := sb.isComplete(); err != nil {
		return "", err
	}
	return "", nil
}

// select
func (sb *SqlBuilder) BuildSelect() (string, error) {
	if err := sb.isComplete(); err != nil {
		return "", err
	}
	return "", nil
}

// build upsert
// INSERT INTO `test` (`name`,`count`,`active`,`value`)
// VALUES( ? , ? , ? , ? ) ON DUPLICATE KEY
// UPDATE  `count` = ? , `active` = `active` + 1, `value` = `value` - 1
func (sb *SqlBuilder) BuildUpsert() (string, error) {
	if err := sb.isComplete(); err != nil {
		return "", err
	}
	sql := "INSERT INTO " + sb.tableName + " ($$Columns$$) VALUES($$Values$$) " +
		"ON DUPLICATE KEY UPDATE $$Updates$$"
	columnList := ExplicitColumnAndValue(reflect.ValueOf(sb.source), reflect.TypeOf(sb.source))
	cols, vals, ups := sb.buildColumn(columnList)
	sql = strings.ReplaceAll(sql, "$$Columns$$", cols)
	sql = strings.ReplaceAll(sql, "$$Values$$", vals)
	sql = strings.ReplaceAll(sql, "$$Updates$$", ups)
	return sql, nil
}

func (sb *SqlBuilder) buildColumn(columnList []*ColumnValue) (string, string, string) {
	columns := ""
	values := ""
	updates := ""
	for _, columnValue := range columnList {
		var isPrimary bool
		for _, existsColumn := range sb.primaryKeys {
			if columnValue.ColumnName == existsColumn {
				isPrimary = true
			}
		}
		// 主键不参与到update中
		if isPrimary {
			columns = columns + columnValue.ColumnName + ","
			columnStrVal := buildValuesByReflect(columnValue.Value)
			values = values + columnStrVal + ","
			continue
		}
		if sb.isUpdateAll {
			columns = columns + columnValue.ColumnName + ","
			columnStrVal := buildValuesByReflect(columnValue.Value)
			values = values + columnStrVal + ","
			updates = updates + columnValue.ColumnName + "=" + columnStrVal + ","
		} else {
			var isExist bool
			for _, existsColumn := range sb.updatedKeys {
				if columnValue.ColumnName == existsColumn {
					isExist = true
				}
			}
			if isExist {
				columns = columns + columnValue.ColumnName + ","
				columnStrVal := buildValuesByReflect(columnValue.Value)
				values = values + columnStrVal + ","
				updates = updates + columnValue.ColumnName + "=" + columnStrVal + ","
			}

		}
	}
	return strings.TrimSuffix(columns, ","), strings.TrimSuffix(values, ","), strings.TrimSuffix(updates, ",")
}

// build
func buildValuesByReflect(value reflect.Value) string {
	switch value.Type().String() {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		ret := value.Int()
		return fmt.Sprint(ret)
	case "float32", "float64":
		ret := value.Float()
		return fmt.Sprint(ret)
	case "time.Time":
		nowTime := value.Interface().(time.Time)
		return "\"" + nowTime.Format("2006-01-02 15:04:05") + "\""
	case "string":
		v := value.String()
		v = strings.ReplaceAll(v, "\"", "\\\"")
		return "\"" + v + "\""
	}
	return ""
}
