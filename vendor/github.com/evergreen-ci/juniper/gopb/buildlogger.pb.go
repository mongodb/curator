// Code generated by protoc-gen-go. DO NOT EDIT.
// source: buildlogger.proto

package gopb

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type LogStorage int32

const (
	LogStorage_LOG_STORAGE_S3     LogStorage = 0
	LogStorage_LOG_STORAGE_GRIDFS LogStorage = 1
	LogStorage_LOG_STORAGE_LOCAL  LogStorage = 2
)

var LogStorage_name = map[int32]string{
	0: "LOG_STORAGE_S3",
	1: "LOG_STORAGE_GRIDFS",
	2: "LOG_STORAGE_LOCAL",
}

var LogStorage_value = map[string]int32{
	"LOG_STORAGE_S3":     0,
	"LOG_STORAGE_GRIDFS": 1,
	"LOG_STORAGE_LOCAL":  2,
}

func (x LogStorage) String() string {
	return proto.EnumName(LogStorage_name, int32(x))
}

func (LogStorage) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_c4f5c52c3a3ee6d6, []int{0}
}

type LogFormat int32

const (
	LogFormat_LOG_FORMAT_UNKNOWN LogFormat = 0
	LogFormat_LOG_FORMAT_TEXT    LogFormat = 1
	LogFormat_LOG_FORMAT_JSON    LogFormat = 2
	LogFormat_LOG_FORMAT_BSON    LogFormat = 3
)

var LogFormat_name = map[int32]string{
	0: "LOG_FORMAT_UNKNOWN",
	1: "LOG_FORMAT_TEXT",
	2: "LOG_FORMAT_JSON",
	3: "LOG_FORMAT_BSON",
}

var LogFormat_value = map[string]int32{
	"LOG_FORMAT_UNKNOWN": 0,
	"LOG_FORMAT_TEXT":    1,
	"LOG_FORMAT_JSON":    2,
	"LOG_FORMAT_BSON":    3,
}

func (x LogFormat) String() string {
	return proto.EnumName(LogFormat_name, int32(x))
}

func (LogFormat) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_c4f5c52c3a3ee6d6, []int{1}
}

type LogData struct {
	Info                 *LogInfo   `protobuf:"bytes,1,opt,name=info,proto3" json:"info,omitempty"`
	Storage              LogStorage `protobuf:"varint,2,opt,name=storage,proto3,enum=cedar.LogStorage" json:"storage,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *LogData) Reset()         { *m = LogData{} }
func (m *LogData) String() string { return proto.CompactTextString(m) }
func (*LogData) ProtoMessage()    {}
func (*LogData) Descriptor() ([]byte, []int) {
	return fileDescriptor_c4f5c52c3a3ee6d6, []int{0}
}

func (m *LogData) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_LogData.Unmarshal(m, b)
}
func (m *LogData) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_LogData.Marshal(b, m, deterministic)
}
func (m *LogData) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LogData.Merge(m, src)
}
func (m *LogData) XXX_Size() int {
	return xxx_messageInfo_LogData.Size(m)
}
func (m *LogData) XXX_DiscardUnknown() {
	xxx_messageInfo_LogData.DiscardUnknown(m)
}

var xxx_messageInfo_LogData proto.InternalMessageInfo

func (m *LogData) GetInfo() *LogInfo {
	if m != nil {
		return m.Info
	}
	return nil
}

func (m *LogData) GetStorage() LogStorage {
	if m != nil {
		return m.Storage
	}
	return LogStorage_LOG_STORAGE_S3
}

type LogInfo struct {
	Project              string            `protobuf:"bytes,1,opt,name=project,proto3" json:"project,omitempty"`
	Version              string            `protobuf:"bytes,2,opt,name=version,proto3" json:"version,omitempty"`
	Variant              string            `protobuf:"bytes,3,opt,name=variant,proto3" json:"variant,omitempty"`
	TaskName             string            `protobuf:"bytes,4,opt,name=task_name,json=taskName,proto3" json:"task_name,omitempty"`
	TaskId               string            `protobuf:"bytes,5,opt,name=task_id,json=taskId,proto3" json:"task_id,omitempty"`
	Execution            int32             `protobuf:"varint,6,opt,name=execution,proto3" json:"execution,omitempty"`
	TestName             string            `protobuf:"bytes,7,opt,name=test_name,json=testName,proto3" json:"test_name,omitempty"`
	Trial                int32             `protobuf:"varint,8,opt,name=trial,proto3" json:"trial,omitempty"`
	ProcName             string            `protobuf:"bytes,9,opt,name=proc_name,json=procName,proto3" json:"proc_name,omitempty"`
	Format               LogFormat         `protobuf:"varint,10,opt,name=format,proto3,enum=cedar.LogFormat" json:"format,omitempty"`
	Tags                 []string          `protobuf:"bytes,11,rep,name=tags,proto3" json:"tags,omitempty"`
	Arguments            map[string]string `protobuf:"bytes,12,rep,name=arguments,proto3" json:"arguments,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Mainline             bool              `protobuf:"varint,13,opt,name=mainline,proto3" json:"mainline,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *LogInfo) Reset()         { *m = LogInfo{} }
func (m *LogInfo) String() string { return proto.CompactTextString(m) }
func (*LogInfo) ProtoMessage()    {}
func (*LogInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_c4f5c52c3a3ee6d6, []int{1}
}

func (m *LogInfo) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_LogInfo.Unmarshal(m, b)
}
func (m *LogInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_LogInfo.Marshal(b, m, deterministic)
}
func (m *LogInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LogInfo.Merge(m, src)
}
func (m *LogInfo) XXX_Size() int {
	return xxx_messageInfo_LogInfo.Size(m)
}
func (m *LogInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_LogInfo.DiscardUnknown(m)
}

var xxx_messageInfo_LogInfo proto.InternalMessageInfo

func (m *LogInfo) GetProject() string {
	if m != nil {
		return m.Project
	}
	return ""
}

func (m *LogInfo) GetVersion() string {
	if m != nil {
		return m.Version
	}
	return ""
}

func (m *LogInfo) GetVariant() string {
	if m != nil {
		return m.Variant
	}
	return ""
}

func (m *LogInfo) GetTaskName() string {
	if m != nil {
		return m.TaskName
	}
	return ""
}

func (m *LogInfo) GetTaskId() string {
	if m != nil {
		return m.TaskId
	}
	return ""
}

func (m *LogInfo) GetExecution() int32 {
	if m != nil {
		return m.Execution
	}
	return 0
}

func (m *LogInfo) GetTestName() string {
	if m != nil {
		return m.TestName
	}
	return ""
}

func (m *LogInfo) GetTrial() int32 {
	if m != nil {
		return m.Trial
	}
	return 0
}

func (m *LogInfo) GetProcName() string {
	if m != nil {
		return m.ProcName
	}
	return ""
}

func (m *LogInfo) GetFormat() LogFormat {
	if m != nil {
		return m.Format
	}
	return LogFormat_LOG_FORMAT_UNKNOWN
}

func (m *LogInfo) GetTags() []string {
	if m != nil {
		return m.Tags
	}
	return nil
}

func (m *LogInfo) GetArguments() map[string]string {
	if m != nil {
		return m.Arguments
	}
	return nil
}

func (m *LogInfo) GetMainline() bool {
	if m != nil {
		return m.Mainline
	}
	return false
}

type LogLines struct {
	LogId                string     `protobuf:"bytes,1,opt,name=log_id,json=logId,proto3" json:"log_id,omitempty"`
	Lines                []*LogLine `protobuf:"bytes,2,rep,name=lines,proto3" json:"lines,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *LogLines) Reset()         { *m = LogLines{} }
func (m *LogLines) String() string { return proto.CompactTextString(m) }
func (*LogLines) ProtoMessage()    {}
func (*LogLines) Descriptor() ([]byte, []int) {
	return fileDescriptor_c4f5c52c3a3ee6d6, []int{2}
}

func (m *LogLines) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_LogLines.Unmarshal(m, b)
}
func (m *LogLines) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_LogLines.Marshal(b, m, deterministic)
}
func (m *LogLines) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LogLines.Merge(m, src)
}
func (m *LogLines) XXX_Size() int {
	return xxx_messageInfo_LogLines.Size(m)
}
func (m *LogLines) XXX_DiscardUnknown() {
	xxx_messageInfo_LogLines.DiscardUnknown(m)
}

var xxx_messageInfo_LogLines proto.InternalMessageInfo

func (m *LogLines) GetLogId() string {
	if m != nil {
		return m.LogId
	}
	return ""
}

func (m *LogLines) GetLines() []*LogLine {
	if m != nil {
		return m.Lines
	}
	return nil
}

type LogLine struct {
	Priority             int32                `protobuf:"varint,1,opt,name=priority,proto3" json:"priority,omitempty"`
	Timestamp            *timestamp.Timestamp `protobuf:"bytes,2,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Data                 []byte               `protobuf:"bytes,3,opt,name=data,proto3" json:"data,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *LogLine) Reset()         { *m = LogLine{} }
func (m *LogLine) String() string { return proto.CompactTextString(m) }
func (*LogLine) ProtoMessage()    {}
func (*LogLine) Descriptor() ([]byte, []int) {
	return fileDescriptor_c4f5c52c3a3ee6d6, []int{3}
}

func (m *LogLine) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_LogLine.Unmarshal(m, b)
}
func (m *LogLine) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_LogLine.Marshal(b, m, deterministic)
}
func (m *LogLine) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LogLine.Merge(m, src)
}
func (m *LogLine) XXX_Size() int {
	return xxx_messageInfo_LogLine.Size(m)
}
func (m *LogLine) XXX_DiscardUnknown() {
	xxx_messageInfo_LogLine.DiscardUnknown(m)
}

var xxx_messageInfo_LogLine proto.InternalMessageInfo

func (m *LogLine) GetPriority() int32 {
	if m != nil {
		return m.Priority
	}
	return 0
}

func (m *LogLine) GetTimestamp() *timestamp.Timestamp {
	if m != nil {
		return m.Timestamp
	}
	return nil
}

func (m *LogLine) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

type LogEndInfo struct {
	LogId                string   `protobuf:"bytes,1,opt,name=log_id,json=logId,proto3" json:"log_id,omitempty"`
	ExitCode             int32    `protobuf:"varint,2,opt,name=exit_code,json=exitCode,proto3" json:"exit_code,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *LogEndInfo) Reset()         { *m = LogEndInfo{} }
func (m *LogEndInfo) String() string { return proto.CompactTextString(m) }
func (*LogEndInfo) ProtoMessage()    {}
func (*LogEndInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_c4f5c52c3a3ee6d6, []int{4}
}

func (m *LogEndInfo) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_LogEndInfo.Unmarshal(m, b)
}
func (m *LogEndInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_LogEndInfo.Marshal(b, m, deterministic)
}
func (m *LogEndInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LogEndInfo.Merge(m, src)
}
func (m *LogEndInfo) XXX_Size() int {
	return xxx_messageInfo_LogEndInfo.Size(m)
}
func (m *LogEndInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_LogEndInfo.DiscardUnknown(m)
}

var xxx_messageInfo_LogEndInfo proto.InternalMessageInfo

func (m *LogEndInfo) GetLogId() string {
	if m != nil {
		return m.LogId
	}
	return ""
}

func (m *LogEndInfo) GetExitCode() int32 {
	if m != nil {
		return m.ExitCode
	}
	return 0
}

type BuildloggerResponse struct {
	LogId                string   `protobuf:"bytes,1,opt,name=log_id,json=logId,proto3" json:"log_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BuildloggerResponse) Reset()         { *m = BuildloggerResponse{} }
func (m *BuildloggerResponse) String() string { return proto.CompactTextString(m) }
func (*BuildloggerResponse) ProtoMessage()    {}
func (*BuildloggerResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_c4f5c52c3a3ee6d6, []int{5}
}

func (m *BuildloggerResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BuildloggerResponse.Unmarshal(m, b)
}
func (m *BuildloggerResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BuildloggerResponse.Marshal(b, m, deterministic)
}
func (m *BuildloggerResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BuildloggerResponse.Merge(m, src)
}
func (m *BuildloggerResponse) XXX_Size() int {
	return xxx_messageInfo_BuildloggerResponse.Size(m)
}
func (m *BuildloggerResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_BuildloggerResponse.DiscardUnknown(m)
}

var xxx_messageInfo_BuildloggerResponse proto.InternalMessageInfo

func (m *BuildloggerResponse) GetLogId() string {
	if m != nil {
		return m.LogId
	}
	return ""
}

func init() {
	proto.RegisterEnum("cedar.LogStorage", LogStorage_name, LogStorage_value)
	proto.RegisterEnum("cedar.LogFormat", LogFormat_name, LogFormat_value)
	proto.RegisterType((*LogData)(nil), "cedar.LogData")
	proto.RegisterType((*LogInfo)(nil), "cedar.LogInfo")
	proto.RegisterMapType((map[string]string)(nil), "cedar.LogInfo.ArgumentsEntry")
	proto.RegisterType((*LogLines)(nil), "cedar.LogLines")
	proto.RegisterType((*LogLine)(nil), "cedar.LogLine")
	proto.RegisterType((*LogEndInfo)(nil), "cedar.LogEndInfo")
	proto.RegisterType((*BuildloggerResponse)(nil), "cedar.BuildloggerResponse")
}

func init() { proto.RegisterFile("buildlogger.proto", fileDescriptor_c4f5c52c3a3ee6d6) }

var fileDescriptor_c4f5c52c3a3ee6d6 = []byte{
	// 709 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x54, 0xdf, 0x6f, 0xda, 0x48,
	0x10, 0x8e, 0x01, 0x03, 0x1e, 0x72, 0xc4, 0xd9, 0x5c, 0xee, 0x2c, 0x72, 0xa7, 0x43, 0xe8, 0x1e,
	0xac, 0xdc, 0x89, 0x48, 0xe4, 0xe1, 0x72, 0x77, 0xad, 0x5a, 0x42, 0x08, 0xa2, 0x75, 0x41, 0x5a,
	0xa8, 0x5a, 0xe5, 0x05, 0x2d, 0x78, 0xb1, 0xdc, 0xd8, 0x5e, 0x6b, 0x77, 0x89, 0x9a, 0xc7, 0xfe,
	0x87, 0xfd, 0x93, 0xaa, 0x5d, 0x9b, 0x1f, 0x89, 0x9a, 0x3c, 0xf4, 0x6d, 0xe7, 0xfb, 0xe6, 0x9b,
	0x1d, 0xcf, 0x7c, 0x5e, 0x38, 0x9c, 0xaf, 0xc2, 0xc8, 0x8f, 0x58, 0x10, 0x50, 0xde, 0x4e, 0x39,
	0x93, 0x0c, 0x99, 0x0b, 0xea, 0x13, 0xde, 0xf8, 0x23, 0x60, 0x2c, 0x88, 0xe8, 0x99, 0x06, 0xe7,
	0xab, 0xe5, 0x99, 0x0c, 0x63, 0x2a, 0x24, 0x89, 0xd3, 0x2c, 0xaf, 0x75, 0x03, 0x15, 0x8f, 0x05,
	0x57, 0x44, 0x12, 0xd4, 0x82, 0x52, 0x98, 0x2c, 0x99, 0x63, 0x34, 0x0d, 0xb7, 0xd6, 0xa9, 0xb7,
	0x75, 0x85, 0xb6, 0xc7, 0x82, 0x61, 0xb2, 0x64, 0x58, 0x73, 0xe8, 0x2f, 0xa8, 0x08, 0xc9, 0x38,
	0x09, 0xa8, 0x53, 0x68, 0x1a, 0x6e, 0xbd, 0x73, 0xb8, 0x4d, 0x9b, 0x64, 0x04, 0x5e, 0x67, 0xb4,
	0xbe, 0x16, 0x75, 0x71, 0x25, 0x47, 0x0e, 0x54, 0x52, 0xce, 0x3e, 0xd1, 0x85, 0xd4, 0xf5, 0x2d,
	0xbc, 0x0e, 0x15, 0x73, 0x47, 0xb9, 0x08, 0x59, 0xa2, 0x4b, 0x5a, 0x78, 0x1d, 0x6a, 0x86, 0xf0,
	0x90, 0x24, 0xd2, 0x29, 0xe6, 0x4c, 0x16, 0xa2, 0x13, 0xb0, 0x24, 0x11, 0xb7, 0xb3, 0x84, 0xc4,
	0xd4, 0x29, 0x69, 0xae, 0xaa, 0x80, 0x11, 0x89, 0x29, 0xfa, 0x15, 0x2a, 0x9a, 0x0c, 0x7d, 0xc7,
	0xd4, 0x54, 0x59, 0x85, 0x43, 0x1f, 0xfd, 0x06, 0x16, 0xfd, 0x4c, 0x17, 0x2b, 0xa9, 0xee, 0x2a,
	0x37, 0x0d, 0xd7, 0xc4, 0x5b, 0x40, 0xd7, 0xa4, 0x42, 0x66, 0x35, 0x2b, 0x79, 0x4d, 0x2a, 0xa4,
	0xae, 0xf9, 0x33, 0x98, 0x92, 0x87, 0x24, 0x72, 0xaa, 0x5a, 0x96, 0x05, 0x4a, 0x92, 0x72, 0xb6,
	0xc8, 0x24, 0x56, 0x26, 0x51, 0x80, 0x96, 0xb8, 0x50, 0x5e, 0x32, 0x1e, 0x13, 0xe9, 0x80, 0x9e,
	0x94, 0xbd, 0x9d, 0xd4, 0xb5, 0xc6, 0x71, 0xce, 0x23, 0x04, 0x25, 0x49, 0x02, 0xe1, 0xd4, 0x9a,
	0x45, 0xd7, 0xc2, 0xfa, 0x8c, 0xfe, 0x07, 0x8b, 0xf0, 0x60, 0x15, 0xd3, 0x44, 0x0a, 0x67, 0xbf,
	0x59, 0x74, 0x6b, 0x9d, 0xdf, 0x1f, 0x6e, 0xa4, 0xdd, 0x5d, 0xf3, 0xfd, 0x44, 0xf2, 0x7b, 0xbc,
	0xcd, 0x47, 0x0d, 0xa8, 0xc6, 0x24, 0x4c, 0xa2, 0x30, 0xa1, 0xce, 0x4f, 0x4d, 0xc3, 0xad, 0xe2,
	0x4d, 0xdc, 0x78, 0x01, 0xf5, 0x87, 0x42, 0x64, 0x43, 0xf1, 0x96, 0xde, 0xe7, 0x6b, 0x51, 0x47,
	0xf5, 0xb5, 0x77, 0x24, 0x5a, 0xd1, 0x7c, 0x21, 0x59, 0xf0, 0x5f, 0xe1, 0xc2, 0x68, 0x0d, 0xa0,
	0xea, 0xb1, 0xc0, 0x0b, 0x13, 0x2a, 0xd0, 0x31, 0x94, 0x23, 0x16, 0xa8, 0x31, 0x67, 0x52, 0x33,
	0x62, 0xc1, 0xd0, 0x47, 0x7f, 0x82, 0xa9, 0x2e, 0x12, 0x4e, 0x41, 0x77, 0xbd, 0xe3, 0x23, 0x25,
	0xc3, 0x19, 0xd9, 0x12, 0xda, 0x1a, 0x0a, 0x51, 0xdd, 0xa6, 0x3c, 0x64, 0x3c, 0x94, 0x59, 0x13,
	0x26, 0xde, 0xc4, 0xe8, 0x02, 0xac, 0x8d, 0x63, 0x75, 0x37, 0xb5, 0x4e, 0xa3, 0x9d, 0x79, 0xba,
	0xbd, 0xf6, 0x74, 0x7b, 0xba, 0xce, 0xc0, 0xdb, 0x64, 0x35, 0x54, 0x9f, 0x48, 0xa2, 0x9d, 0xb3,
	0x8f, 0xf5, 0xb9, 0xf5, 0x1a, 0xc0, 0x63, 0x41, 0x3f, 0xf1, 0xb5, 0x25, 0x9f, 0xe8, 0xff, 0x44,
	0xb9, 0x24, 0x94, 0xb3, 0x05, 0xf3, 0xb3, 0x01, 0x98, 0xb8, 0xaa, 0x80, 0x1e, 0xf3, 0x69, 0xeb,
	0x6f, 0x38, 0xba, 0xdc, 0xfe, 0x6b, 0x98, 0x8a, 0x94, 0x25, 0x82, 0x3e, 0x51, 0xea, 0x74, 0xac,
	0xef, 0xcb, 0xff, 0x0b, 0x84, 0xa0, 0xee, 0x8d, 0x07, 0xb3, 0xc9, 0x74, 0x8c, 0xbb, 0x83, 0xfe,
	0x6c, 0x72, 0x6e, 0xef, 0xa1, 0x5f, 0x00, 0xed, 0x62, 0x03, 0x3c, 0xbc, 0xba, 0x9e, 0xd8, 0x06,
	0x3a, 0x86, 0xc3, 0x5d, 0xdc, 0x1b, 0xf7, 0xba, 0x9e, 0x5d, 0x38, 0x9d, 0x83, 0xb5, 0xb1, 0xcf,
	0x5a, 0x7b, 0x3d, 0xc6, 0xef, 0xba, 0xd3, 0xd9, 0xfb, 0xd1, 0xdb, 0xd1, 0xf8, 0xc3, 0xc8, 0xde,
	0x43, 0x47, 0x70, 0xb0, 0x83, 0x4f, 0xfb, 0x1f, 0xa7, 0xb6, 0xf1, 0x08, 0x7c, 0x33, 0x19, 0x8f,
	0xec, 0xc2, 0x23, 0xf0, 0x52, 0x81, 0xc5, 0xce, 0x97, 0x02, 0xd4, 0x76, 0xbe, 0x11, 0xfd, 0x03,
	0x56, 0x8f, 0x53, 0x22, 0xa9, 0xc7, 0x02, 0xb4, 0xb3, 0x4d, 0xf5, 0x66, 0x34, 0x1a, 0x79, 0xfc,
	0xbd, 0xa1, 0xbc, 0x84, 0x7a, 0x37, 0x4d, 0x69, 0xe2, 0x6f, 0x1c, 0x73, 0xf0, 0xd0, 0x0b, 0xe2,
	0x59, 0xf9, 0x2b, 0xa8, 0x4f, 0x24, 0xa7, 0x24, 0xfe, 0x21, 0xb9, 0x6b, 0xa0, 0x7f, 0xa1, 0xda,
	0x8b, 0x98, 0xd0, 0x7d, 0xef, 0x3c, 0x53, 0xf9, 0xfa, 0x9f, 0x13, 0x5f, 0x96, 0x6f, 0x4a, 0x01,
	0x4b, 0xe7, 0xf3, 0xb2, 0xf6, 0xd8, 0xf9, 0xb7, 0x00, 0x00, 0x00, 0xff, 0xff, 0x02, 0x67, 0x30,
	0xa4, 0x61, 0x05, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// BuildloggerClient is the client API for Buildlogger service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type BuildloggerClient interface {
	CreateLog(ctx context.Context, in *LogData, opts ...grpc.CallOption) (*BuildloggerResponse, error)
	AppendLogLines(ctx context.Context, in *LogLines, opts ...grpc.CallOption) (*BuildloggerResponse, error)
	StreamLogLines(ctx context.Context, opts ...grpc.CallOption) (Buildlogger_StreamLogLinesClient, error)
	CloseLog(ctx context.Context, in *LogEndInfo, opts ...grpc.CallOption) (*BuildloggerResponse, error)
}

type buildloggerClient struct {
	cc *grpc.ClientConn
}

func NewBuildloggerClient(cc *grpc.ClientConn) BuildloggerClient {
	return &buildloggerClient{cc}
}

func (c *buildloggerClient) CreateLog(ctx context.Context, in *LogData, opts ...grpc.CallOption) (*BuildloggerResponse, error) {
	out := new(BuildloggerResponse)
	err := c.cc.Invoke(ctx, "/cedar.Buildlogger/CreateLog", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *buildloggerClient) AppendLogLines(ctx context.Context, in *LogLines, opts ...grpc.CallOption) (*BuildloggerResponse, error) {
	out := new(BuildloggerResponse)
	err := c.cc.Invoke(ctx, "/cedar.Buildlogger/AppendLogLines", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *buildloggerClient) StreamLogLines(ctx context.Context, opts ...grpc.CallOption) (Buildlogger_StreamLogLinesClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Buildlogger_serviceDesc.Streams[0], "/cedar.Buildlogger/StreamLogLines", opts...)
	if err != nil {
		return nil, err
	}
	x := &buildloggerStreamLogLinesClient{stream}
	return x, nil
}

type Buildlogger_StreamLogLinesClient interface {
	Send(*LogLines) error
	CloseAndRecv() (*BuildloggerResponse, error)
	grpc.ClientStream
}

type buildloggerStreamLogLinesClient struct {
	grpc.ClientStream
}

func (x *buildloggerStreamLogLinesClient) Send(m *LogLines) error {
	return x.ClientStream.SendMsg(m)
}

func (x *buildloggerStreamLogLinesClient) CloseAndRecv() (*BuildloggerResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(BuildloggerResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *buildloggerClient) CloseLog(ctx context.Context, in *LogEndInfo, opts ...grpc.CallOption) (*BuildloggerResponse, error) {
	out := new(BuildloggerResponse)
	err := c.cc.Invoke(ctx, "/cedar.Buildlogger/CloseLog", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BuildloggerServer is the server API for Buildlogger service.
type BuildloggerServer interface {
	CreateLog(context.Context, *LogData) (*BuildloggerResponse, error)
	AppendLogLines(context.Context, *LogLines) (*BuildloggerResponse, error)
	StreamLogLines(Buildlogger_StreamLogLinesServer) error
	CloseLog(context.Context, *LogEndInfo) (*BuildloggerResponse, error)
}

// UnimplementedBuildloggerServer can be embedded to have forward compatible implementations.
type UnimplementedBuildloggerServer struct {
}

func (*UnimplementedBuildloggerServer) CreateLog(ctx context.Context, req *LogData) (*BuildloggerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateLog not implemented")
}
func (*UnimplementedBuildloggerServer) AppendLogLines(ctx context.Context, req *LogLines) (*BuildloggerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AppendLogLines not implemented")
}
func (*UnimplementedBuildloggerServer) StreamLogLines(srv Buildlogger_StreamLogLinesServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamLogLines not implemented")
}
func (*UnimplementedBuildloggerServer) CloseLog(ctx context.Context, req *LogEndInfo) (*BuildloggerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CloseLog not implemented")
}

func RegisterBuildloggerServer(s *grpc.Server, srv BuildloggerServer) {
	s.RegisterService(&_Buildlogger_serviceDesc, srv)
}

func _Buildlogger_CreateLog_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LogData)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BuildloggerServer).CreateLog(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cedar.Buildlogger/CreateLog",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BuildloggerServer).CreateLog(ctx, req.(*LogData))
	}
	return interceptor(ctx, in, info, handler)
}

func _Buildlogger_AppendLogLines_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LogLines)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BuildloggerServer).AppendLogLines(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cedar.Buildlogger/AppendLogLines",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BuildloggerServer).AppendLogLines(ctx, req.(*LogLines))
	}
	return interceptor(ctx, in, info, handler)
}

func _Buildlogger_StreamLogLines_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(BuildloggerServer).StreamLogLines(&buildloggerStreamLogLinesServer{stream})
}

type Buildlogger_StreamLogLinesServer interface {
	SendAndClose(*BuildloggerResponse) error
	Recv() (*LogLines, error)
	grpc.ServerStream
}

type buildloggerStreamLogLinesServer struct {
	grpc.ServerStream
}

func (x *buildloggerStreamLogLinesServer) SendAndClose(m *BuildloggerResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *buildloggerStreamLogLinesServer) Recv() (*LogLines, error) {
	m := new(LogLines)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _Buildlogger_CloseLog_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LogEndInfo)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BuildloggerServer).CloseLog(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cedar.Buildlogger/CloseLog",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BuildloggerServer).CloseLog(ctx, req.(*LogEndInfo))
	}
	return interceptor(ctx, in, info, handler)
}

var _Buildlogger_serviceDesc = grpc.ServiceDesc{
	ServiceName: "cedar.Buildlogger",
	HandlerType: (*BuildloggerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateLog",
			Handler:    _Buildlogger_CreateLog_Handler,
		},
		{
			MethodName: "AppendLogLines",
			Handler:    _Buildlogger_AppendLogLines_Handler,
		},
		{
			MethodName: "CloseLog",
			Handler:    _Buildlogger_CloseLog_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamLogLines",
			Handler:       _Buildlogger_StreamLogLines_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "buildlogger.proto",
}