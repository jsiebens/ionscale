// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.17.3
// source: api/machines.proto

package api

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	_ "google.golang.org/protobuf/types/known/durationpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ListMachinesRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TailnetId uint64 `protobuf:"varint,1,opt,name=tailnet_id,json=tailnetId,proto3" json:"tailnet_id,omitempty"`
}

func (x *ListMachinesRequest) Reset() {
	*x = ListMachinesRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_machines_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListMachinesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListMachinesRequest) ProtoMessage() {}

func (x *ListMachinesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_machines_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListMachinesRequest.ProtoReflect.Descriptor instead.
func (*ListMachinesRequest) Descriptor() ([]byte, []int) {
	return file_api_machines_proto_rawDescGZIP(), []int{0}
}

func (x *ListMachinesRequest) GetTailnetId() uint64 {
	if x != nil {
		return x.TailnetId
	}
	return 0
}

type ListMachinesResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Machines []*Machine `protobuf:"bytes,1,rep,name=machines,proto3" json:"machines,omitempty"`
}

func (x *ListMachinesResponse) Reset() {
	*x = ListMachinesResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_machines_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListMachinesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListMachinesResponse) ProtoMessage() {}

func (x *ListMachinesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_machines_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListMachinesResponse.ProtoReflect.Descriptor instead.
func (*ListMachinesResponse) Descriptor() ([]byte, []int) {
	return file_api_machines_proto_rawDescGZIP(), []int{1}
}

func (x *ListMachinesResponse) GetMachines() []*Machine {
	if x != nil {
		return x.Machines
	}
	return nil
}

type DeleteMachineRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	MachineId uint64 `protobuf:"varint,1,opt,name=machine_id,json=machineId,proto3" json:"machine_id,omitempty"`
}

func (x *DeleteMachineRequest) Reset() {
	*x = DeleteMachineRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_machines_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeleteMachineRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteMachineRequest) ProtoMessage() {}

func (x *DeleteMachineRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_machines_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteMachineRequest.ProtoReflect.Descriptor instead.
func (*DeleteMachineRequest) Descriptor() ([]byte, []int) {
	return file_api_machines_proto_rawDescGZIP(), []int{2}
}

func (x *DeleteMachineRequest) GetMachineId() uint64 {
	if x != nil {
		return x.MachineId
	}
	return 0
}

type DeleteMachineResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *DeleteMachineResponse) Reset() {
	*x = DeleteMachineResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_machines_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeleteMachineResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteMachineResponse) ProtoMessage() {}

func (x *DeleteMachineResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_machines_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteMachineResponse.ProtoReflect.Descriptor instead.
func (*DeleteMachineResponse) Descriptor() ([]byte, []int) {
	return file_api_machines_proto_rawDescGZIP(), []int{3}
}

type Machine struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id        uint64                 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Name      string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Ipv4      string                 `protobuf:"bytes,3,opt,name=ipv4,proto3" json:"ipv4,omitempty"`
	Ipv6      string                 `protobuf:"bytes,4,opt,name=ipv6,proto3" json:"ipv6,omitempty"`
	Ephemeral bool                   `protobuf:"varint,5,opt,name=ephemeral,proto3" json:"ephemeral,omitempty"`
	LastSeen  *timestamppb.Timestamp `protobuf:"bytes,6,opt,name=last_seen,json=lastSeen,proto3" json:"last_seen,omitempty"`
	Connected bool                   `protobuf:"varint,7,opt,name=connected,proto3" json:"connected,omitempty"`
	Tailnet   *Ref                   `protobuf:"bytes,8,opt,name=tailnet,proto3" json:"tailnet,omitempty"`
	User      *Ref                   `protobuf:"bytes,9,opt,name=user,proto3" json:"user,omitempty"`
	Tags      []string               `protobuf:"bytes,10,rep,name=tags,proto3" json:"tags,omitempty"`
}

func (x *Machine) Reset() {
	*x = Machine{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_machines_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Machine) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Machine) ProtoMessage() {}

func (x *Machine) ProtoReflect() protoreflect.Message {
	mi := &file_api_machines_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Machine.ProtoReflect.Descriptor instead.
func (*Machine) Descriptor() ([]byte, []int) {
	return file_api_machines_proto_rawDescGZIP(), []int{4}
}

func (x *Machine) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Machine) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Machine) GetIpv4() string {
	if x != nil {
		return x.Ipv4
	}
	return ""
}

func (x *Machine) GetIpv6() string {
	if x != nil {
		return x.Ipv6
	}
	return ""
}

func (x *Machine) GetEphemeral() bool {
	if x != nil {
		return x.Ephemeral
	}
	return false
}

func (x *Machine) GetLastSeen() *timestamppb.Timestamp {
	if x != nil {
		return x.LastSeen
	}
	return nil
}

func (x *Machine) GetConnected() bool {
	if x != nil {
		return x.Connected
	}
	return false
}

func (x *Machine) GetTailnet() *Ref {
	if x != nil {
		return x.Tailnet
	}
	return nil
}

func (x *Machine) GetUser() *Ref {
	if x != nil {
		return x.User
	}
	return nil
}

func (x *Machine) GetTags() []string {
	if x != nil {
		return x.Tags
	}
	return nil
}

var File_api_machines_proto protoreflect.FileDescriptor

var file_api_machines_proto_rawDesc = []byte{
	0x0a, 0x12, 0x61, 0x70, 0x69, 0x2f, 0x6d, 0x61, 0x63, 0x68, 0x69, 0x6e, 0x65, 0x73, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x03, 0x61, 0x70, 0x69, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x75, 0x72, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x0d, 0x61, 0x70, 0x69, 0x2f,
	0x72, 0x65, 0x66, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x34, 0x0a, 0x13, 0x4c, 0x69, 0x73,
	0x74, 0x4d, 0x61, 0x63, 0x68, 0x69, 0x6e, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x12, 0x1d, 0x0a, 0x0a, 0x74, 0x61, 0x69, 0x6c, 0x6e, 0x65, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x04, 0x52, 0x09, 0x74, 0x61, 0x69, 0x6c, 0x6e, 0x65, 0x74, 0x49, 0x64, 0x22,
	0x40, 0x0a, 0x14, 0x4c, 0x69, 0x73, 0x74, 0x4d, 0x61, 0x63, 0x68, 0x69, 0x6e, 0x65, 0x73, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x28, 0x0a, 0x08, 0x6d, 0x61, 0x63, 0x68, 0x69,
	0x6e, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0c, 0x2e, 0x61, 0x70, 0x69, 0x2e,
	0x4d, 0x61, 0x63, 0x68, 0x69, 0x6e, 0x65, 0x52, 0x08, 0x6d, 0x61, 0x63, 0x68, 0x69, 0x6e, 0x65,
	0x73, 0x22, 0x35, 0x0a, 0x14, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x4d, 0x61, 0x63, 0x68, 0x69,
	0x6e, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x6d, 0x61, 0x63,
	0x68, 0x69, 0x6e, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x09, 0x6d,
	0x61, 0x63, 0x68, 0x69, 0x6e, 0x65, 0x49, 0x64, 0x22, 0x17, 0x0a, 0x15, 0x44, 0x65, 0x6c, 0x65,
	0x74, 0x65, 0x4d, 0x61, 0x63, 0x68, 0x69, 0x6e, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x22, 0xa0, 0x02, 0x0a, 0x07, 0x4d, 0x61, 0x63, 0x68, 0x69, 0x6e, 0x65, 0x12, 0x0e, 0x0a,
	0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12, 0x0a,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x12, 0x12, 0x0a, 0x04, 0x69, 0x70, 0x76, 0x34, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x69, 0x70, 0x76, 0x34, 0x12, 0x12, 0x0a, 0x04, 0x69, 0x70, 0x76, 0x36, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x69, 0x70, 0x76, 0x36, 0x12, 0x1c, 0x0a, 0x09, 0x65, 0x70, 0x68,
	0x65, 0x6d, 0x65, 0x72, 0x61, 0x6c, 0x18, 0x05, 0x20, 0x01, 0x28, 0x08, 0x52, 0x09, 0x65, 0x70,
	0x68, 0x65, 0x6d, 0x65, 0x72, 0x61, 0x6c, 0x12, 0x37, 0x0a, 0x09, 0x6c, 0x61, 0x73, 0x74, 0x5f,
	0x73, 0x65, 0x65, 0x6e, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d,
	0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x08, 0x6c, 0x61, 0x73, 0x74, 0x53, 0x65, 0x65, 0x6e,
	0x12, 0x1c, 0x0a, 0x09, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x65, 0x64, 0x18, 0x07, 0x20,
	0x01, 0x28, 0x08, 0x52, 0x09, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x65, 0x64, 0x12, 0x22,
	0x0a, 0x07, 0x74, 0x61, 0x69, 0x6c, 0x6e, 0x65, 0x74, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x08, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x52, 0x65, 0x66, 0x52, 0x07, 0x74, 0x61, 0x69, 0x6c, 0x6e,
	0x65, 0x74, 0x12, 0x1c, 0x0a, 0x04, 0x75, 0x73, 0x65, 0x72, 0x18, 0x09, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x08, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x52, 0x65, 0x66, 0x52, 0x04, 0x75, 0x73, 0x65, 0x72,
	0x12, 0x12, 0x0a, 0x04, 0x74, 0x61, 0x67, 0x73, 0x18, 0x0a, 0x20, 0x03, 0x28, 0x09, 0x52, 0x04,
	0x74, 0x61, 0x67, 0x73, 0x42, 0x2a, 0x5a, 0x28, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x6a, 0x73, 0x69, 0x65, 0x62, 0x65, 0x6e, 0x73, 0x2f, 0x69, 0x6f, 0x6e, 0x73,
	0x63, 0x61, 0x6c, 0x65, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x67, 0x65, 0x6e, 0x3b, 0x61, 0x70, 0x69,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_api_machines_proto_rawDescOnce sync.Once
	file_api_machines_proto_rawDescData = file_api_machines_proto_rawDesc
)

func file_api_machines_proto_rawDescGZIP() []byte {
	file_api_machines_proto_rawDescOnce.Do(func() {
		file_api_machines_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_machines_proto_rawDescData)
	})
	return file_api_machines_proto_rawDescData
}

var file_api_machines_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_api_machines_proto_goTypes = []interface{}{
	(*ListMachinesRequest)(nil),   // 0: api.ListMachinesRequest
	(*ListMachinesResponse)(nil),  // 1: api.ListMachinesResponse
	(*DeleteMachineRequest)(nil),  // 2: api.DeleteMachineRequest
	(*DeleteMachineResponse)(nil), // 3: api.DeleteMachineResponse
	(*Machine)(nil),               // 4: api.Machine
	(*timestamppb.Timestamp)(nil), // 5: google.protobuf.Timestamp
	(*Ref)(nil),                   // 6: api.Ref
}
var file_api_machines_proto_depIdxs = []int32{
	4, // 0: api.ListMachinesResponse.machines:type_name -> api.Machine
	5, // 1: api.Machine.last_seen:type_name -> google.protobuf.Timestamp
	6, // 2: api.Machine.tailnet:type_name -> api.Ref
	6, // 3: api.Machine.user:type_name -> api.Ref
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_api_machines_proto_init() }
func file_api_machines_proto_init() {
	if File_api_machines_proto != nil {
		return
	}
	file_api_ref_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_api_machines_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListMachinesRequest); i {
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
		file_api_machines_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListMachinesResponse); i {
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
		file_api_machines_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeleteMachineRequest); i {
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
		file_api_machines_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeleteMachineResponse); i {
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
		file_api_machines_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Machine); i {
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
			RawDescriptor: file_api_machines_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_api_machines_proto_goTypes,
		DependencyIndexes: file_api_machines_proto_depIdxs,
		MessageInfos:      file_api_machines_proto_msgTypes,
	}.Build()
	File_api_machines_proto = out.File
	file_api_machines_proto_rawDesc = nil
	file_api_machines_proto_goTypes = nil
	file_api_machines_proto_depIdxs = nil
}
