package cache

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"
)

const defaultShardCount = 16

// FnvHash 对泛型 key 计算 FNV-1a 哈希值，作为分片路由的默认实现。
func FnvHash[K comparable](key K) uint64 {
	h := fnv.New64a()

	switch k := any(key).(type) {
	case string:
		h.Write([]byte(k))
	case []byte:
		h.Write(k)
	case int:
		h.Write(uint64Bytes(uint64(k)))
	case int8:
		h.Write(uint64Bytes(uint64(k)))
	case int16:
		h.Write(uint64Bytes(uint64(k)))
	case int32:
		h.Write(uint64Bytes(uint64(k)))
	case int64:
		h.Write(uint64Bytes(uint64(k)))
	case uint:
		h.Write(uint64Bytes(uint64(k)))
	case uint8:
		h.Write(uint64Bytes(uint64(k)))
	case uint16:
		h.Write(uint64Bytes(uint64(k)))
	case uint32:
		h.Write(uint64Bytes(uint64(k)))
	case uint64:
		h.Write(uint64Bytes(k))
	case float32:
		h.Write(uint64Bytes(uint64(math.Float64bits(float64(k)))))
	case float64:
		h.Write(uint64Bytes(math.Float64bits(k)))
	case bool:
		if k {
			h.Write([]byte{1})
		} else {
			h.Write([]byte{0})
		}
	default:
		h.Write(fmt.Appendf(nil, "%v", key))
	}

	return h.Sum64()
}

// uint64Bytes 将 uint64 转换为 8 字节的 big-endian 表示
func uint64Bytes(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}
