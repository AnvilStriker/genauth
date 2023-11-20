package main

import (
	"encoding/json"
)

type (
	NSName       string
	AppName      string
	KSAName      string
	GSAName      string
	IAMRole      string
	RsrcKind     string
	RsrcOwnerKey string
	RsrcName     string
	RsrcFullName string
	OperName     string
)

//
// Input Types
//

type Apps map[NSName][]AppName

type Resources map[RsrcKind]map[RsrcOwnerKey][]RsrcName

// type Permissions: see permissions.go; may ultimately be loaded separately from usage

type Usage map[AppName]map[RsrcKind]map[OperName][]RsrcName

type ResourceUsage struct {
	Resources   Resources
	Permissions Permissions
	Usage       Usage
}

//
// Computed Types
//

type RoleBindingCore struct {
	IAMRole IAMRole   `json:"role"`
	Members []GSAName `json:"members"`
}

type RoleBinding struct {
	RoleBindingCore
	membersMap map[GSAName]bool
}

func (rb *RoleBinding) Add(gsaName GSAName) {
	rb.membersMap[gsaName] = true
}

func (rb *RoleBinding) MarshalJSON() ([]byte, error) {
	for m := range rb.membersMap {
		rb.Members = append(rb.Members, "serviceAccount:"+m)
	}
	return json.Marshal(&rb.RoleBindingCore)
}

type RoleBindingMap map[IAMRole]*RoleBinding

type ResourcePolicyCore struct {
	RsrcFullName RsrcFullName   `json:"resource"`
	RoleBindings []*RoleBinding `json:"bindings"`
}

type ResourcePolicy struct {
	ResourcePolicyCore
	roleBindingsMap RoleBindingMap
}

func (rp *ResourcePolicy) Add(roles []IAMRole, gsaName GSAName) {
	for _, role := range roles {
		rb, ok := rp.roleBindingsMap[role]
		if !ok {
			rb = &RoleBinding{
				RoleBindingCore: RoleBindingCore{
					IAMRole: role,
				},
				membersMap: make(map[GSAName]bool),
			}
			rp.roleBindingsMap[role] = rb
		}
		rb.Add(gsaName)
	}
}

func (rp *ResourcePolicy) MarshalJSON() ([]byte, error) {
	for _, rb := range rp.roleBindingsMap {
		rp.RoleBindings = append(rp.RoleBindings, rb)
	}
	return json.Marshal(&rp.ResourcePolicyCore)
}

type ResourcePolicyMap map[RsrcFullName]*ResourcePolicy

func (rpm ResourcePolicyMap) Add(rsrcFullName RsrcFullName, roles []IAMRole, gsaName GSAName) {
	rp, ok := rpm[rsrcFullName]
	if !ok {
		rp = &ResourcePolicy{
			ResourcePolicyCore: ResourcePolicyCore{
				RsrcFullName: rsrcFullName,
			},
			roleBindingsMap: make(RoleBindingMap),
		}
		rpm[rsrcFullName] = rp
	}
	rp.Add(roles, gsaName)
}
