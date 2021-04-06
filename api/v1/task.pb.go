// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.15.7
// source: api/v1/task.proto

package task_v1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Task struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id      int32  `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Session string `protobuf:"bytes,2,opt,name=session,proto3" json:"session,omitempty"`
	Payload []byte `protobuf:"bytes,3,opt,name=payload,proto3" json:"payload,omitempty"`
}

func (x *Task) Reset() {
	*x = Task{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_task_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Task) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Task) ProtoMessage() {}

func (x *Task) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_task_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Task.ProtoReflect.Descriptor instead.
func (*Task) Descriptor() ([]byte, []int) {
	return file_api_v1_task_proto_rawDescGZIP(), []int{0}
}

func (x *Task) GetId() int32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Task) GetSession() string {
	if x != nil {
		return x.Session
	}
	return ""
}

func (x *Task) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

type Batch struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Tasks []*Task `protobuf:"bytes,1,rep,name=tasks,proto3" json:"tasks,omitempty"`
}

func (x *Batch) Reset() {
	*x = Batch{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_task_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Batch) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Batch) ProtoMessage() {}

func (x *Batch) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_task_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Batch.ProtoReflect.Descriptor instead.
func (*Batch) Descriptor() ([]byte, []int) {
	return file_api_v1_task_proto_rawDescGZIP(), []int{1}
}

func (x *Batch) GetTasks() []*Task {
	if x != nil {
		return x.Tasks
	}
	return nil
}

type SleepExample struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Sleepduration int32  `protobuf:"varint,1,opt,name=sleepduration,proto3" json:"sleepduration,omitempty"`
	Fakepayload   []byte `protobuf:"bytes,2,opt,name=fakepayload,proto3" json:"fakepayload,omitempty"`
}

func (x *SleepExample) Reset() {
	*x = SleepExample{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_task_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SleepExample) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SleepExample) ProtoMessage() {}

func (x *SleepExample) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_task_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SleepExample.ProtoReflect.Descriptor instead.
func (*SleepExample) Descriptor() ([]byte, []int) {
	return file_api_v1_task_proto_rawDescGZIP(), []int{2}
}

func (x *SleepExample) GetSleepduration() int32 {
	if x != nil {
		return x.Sleepduration
	}
	return 0
}

func (x *SleepExample) GetFakepayload() []byte {
	if x != nil {
		return x.Fakepayload
	}
	return nil
}

var File_api_v1_task_proto protoreflect.FileDescriptor

var file_api_v1_task_proto_rawDesc = []byte{
	0x0a, 0x11, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x2f, 0x74, 0x61, 0x73, 0x6b, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x07, 0x74, 0x61, 0x73, 0x6b, 0x2e, 0x76, 0x31, 0x22, 0x4a, 0x0a, 0x04,
	0x54, 0x61, 0x73, 0x6b, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05,
	0x52, 0x02, 0x69, 0x64, 0x12, 0x18, 0x0a, 0x07, 0x73, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x73, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x18,
	0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x22, 0x2c, 0x0a, 0x05, 0x42, 0x61, 0x74, 0x63,
	0x68, 0x12, 0x23, 0x0a, 0x05, 0x74, 0x61, 0x73, 0x6b, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x0d, 0x2e, 0x74, 0x61, 0x73, 0x6b, 0x2e, 0x76, 0x31, 0x2e, 0x54, 0x61, 0x73, 0x6b, 0x52,
	0x05, 0x74, 0x61, 0x73, 0x6b, 0x73, 0x22, 0x56, 0x0a, 0x0c, 0x53, 0x6c, 0x65, 0x65, 0x70, 0x45,
	0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x12, 0x24, 0x0a, 0x0d, 0x73, 0x6c, 0x65, 0x65, 0x70, 0x64,
	0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0d, 0x73,
	0x6c, 0x65, 0x65, 0x70, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x20, 0x0a, 0x0b,
	0x66, 0x61, 0x6b, 0x65, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x0b, 0x66, 0x61, 0x6b, 0x65, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x42, 0x2c,
	0x5a, 0x2a, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6c, 0x69, 0x6d,
	0x65, 0x2d, 0x6c, 0x61, 0x62, 0x73, 0x2f, 0x6d, 0x65, 0x74, 0x61, 0x6c, 0x63, 0x6f, 0x72, 0x65,
	0x2f, 0x61, 0x70, 0x69, 0x2f, 0x74, 0x61, 0x73, 0x6b, 0x5f, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_api_v1_task_proto_rawDescOnce sync.Once
	file_api_v1_task_proto_rawDescData = file_api_v1_task_proto_rawDesc
)

func file_api_v1_task_proto_rawDescGZIP() []byte {
	file_api_v1_task_proto_rawDescOnce.Do(func() {
		file_api_v1_task_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_v1_task_proto_rawDescData)
	})
	return file_api_v1_task_proto_rawDescData
}

var file_api_v1_task_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_api_v1_task_proto_goTypes = []interface{}{
	(*Task)(nil),         // 0: task.v1.Task
	(*Batch)(nil),        // 1: task.v1.Batch
	(*SleepExample)(nil), // 2: task.v1.SleepExample
}
var file_api_v1_task_proto_depIdxs = []int32{
	0, // 0: task.v1.Batch.tasks:type_name -> task.v1.Task
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_api_v1_task_proto_init() }
func file_api_v1_task_proto_init() {
	if File_api_v1_task_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_api_v1_task_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Task); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_v1_task_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Batch); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_v1_task_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SleepExample); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_api_v1_task_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_api_v1_task_proto_goTypes,
		DependencyIndexes: file_api_v1_task_proto_depIdxs,
		MessageInfos:      file_api_v1_task_proto_msgTypes,
	}.Build()
	File_api_v1_task_proto = out.File
	file_api_v1_task_proto_rawDesc = nil
	file_api_v1_task_proto_goTypes = nil
	file_api_v1_task_proto_depIdxs = nil
}