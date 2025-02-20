// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        (unknown)
// source: opencloud/services/eventhistory/v0/eventhistory.proto

package v0

import (
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2/options"
	v0 "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/eventhistory/v0"
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

// A request to retrieve events
type GetEventsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// the ids of the events we want to get
	Ids []string `protobuf:"bytes,1,rep,name=ids,proto3" json:"ids,omitempty"`
}

func (x *GetEventsRequest) Reset() {
	*x = GetEventsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_opencloud_services_eventhistory_v0_eventhistory_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetEventsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetEventsRequest) ProtoMessage() {}

func (x *GetEventsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_opencloud_services_eventhistory_v0_eventhistory_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetEventsRequest.ProtoReflect.Descriptor instead.
func (*GetEventsRequest) Descriptor() ([]byte, []int) {
	return file_opencloud_services_eventhistory_v0_eventhistory_proto_rawDescGZIP(), []int{0}
}

func (x *GetEventsRequest) GetIds() []string {
	if x != nil {
		return x.Ids
	}
	return nil
}

// A request to retrieve events belonging to a userID
type GetEventsForUserRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// the userID of the events we want to get
	UserID string `protobuf:"bytes,1,opt,name=userID,proto3" json:"userID,omitempty"`
}

func (x *GetEventsForUserRequest) Reset() {
	*x = GetEventsForUserRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_opencloud_services_eventhistory_v0_eventhistory_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetEventsForUserRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetEventsForUserRequest) ProtoMessage() {}

func (x *GetEventsForUserRequest) ProtoReflect() protoreflect.Message {
	mi := &file_opencloud_services_eventhistory_v0_eventhistory_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetEventsForUserRequest.ProtoReflect.Descriptor instead.
func (*GetEventsForUserRequest) Descriptor() ([]byte, []int) {
	return file_opencloud_services_eventhistory_v0_eventhistory_proto_rawDescGZIP(), []int{1}
}

func (x *GetEventsForUserRequest) GetUserID() string {
	if x != nil {
		return x.UserID
	}
	return ""
}

// The service response
type GetEventsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Events []*v0.Event `protobuf:"bytes,1,rep,name=events,proto3" json:"events,omitempty"`
}

func (x *GetEventsResponse) Reset() {
	*x = GetEventsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_opencloud_services_eventhistory_v0_eventhistory_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetEventsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetEventsResponse) ProtoMessage() {}

func (x *GetEventsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_opencloud_services_eventhistory_v0_eventhistory_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetEventsResponse.ProtoReflect.Descriptor instead.
func (*GetEventsResponse) Descriptor() ([]byte, []int) {
	return file_opencloud_services_eventhistory_v0_eventhistory_proto_rawDescGZIP(), []int{2}
}

func (x *GetEventsResponse) GetEvents() []*v0.Event {
	if x != nil {
		return x.Events
	}
	return nil
}

var File_opencloud_services_eventhistory_v0_eventhistory_proto protoreflect.FileDescriptor

var file_opencloud_services_eventhistory_v0_eventhistory_proto_rawDesc = []byte{
	0x0a, 0x35, 0x6f, 0x70, 0x65, 0x6e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2f, 0x73, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x73, 0x2f, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x68, 0x69, 0x73, 0x74, 0x6f, 0x72,
	0x79, 0x2f, 0x76, 0x30, 0x2f, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x68, 0x69, 0x73, 0x74, 0x6f, 0x72,
	0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x22, 0x6f, 0x70, 0x65, 0x6e, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x65, 0x76, 0x65, 0x6e,
	0x74, 0x68, 0x69, 0x73, 0x74, 0x6f, 0x72, 0x79, 0x2e, 0x76, 0x30, 0x1a, 0x35, 0x6f, 0x70, 0x65,
	0x6e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x73, 0x2f,
	0x65, 0x76, 0x65, 0x6e, 0x74, 0x68, 0x69, 0x73, 0x74, 0x6f, 0x72, 0x79, 0x2f, 0x76, 0x30, 0x2f,
	0x65, 0x76, 0x65, 0x6e, 0x74, 0x68, 0x69, 0x73, 0x74, 0x6f, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x2d, 0x67, 0x65, 0x6e, 0x2d, 0x6f,
	0x70, 0x65, 0x6e, 0x61, 0x70, 0x69, 0x76, 0x32, 0x2f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0x24, 0x0a, 0x10, 0x47, 0x65, 0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x69, 0x64, 0x73, 0x18, 0x01, 0x20,
	0x03, 0x28, 0x09, 0x52, 0x03, 0x69, 0x64, 0x73, 0x22, 0x31, 0x0a, 0x17, 0x47, 0x65, 0x74, 0x45,
	0x76, 0x65, 0x6e, 0x74, 0x73, 0x46, 0x6f, 0x72, 0x55, 0x73, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x44, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x44, 0x22, 0x56, 0x0a, 0x11, 0x47,
	0x65, 0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x41, 0x0a, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x29, 0x2e, 0x6f, 0x70, 0x65, 0x6e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2e, 0x6d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x73, 0x2e, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x68, 0x69, 0x73, 0x74, 0x6f,
	0x72, 0x79, 0x2e, 0x76, 0x30, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x52, 0x06, 0x65, 0x76, 0x65,
	0x6e, 0x74, 0x73, 0x32, 0x98, 0x02, 0x0a, 0x13, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x48, 0x69, 0x73,
	0x74, 0x6f, 0x72, 0x79, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x78, 0x0a, 0x09, 0x47,
	0x65, 0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x12, 0x34, 0x2e, 0x6f, 0x70, 0x65, 0x6e, 0x63,
	0x6c, 0x6f, 0x75, 0x64, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x65, 0x76,
	0x65, 0x6e, 0x74, 0x68, 0x69, 0x73, 0x74, 0x6f, 0x72, 0x79, 0x2e, 0x76, 0x30, 0x2e, 0x47, 0x65,
	0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x35,
	0x2e, 0x6f, 0x70, 0x65, 0x6e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x73, 0x2e, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x68, 0x69, 0x73, 0x74, 0x6f, 0x72, 0x79,
	0x2e, 0x76, 0x30, 0x2e, 0x47, 0x65, 0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x86, 0x01, 0x0a, 0x10, 0x47, 0x65, 0x74, 0x45, 0x76, 0x65,
	0x6e, 0x74, 0x73, 0x46, 0x6f, 0x72, 0x55, 0x73, 0x65, 0x72, 0x12, 0x3b, 0x2e, 0x6f, 0x70, 0x65,
	0x6e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e,
	0x65, 0x76, 0x65, 0x6e, 0x74, 0x68, 0x69, 0x73, 0x74, 0x6f, 0x72, 0x79, 0x2e, 0x76, 0x30, 0x2e,
	0x47, 0x65, 0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x46, 0x6f, 0x72, 0x55, 0x73, 0x65, 0x72,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x35, 0x2e, 0x6f, 0x70, 0x65, 0x6e, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x65, 0x76, 0x65,
	0x6e, 0x74, 0x68, 0x69, 0x73, 0x74, 0x6f, 0x72, 0x79, 0x2e, 0x76, 0x30, 0x2e, 0x47, 0x65, 0x74,
	0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x85,
	0x03, 0x5a, 0x51, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6f, 0x70,
	0x65, 0x6e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2d, 0x65, 0x75, 0x2f, 0x6f, 0x70, 0x65, 0x6e, 0x63,
	0x6c, 0x6f, 0x75, 0x64, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x67, 0x65, 0x6e, 0x2f, 0x67, 0x65,
	0x6e, 0x2f, 0x6f, 0x70, 0x65, 0x6e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2f, 0x73, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x73, 0x2f, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x68, 0x69, 0x73, 0x74, 0x6f, 0x72,
	0x79, 0x2f, 0x76, 0x30, 0x92, 0x41, 0xae, 0x02, 0x12, 0xbd, 0x01, 0x0a, 0x16, 0x4f, 0x70, 0x65,
	0x6e, 0x43, 0x6c, 0x6f, 0x75, 0x64, 0x20, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x68, 0x69, 0x73, 0x74,
	0x6f, 0x72, 0x79, 0x22, 0x51, 0x0a, 0x0e, 0x4f, 0x70, 0x65, 0x6e, 0x43, 0x6c, 0x6f, 0x75, 0x64,
	0x20, 0x47, 0x6d, 0x62, 0x48, 0x12, 0x29, 0x68, 0x74, 0x74, 0x70, 0x73, 0x3a, 0x2f, 0x2f, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6f, 0x70, 0x65, 0x6e, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x2d, 0x65, 0x75, 0x2f, 0x6f, 0x70, 0x65, 0x6e, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x1a, 0x14, 0x73, 0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x40, 0x6f, 0x70, 0x65, 0x6e, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x2e, 0x65, 0x75, 0x2a, 0x49, 0x0a, 0x0a, 0x41, 0x70, 0x61, 0x63, 0x68, 0x65,
	0x2d, 0x32, 0x2e, 0x30, 0x12, 0x3b, 0x68, 0x74, 0x74, 0x70, 0x73, 0x3a, 0x2f, 0x2f, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6f, 0x70, 0x65, 0x6e, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x2d, 0x65, 0x75, 0x2f, 0x6f, 0x70, 0x65, 0x6e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2f,
	0x62, 0x6c, 0x6f, 0x62, 0x2f, 0x6d, 0x61, 0x69, 0x6e, 0x2f, 0x4c, 0x49, 0x43, 0x45, 0x4e, 0x53,
	0x45, 0x32, 0x05, 0x31, 0x2e, 0x30, 0x2e, 0x30, 0x2a, 0x02, 0x01, 0x02, 0x32, 0x10, 0x61, 0x70,
	0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x6a, 0x73, 0x6f, 0x6e, 0x3a, 0x10,
	0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x6a, 0x73, 0x6f, 0x6e,
	0x72, 0x44, 0x0a, 0x10, 0x44, 0x65, 0x76, 0x65, 0x6c, 0x6f, 0x70, 0x65, 0x72, 0x20, 0x4d, 0x61,
	0x6e, 0x75, 0x61, 0x6c, 0x12, 0x30, 0x68, 0x74, 0x74, 0x70, 0x73, 0x3a, 0x2f, 0x2f, 0x64, 0x6f,
	0x63, 0x73, 0x2e, 0x6f, 0x70, 0x65, 0x6e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2e, 0x65, 0x75, 0x2f,
	0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2f, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x68, 0x69,
	0x73, 0x74, 0x6f, 0x72, 0x79, 0x2f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_opencloud_services_eventhistory_v0_eventhistory_proto_rawDescOnce sync.Once
	file_opencloud_services_eventhistory_v0_eventhistory_proto_rawDescData = file_opencloud_services_eventhistory_v0_eventhistory_proto_rawDesc
)

func file_opencloud_services_eventhistory_v0_eventhistory_proto_rawDescGZIP() []byte {
	file_opencloud_services_eventhistory_v0_eventhistory_proto_rawDescOnce.Do(func() {
		file_opencloud_services_eventhistory_v0_eventhistory_proto_rawDescData = protoimpl.X.CompressGZIP(file_opencloud_services_eventhistory_v0_eventhistory_proto_rawDescData)
	})
	return file_opencloud_services_eventhistory_v0_eventhistory_proto_rawDescData
}

var file_opencloud_services_eventhistory_v0_eventhistory_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_opencloud_services_eventhistory_v0_eventhistory_proto_goTypes = []interface{}{
	(*GetEventsRequest)(nil),        // 0: opencloud.services.eventhistory.v0.GetEventsRequest
	(*GetEventsForUserRequest)(nil), // 1: opencloud.services.eventhistory.v0.GetEventsForUserRequest
	(*GetEventsResponse)(nil),       // 2: opencloud.services.eventhistory.v0.GetEventsResponse
	(*v0.Event)(nil),                // 3: opencloud.messages.eventhistory.v0.Event
}
var file_opencloud_services_eventhistory_v0_eventhistory_proto_depIdxs = []int32{
	3, // 0: opencloud.services.eventhistory.v0.GetEventsResponse.events:type_name -> opencloud.messages.eventhistory.v0.Event
	0, // 1: opencloud.services.eventhistory.v0.EventHistoryService.GetEvents:input_type -> opencloud.services.eventhistory.v0.GetEventsRequest
	1, // 2: opencloud.services.eventhistory.v0.EventHistoryService.GetEventsForUser:input_type -> opencloud.services.eventhistory.v0.GetEventsForUserRequest
	2, // 3: opencloud.services.eventhistory.v0.EventHistoryService.GetEvents:output_type -> opencloud.services.eventhistory.v0.GetEventsResponse
	2, // 4: opencloud.services.eventhistory.v0.EventHistoryService.GetEventsForUser:output_type -> opencloud.services.eventhistory.v0.GetEventsResponse
	3, // [3:5] is the sub-list for method output_type
	1, // [1:3] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_opencloud_services_eventhistory_v0_eventhistory_proto_init() }
func file_opencloud_services_eventhistory_v0_eventhistory_proto_init() {
	if File_opencloud_services_eventhistory_v0_eventhistory_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_opencloud_services_eventhistory_v0_eventhistory_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetEventsRequest); i {
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
		file_opencloud_services_eventhistory_v0_eventhistory_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetEventsForUserRequest); i {
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
		file_opencloud_services_eventhistory_v0_eventhistory_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetEventsResponse); i {
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
			RawDescriptor: file_opencloud_services_eventhistory_v0_eventhistory_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_opencloud_services_eventhistory_v0_eventhistory_proto_goTypes,
		DependencyIndexes: file_opencloud_services_eventhistory_v0_eventhistory_proto_depIdxs,
		MessageInfos:      file_opencloud_services_eventhistory_v0_eventhistory_proto_msgTypes,
	}.Build()
	File_opencloud_services_eventhistory_v0_eventhistory_proto = out.File
	file_opencloud_services_eventhistory_v0_eventhistory_proto_rawDesc = nil
	file_opencloud_services_eventhistory_v0_eventhistory_proto_goTypes = nil
	file_opencloud_services_eventhistory_v0_eventhistory_proto_depIdxs = nil
}
