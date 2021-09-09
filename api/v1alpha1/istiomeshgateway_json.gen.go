// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: api/v1alpha1/istiomeshgateway.proto

package v1alpha1

import (
	bytes "bytes"
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	github_com_gogo_protobuf_jsonpb "github.com/gogo/protobuf/jsonpb"
	proto "github.com/gogo/protobuf/proto"
	_ "github.com/gogo/protobuf/types"
	_ "istio.io/gogo-genproto/googleapis/google/api"
	_ "k8s.io/api/core/v1"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// MarshalJSON is a custom marshaler for IstioMeshGatewaySpec
func (this *IstioMeshGatewaySpec) MarshalJSON() ([]byte, error) {
	str, err := IstiomeshgatewayMarshaler.MarshalToString(this)
	return []byte(str), err
}

// UnmarshalJSON is a custom unmarshaler for IstioMeshGatewaySpec
func (this *IstioMeshGatewaySpec) UnmarshalJSON(b []byte) error {
	return IstiomeshgatewayUnmarshaler.Unmarshal(bytes.NewReader(b), this)
}

// MarshalJSON is a custom marshaler for Properties
func (this *Properties) MarshalJSON() ([]byte, error) {
	str, err := IstiomeshgatewayMarshaler.MarshalToString(this)
	return []byte(str), err
}

// UnmarshalJSON is a custom unmarshaler for Properties
func (this *Properties) UnmarshalJSON(b []byte) error {
	return IstiomeshgatewayUnmarshaler.Unmarshal(bytes.NewReader(b), this)
}

// MarshalJSON is a custom marshaler for IstioMeshGatewayStatus
func (this *IstioMeshGatewayStatus) MarshalJSON() ([]byte, error) {
	str, err := IstiomeshgatewayMarshaler.MarshalToString(this)
	return []byte(str), err
}

// UnmarshalJSON is a custom unmarshaler for IstioMeshGatewayStatus
func (this *IstioMeshGatewayStatus) UnmarshalJSON(b []byte) error {
	return IstiomeshgatewayUnmarshaler.Unmarshal(bytes.NewReader(b), this)
}

var (
	IstiomeshgatewayMarshaler   = &github_com_gogo_protobuf_jsonpb.Marshaler{Int64Uint64asIntegers: true}
	IstiomeshgatewayUnmarshaler = &github_com_gogo_protobuf_jsonpb.Unmarshaler{AllowUnknownFields: true}
)
