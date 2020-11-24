package util

import (
	"fmt"

	"github.com/giorgisio/goav/avutil"
)

// GetAVDictionaryFromMap convert map to avutil.Dictionary
func GetAVDictionaryFromMap(m map[string]interface{}) (d *avutil.Dictionary, err error) {
	for k, v := range m {
		switch v.(type) {
		case int64:
			d.AvDictSetInt(k, v.(int64), 0)
		case string:
			d.AvDictSet(k, v.(string), 0)
		default:
			err = fmt.Errorf("GetAVDictionaryFromMap: unsupported type(%t)", v)
			d.AvDictFree()
			return
		}
	}
	return
}
