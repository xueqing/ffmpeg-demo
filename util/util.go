package util

import (
	"fmt"

	"github.com/xueqing/goav/libavutil"
)

// GetAVDictionaryFromMap convert map to libavutil.Dictionary
func GetAVDictionaryFromMap(m map[string]interface{}) (d *libavutil.AvDictionary, err error) {
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
