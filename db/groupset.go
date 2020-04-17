package db

import (
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func newGroupSet(organisationGroups []string, serviceIDToGroups map[string][]string) *groupSet {
	gs := &groupSet{}
	gs.AddToGroups(organisationGroups...)
	if serviceIDToGroups != nil {
		for serviceID := range serviceIDToGroups {
			gs.AddToServiceGroups(serviceID, serviceIDToGroups[serviceID]...)
		}
	}
	return gs
}

// A groupSet maps both Organisation level and Service-level groups into a single DynamoDB set.
type groupSet struct {
	m                  sync.Mutex
	organisationGroups map[string]struct{}
	serviceIDToGroups  map[string]map[string]struct{}
}

// OrganisationGroups gets the list of Organisation level group names.
func (gs *groupSet) OrganisationGroups() (groups []string) {
	if gs.organisationGroups == nil {
		return
	}
	for g := range gs.organisationGroups {
		g := g
		groups = append(groups, g)
	}
	return
}

// ServiceGroups gets a map of ServiceIDs and group names.
func (gs *groupSet) ServiceGroups() (groups map[string][]string) {
	if gs.serviceIDToGroups == nil {
		return
	}
	groups = make(map[string][]string)
	for serviceID, setOfGroups := range gs.serviceIDToGroups {
		for g := range setOfGroups {
			g := g
			groups[serviceID] = append(groups[serviceID], g)
		}
	}
	return groups
}

// AddToGroups assigns membership of the Organisation groups.
func (gs *groupSet) AddToGroups(groups ...string) {
	if len(groups) == 0 {
		return
	}
	gs.m.Lock()
	defer gs.m.Unlock()
	if gs.organisationGroups == nil {
		gs.organisationGroups = make(map[string]struct{}, len(groups))
	}
	for i := 0; i < len(groups); i++ {
		gs.organisationGroups[groups[i]] = struct{}{}
	}
}

// AddToServiceGroups assigns membership of the Service groups.
func (gs *groupSet) AddToServiceGroups(serviceID string, groups ...string) {
	if len(groups) == 0 {
		return
	}
	gs.m.Lock()
	defer gs.m.Unlock()
	if gs.serviceIDToGroups == nil {
		gs.serviceIDToGroups = make(map[string]map[string]struct{})
	}
	sgs := gs.serviceIDToGroups[serviceID]
	if sgs == nil {
		sgs = make(map[string]struct{})
	}
	for i := 0; i < len(groups); i++ {
		sgs[groups[i]] = struct{}{}
	}
	gs.serviceIDToGroups[serviceID] = sgs
}

func (gs *groupSet) UnmarshalDynamoDBAttributeValue(av *dynamodb.AttributeValue) error {
	gs.organisationGroups = make(map[string]struct{})
	gs.serviceIDToGroups = make(map[string]map[string]struct{})
	if av.NULL != nil && *av.NULL == true {
		return nil
	}
	for i := 0; i < len(av.SS); i++ {
		g := av.SS[i]
		if g == nil {
			continue
		}
		parts := strings.SplitN(*g, "/", 3)
		if len(parts) < 2 {
			return fmt.Errorf("groupSet: cannot unmarshal string value %q into a group", *g)
		}
		switch parts[0] {
		case "organisationGroup":
			gs.AddToGroups(parts[1])
		case "serviceGroup":
			gs.AddToServiceGroups(parts[1], parts[2])
		}
	}
	return nil
}

func (gs *groupSet) MarshalDynamoDBAttributeValue(av *dynamodb.AttributeValue) error {
	var ss []string
	if gs.organisationGroups != nil {
		for g := range gs.organisationGroups {
			ss = append(ss, "organisationGroup/"+g)
		}
	}
	if gs.serviceIDToGroups != nil {
		for serviceID, groupNames := range gs.serviceIDToGroups {
			for g := range groupNames {
				ss = append(ss, "serviceGroup/"+serviceID+"/"+g)
			}
		}
	}
	av.SetSS(aws.StringSlice(ss))
	return nil
}
