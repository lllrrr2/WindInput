package binformat

import (
	"encoding/binary"
	"fmt"
)

// 二进制词库文件格式常量

var byteOrder = binary.LittleEndian

// ---- pinyin.wdb ----

// DictMagic pinyin.wdb 文件魔数
var DictMagic = [4]byte{'W', 'D', 'I', 'C'}

const DictVersion uint32 = 3

// DictFileHeader wdb 文件头 (32 bytes)
type DictFileHeader struct {
	Magic     [4]byte // "WDIC"
	Version   uint32  // 2
	KeyCount  uint32  // 主索引的 key 数量
	IndexOff  uint32  // KeyIndex 区偏移
	DataOff   uint32  // EntryRecords 区偏移
	StrOff    uint32  // StringPool 区偏移
	AbbrevOff uint32  // Abbrev Section 偏移 (0 表示无)
	MetaOff   uint32  // Meta Section 偏移 (0 表示无)
}

const DictFileHeaderSize = 32

// DictKeyIndex 主索引条目 (12 bytes)
type DictKeyIndex struct {
	CodeOff  uint32 // code 字符串在 StringPool 的偏移
	CodeLen  uint16 // code 字符串长度
	EntryOff uint32 // 第一条 Entry 在 EntryRecords 区的偏移
	EntryLen uint16 // 候选数量
}

const DictKeyIndexSize = 12

// DictEntryRecord 词条记录 (14 bytes)
type DictEntryRecord struct {
	TextOff uint32 // Text 在 StringPool 的偏移
	TextLen uint16 // Text 字符串长度
	Weight  int32  // 权重
	Order   int32  // 全局顺序（词库文件中的出现位置，跨编码递增）
}

const DictEntryRecordSize = 14

// DictEntryRecordSizeV2 V2 版本的词条记录大小（无 Order 字段）
const DictEntryRecordSizeV2 = 10

// AbbrevHeader 简拼索引头 (16 bytes)
type AbbrevHeader struct {
	AbbrevCount  uint32
	AbbrevIdxOff uint32 // AbbrevIndex 区偏移
	Reserved1    uint32
	Reserved2    uint32
}

const AbbrevHeaderSize = 16

// AbbrevIndex 简拼索引条目 (12 bytes)
type AbbrevIndex struct {
	AbbrevOff uint32 // 简拼字符串在 StringPool 的偏移
	AbbrevLen uint16
	EntryOff  uint32 // 在 EntryRecords 中的偏移（复用主词条）
	EntryLen  uint16
}

const AbbrevIndexSize = 12

// ---- unigram.wdb ----

// UnigramMagic unigram.wdb 文件魔数
var UnigramMagic = [4]byte{'W', 'U', 'N', 'I'}

const UnigramVersion uint32 = 1

// UnigramFileHeader unigram.wdb 文件头 (24 bytes)
type UnigramFileHeader struct {
	Magic       [4]byte // "WUNI"
	Version     uint32  // 1
	KeyCount    uint32
	IndexOff    uint32 // KeyIndex 区偏移
	StrOff      uint32 // StringPool 区偏移
	MinFreqMark uint32 // 生成时使用的 min-freq 阈值（整数化），用于运行时校验 wdb 是否需要重建
}

const UnigramFileHeaderSize = 24

// UnigramKeyIndex unigram 索引条目 (12 bytes)
type UnigramKeyIndex struct {
	KeyOff   uint32 // key 字符串在 StringPool 的偏移
	KeyLen   uint16
	LogProb  float32 // 对数概率
	Reserved uint16
}

const UnigramKeyIndexSize = 12

// MetaHeaderSize Meta 段头部大小 (4 bytes: DataLen)
const MetaHeaderSize = 4

// Validate 验证 DictFileHeader（兼容 V1 和 V2）
func (h *DictFileHeader) Validate() error {
	if h.Magic != DictMagic {
		return fmt.Errorf("无效的词库文件魔数: %v", h.Magic)
	}
	if h.Version != DictVersion && h.Version != 2 && h.Version != 1 {
		return fmt.Errorf("不支持的词库版本: %d", h.Version)
	}
	return nil
}

// Validate 验证 UnigramFileHeader
func (h *UnigramFileHeader) Validate() error {
	if h.Magic != UnigramMagic {
		return fmt.Errorf("无效的 Unigram 文件魔数: %v", h.Magic)
	}
	if h.Version != UnigramVersion {
		return fmt.Errorf("不支持的 Unigram 版本: %d", h.Version)
	}
	return nil
}
