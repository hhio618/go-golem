package props

import (
	"fmt"
	"strings"

	"github.com/fatih/structs"
)

type DemandBuilder struct {
	properties  map[string]interface{}
	constraints []string
}

/*Builds a dictionary of properties and constraints from high-level models.

The dictionary represents a Demand object, which is later matched by the new Golem's
market implementation against Offers coming from providers to find those providers
who can satisfy the requestor's demand.

example usage:

```python
>>> import yapapi
>>> from yapapi import properties as yp
>>> from yapapi.props.builder import DemandBuilder
>>> from datetime import datetime, timezone
>>> builder = DemandBuilder()
>>> builder.add(yp.NodeInfo(name="a node", subnet_tag="testnet"))
>>> builder.add(yp.Activity(expiration=datetime.now(timezone.utc)))
>>> builder.__repr__
>>> print(builder)
{'properties':
	{'golem.node.id.name': 'a node',
	 'golem.node.debug.subnet': 'testnet',
	 'golem.srv.comp.expiration': 1601655628772},
 'constraints': []}
```
*/
func NewDemandBuilder() *DemandBuilder {
	return &DemandBuilder{
		constraints: make([]string, 0),
		properties:  make(map[string]interface{}),
	}
}

func (db *DemandBuilder) String() string {
	return fmt.Sprintf("properties: %v, properties: %v", db.properties, db.constraints)
}

func (db *DemandBuilder) Properties() map[string]interface{} {
	return db.properties
}

func (db *DemandBuilder) Constraints() string {
	cList := db.constraints
	var cValue string
	if len(cList) == 0 {
		cValue = "()"
	} else if len(cList) == 1 {
		cValue = cList[0]
	} else {
		rules := strings.Join(cList, "\n\t")
		cValue = fmt.Sprintf("(&%v)", rules)
	}
	return cValue
}

func (db *DemandBuilder) Ensure(constraint string) {
	db.constraints = append(db.constraints, constraint)
}

func (db *DemandBuilder) Add(m Model) {
	kv := m.Keys()
	base := structs.Map(m)
	for name := range kv {
		propId := kv[name]
		value := base[name]
		if value == nil {
			continue
		}
		db.properties[propId] = value
	}
}

// TODO: fix this after api fixes.
// func (db *DemandBuilder) Subscribe(market) (Subscription){
// 	return market.Subscribe(db.properties, db.constraints)
// }
